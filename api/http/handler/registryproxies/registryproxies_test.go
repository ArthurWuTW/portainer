package registryproxies

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/crypto"
	"github.com/portainer/portainer/api/datastore"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/internal/testhelpers"
	"github.com/portainer/portainer/pkg/fips"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRegistryURL(t *testing.T) {
	is := assert.New(t)

	host, err := normalizeRegistryURL("registry.local:5000")
	is.NoError(err)
	is.Equal("registry.local:5000", host)

	host, err = normalizeRegistryURL("http://registry.local:5000")
	is.NoError(err)
	is.Equal("registry.local:5000", host)

	host, err = normalizeRegistryURL("https://registry.local")
	is.NoError(err)
	is.Equal("registry.local", host)

	_, err = normalizeRegistryURL("registry.local:5000/some/path")
	is.Error(err)

	_, err = normalizeRegistryURL("")
	is.Error(err)

	// Docker Hub targets must be rejected
	for _, dockerHub := range []string{
		"docker.io",
		"index.docker.io",
		"registry-1.docker.io",
		"https://index.docker.io/v1/",
	} {
		_, err = normalizeRegistryURL(dockerHub)
		is.Error(err, dockerHub)
	}
}

func TestProxyMountPrefixes(t *testing.T) {
	routePrefix, locationPrefix := proxyMountPrefixes("/v2/registry-proxy/3/nginx/manifests/latest", "3")
	assert.Equal(t, "/v2/registry-proxy/3", routePrefix)
	assert.Equal(t, "/v2/registry-proxy/3", locationPrefix)

	routePrefix, locationPrefix = proxyMountPrefixes("/registry-proxies/3/v2/nginx/manifests/latest", "3")
	assert.Equal(t, "/registry-proxies/3/v2", routePrefix)
	assert.Equal(t, "/api/registry-proxies/3/v2", locationPrefix)
}

func newTestHandler(t *testing.T) (*Handler, *datastore.Store) {
	t.Helper()

	fips.InitFIPS(false)

	_, store := datastore.MustNewTestStore(t, true, false)

	h := NewHandler(testhelpers.NewTestRequestBouncer())
	h.DataStore = store
	h.CryptoService = &crypto.Service{}

	return h, store
}

func newAdminRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	ctx := security.StoreRestrictedRequestContext(req, &security.RestrictedRequestContext{
		IsAdmin: true,
		UserID:  1,
		User:    &portainer.User{ID: 1, Username: "testadmin", Role: portainer.AdministratorRole},
	})
	return req.WithContext(ctx)
}

func TestRegistryProxyCreateAndList(t *testing.T) {
	h, store := newTestHandler(t)

	payload, err := json.Marshal(map[string]any{
		"Name":           "local",
		"URL":            "http://registry.local:5000",
		"Authentication": true,
		"Username":       "registry-user",
		"Password":       "registry-pass",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newAdminRequest(http.MethodPost, "/registry-proxies", bytes.NewReader(payload)))
	require.Equal(t, http.StatusOK, w.Code)

	var created portainer.RegistryProxy
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "local", created.Name)
	assert.Equal(t, "registry.local:5000", created.URL)
	assert.Empty(t, created.Password, "password must be redacted in the response")

	w = httptest.NewRecorder()
	h.ServeHTTP(w, newAdminRequest(http.MethodGet, "/registry-proxies", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var proxies []portainer.RegistryProxy
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &proxies))
	require.Len(t, proxies, 1)
	assert.Empty(t, proxies[0].Password)

	stored, err := store.RegistryProxy().Read(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "registry-pass", stored.Password)
}

func TestRegistryProxyCreateRejectsDockerHub(t *testing.T) {
	h, _ := newTestHandler(t)

	payload, err := json.Marshal(map[string]any{
		"Name": "hub",
		"URL":  "registry-1.docker.io",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newAdminRequest(http.MethodPost, "/registry-proxies", bytes.NewReader(payload)))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegistryProxyDelete(t *testing.T) {
	h, store := newTestHandler(t)

	proxy := &portainer.RegistryProxy{Name: "local", URL: "registry.local:5000"}
	require.NoError(t, store.RegistryProxy().Create(proxy))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newAdminRequest(http.MethodDelete, "/registry-proxies/1", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err := store.RegistryProxy().Read(proxy.ID)
	assert.Error(t, err)
}

func createProxyForTest(t *testing.T, store *datastore.Store) *portainer.RegistryProxy {
	t.Helper()

	proxy := &portainer.RegistryProxy{Name: "local", URL: "registry.invalid:5000"}
	require.NoError(t, store.RegistryProxy().Create(proxy))

	return proxy
}

func TestV2PingWithPortainerBasicAuth(t *testing.T) {
	h, store := newTestHandler(t)
	createProxyForTest(t, store)

	hash, err := (&crypto.Service{}).Hash("secret")
	require.NoError(t, err)
	require.NoError(t, store.User().Create(&portainer.User{
		ID:       10,
		Username: "docker-user",
		Role:     portainer.StandardUserRole,
		Password: hash,
	}))

	req := httptest.NewRequest(http.MethodGet, "/v2/", nil)
	req.SetBasicAuth("docker-user", "secret")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "registry/2.0", w.Header().Get("Docker-Distribution-API-Version"))
}

func TestV2PingRejectsBadPortainerCredentials(t *testing.T) {
	h, store := newTestHandler(t)
	createProxyForTest(t, store)

	hash, err := (&crypto.Service{}).Hash("secret")
	require.NoError(t, err)
	require.NoError(t, store.User().Create(&portainer.User{
		ID:       10,
		Username: "docker-user",
		Role:     portainer.StandardUserRole,
		Password: hash,
	}))

	req := httptest.NewRequest(http.MethodGet, "/v2/", nil)
	req.SetBasicAuth("docker-user", "wrong-password")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestV2PushRequiresAdministrator(t *testing.T) {
	h, store := newTestHandler(t)
	createProxyForTest(t, store)

	hash, err := (&crypto.Service{}).Hash("secret")
	require.NoError(t, err)
	require.NoError(t, store.User().Create(&portainer.User{
		ID:       10,
		Username: "standard-user",
		Role:     portainer.StandardUserRole,
		Password: hash,
	}))

	req := httptest.NewRequest(http.MethodPost, "/v2/registry-proxy/1/nginx/blobs/uploads/", nil)
	req.SetBasicAuth("standard-user", "secret")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestV2PullFromStandardUserIsForwarded(t *testing.T) {
	h, store := newTestHandler(t)
	createProxyForTest(t, store)

	hash, err := (&crypto.Service{}).Hash("secret")
	require.NoError(t, err)
	require.NoError(t, store.User().Create(&portainer.User{
		ID:       10,
		Username: "standard-user",
		Role:     portainer.StandardUserRole,
		Password: hash,
	}))

	// registry.invalid does not resolve, so a successful authentication must
	// surface as a 502 proxy failure rather than a 401/403.
	req := httptest.NewRequest(http.MethodGet, "/v2/registry-proxy/1/nginx/manifests/latest", nil)
	req.SetBasicAuth("standard-user", "secret")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
