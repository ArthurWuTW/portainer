package registryproxies

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
	"github.com/portainer/portainer/api/http/security"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// dockerHubHosts are rejected as proxy targets: the registry proxy only points
// to local registries, never to external Docker Hub.
var dockerHubHosts = map[string]bool{
	"docker.io":               true,
	"index.docker.io":         true,
	"registry-1.docker.io":    true,
	"registry.hub.docker.com": true,
	"hub.docker.com":          true,
}

// Handler is the HTTP handler used to handle registry proxy operations.
type Handler struct {
	*mux.Router
	requestBouncer security.BouncerService
	DataStore      dataservices.DataStore
	CryptoService  portainer.CryptoService
}

// NewHandler creates a handler to manage registry proxy operations.
func NewHandler(bouncer security.BouncerService) *Handler {
	h := &Handler{
		Router:         mux.NewRouter(),
		requestBouncer: bouncer,
	}

	adminRouter := h.NewRoute().Subrouter()
	adminRouter.Use(bouncer.AdminAccess)
	adminRouter.Handle("/registry-proxies", httperror.LoggerHandler(h.registryProxyList)).Methods(http.MethodGet)
	adminRouter.Handle("/registry-proxies", httperror.LoggerHandler(h.registryProxyCreate)).Methods(http.MethodPost)
	adminRouter.Handle("/registry-proxies/{id}", httperror.LoggerHandler(h.registryProxyUpdate)).Methods(http.MethodPut)
	adminRouter.Handle("/registry-proxies/{id}", httperror.LoggerHandler(h.registryProxyDelete)).Methods(http.MethodDelete)
	adminRouter.Handle("/registry-proxies/{id}/tags", httperror.LoggerHandler(h.registryProxyTagAdd)).Methods(http.MethodPost)
	adminRouter.Handle("/registry-proxies/{id}/manifest", httperror.LoggerHandler(h.registryProxyManifestDelete)).Methods(http.MethodDelete)

	authenticatedRouter := h.NewRoute().Subrouter()
	authenticatedRouter.Use(bouncer.AuthenticatedAccess)
	authenticatedRouter.Handle("/registry-proxies/{id}", httperror.LoggerHandler(h.registryProxyInspect)).Methods(http.MethodGet)
	authenticatedRouter.Handle("/registry-proxies/{id}/catalog", httperror.LoggerHandler(h.registryProxyCatalog)).Methods(http.MethodGet)
	authenticatedRouter.Handle("/registry-proxies/{id}/tags", httperror.LoggerHandler(h.registryProxyTags)).Methods(http.MethodGet)

	// Docker registry v2 passthrough. Authenticated with Portainer credentials
	// (HTTP basic auth) or any regular Portainer authentication.
	h.PathPrefix("/registry-proxies/{id}/v2").Handler(h.portainerAuth(httperror.LoggerHandler(h.proxyToRegistry)))
	h.PathPrefix("/v2/registry-proxy/{id}/").Handler(h.portainerAuth(httperror.LoggerHandler(h.proxyToRegistry)))
	h.PathPrefix("/v2").Handler(h.portainerAuth(httperror.LoggerHandler(h.registryV2Ping)))

	return h
}

func hideFields(registryProxy *portainer.RegistryProxy) {
	registryProxy.Password = ""
}

func parseProxyID(raw string) (portainer.RegistryProxyID, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return 0, errors.New("invalid registry proxy identifier")
	}

	return portainer.RegistryProxyID(id), nil
}

// normalizeRegistryURL validates a registry proxy URL and returns the
// host[:port] part, rejecting schemes, paths and Docker Hub targets.
func normalizeRegistryURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("missing URL")
	}

	guessable := trimmed
	if !strings.Contains(guessable, "://") {
		guessable = "//" + guessable
	}

	parsed, err := url.Parse(guessable)
	if err != nil {
		return "", errors.Wrap(err, "invalid URL")
	}

	host := parsed.Host
	if host == "" {
		host = parsed.Path // input was a bare host[:port] without scheme
	}

	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("URL must not contain a path, only a host and optional port")
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", errors.New("URL must contain a valid host")
	}

	if dockerHubHosts[hostname] {
		return "", errors.New("proxying to Docker Hub is not supported, only local registries are allowed")
	}

	return host, nil
}

// toRegistry converts a registry proxy into the registry model consumed by the
// ORAS based registry helpers.
func toRegistry(proxy *portainer.RegistryProxy) portainer.Registry {
	return portainer.Registry{
		ID:             portainer.RegistryID(proxy.ID),
		Name:           proxy.Name,
		Type:           portainer.CustomRegistry,
		URL:            proxy.URL,
		Authentication: proxy.Authentication,
		Username:       proxy.Username,
		Password:       proxy.Password,
		ManagementConfiguration: &portainer.RegistryManagementConfiguration{
			Type: portainer.CustomRegistry,
			TLSConfig: portainer.TLSConfiguration{
				TLS:           proxy.TLS,
				TLSSkipVerify: proxy.TLSSkipVerify,
			},
		},
	}
}
