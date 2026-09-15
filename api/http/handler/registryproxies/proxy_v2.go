package registryproxies

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/registryhttp"

	"github.com/gorilla/mux"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type registryProxyIdentityKey struct{}

type registryProxyIdentity struct {
	username string
	isAdmin  bool
}

// portainerAuth authenticates registry proxy requests with Portainer
// credentials. It accepts HTTP basic auth (docker login style) checked against
// the Portainer user database, and falls back to regular Portainer
// authentication (JWT, API key or session cookie).
func (handler *Handler) portainerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			// advertise the Basic challenge so docker clients retry with their
			// stored credentials instead of failing on a bare 401
			w.Header().Set("WWW-Authenticate", `Basic realm="Portainer Registry Proxy", charset="UTF-8"`)

			handler.requestBouncer.AuthenticatedAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tokenData, err := security.RetrieveTokenData(r)
				if err != nil {
					handler.writeUnauthorized(w, r)
					return
				}

				identity := registryProxyIdentity{
					username: tokenData.Username,
					isAdmin:  tokenData.Role == portainer.AdministratorRole,
				}

				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), registryProxyIdentityKey{}, identity)))
			})).ServeHTTP(w, r)

			return
		}

		user, err := handler.DataStore.User().UserByUsername(username)
		if err != nil {
			handler.writeUnauthorized(w, r)
			return
		}

		if err := handler.CryptoService.CompareHashAndData(user.Password, password); err != nil {
			handler.writeUnauthorized(w, r)
			return
		}

		identity := registryProxyIdentity{
			username: user.Username,
			isAdmin:  user.Role == portainer.AdministratorRole,
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), registryProxyIdentityKey{}, identity)))
	})
}

func (handler *Handler) writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Portainer Registry Proxy", charset="UTF-8"`)
	httperror.WriteError(w, http.StatusUnauthorized, "Invalid Portainer credentials", nil)
}

func identityFromContext(r *http.Request) (registryProxyIdentity, bool) {
	identity, ok := r.Context().Value(registryProxyIdentityKey{}).(registryProxyIdentity)
	return identity, ok
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// proxyToRegistry streams Docker registry v2 API requests to the configured
// local registry. Two mount forms are supported:
//
//	/v2/registry-proxy/{id}/<repo>/...            (docker CLI compatible, root mount)
//	/api/registry-proxies/{id}/v2/<repo>/...      (API mount)
//
// The caller authenticates with Portainer credentials; the upstream registry is
// contacted with the credentials stored on the registry proxy.
func (handler *Handler) proxyToRegistry(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	identity, ok := identityFromContext(r)
	if !ok {
		return httperror.Unauthorized("Authentication required", nil)
	}

	if isWriteMethod(r.Method) && !identity.isAdmin {
		return httperror.Forbidden("Only Portainer administrators can push or delete images through the registry proxy", nil)
	}

	vars := mux.Vars(r)

	proxyID, err := parseProxyID(vars["id"])
	if err != nil {
		return httperror.BadRequest("Invalid registry proxy identifier", err)
	}

	registryProxy, err := handler.DataStore.RegistryProxy().Read(proxyID)
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find a registry proxy with the specified identifier inside the database", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find a registry proxy with the specified identifier inside the database", err)
	}

	routePrefix, locationPrefix := proxyMountPrefixes(r.URL.Path, vars["id"])

	remaining := strings.TrimPrefix(r.URL.Path, routePrefix)
	if remaining == "" || remaining == "/" {
		remaining = "/"
	}

	transport, scheme, err := registryhttp.BuildTransportAndSchemeFromTLSConfig(portainer.TLSConfiguration{
		TLS:           registryProxy.TLS,
		TLSSkipVerify: registryProxy.TLSSkipVerify,
	})
	if err != nil {
		return httperror.InternalServerError("Unable to create a transport for the proxied registry", err)
	}

	target := &url.URL{Scheme: scheme, Host: registryProxy.URL}

	reverseProxy := &httputil.ReverseProxy{
		Transport: retry.NewTransport(transport),
		Rewrite: func(proxyReq *httputil.ProxyRequest) {
			proxyReq.SetURL(target)
			proxyReq.Out.URL.Path = "/v2" + remaining
			proxyReq.Out.URL.RawPath = ""
			proxyReq.SetXForwarded()

			// never forward Portainer credentials to the upstream registry
			proxyReq.Out.Header.Del("Authorization")
			proxyReq.Out.Header.Del("Cookie")

			if registryProxy.Authentication {
				proxyReq.Out.SetBasicAuth(registryProxy.Username, registryProxy.Password)
			}
		},
		ModifyResponse: func(res *http.Response) error {
			// prevent clients from bypassing Portainer via the registry token realm
			res.Header.Del("WWW-Authenticate")

			if location := res.Header.Get("Location"); location != "" {
				res.Header.Set("Location", rewriteLocationHeader(location, locationPrefix, r))
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			httperror.WriteError(w, http.StatusBadGateway, "Registry proxy failure", err)
		},
	}

	reverseProxy.ServeHTTP(w, r)

	return nil
}

// proxyMountPrefixes returns the prefix to strip from the incoming (possibly
// /api stripped) path and the prefix to use when rewriting Location headers.
func proxyMountPrefixes(path, proxyID string) (routePrefix string, locationPrefix string) {
	if strings.HasPrefix(path, "/v2/registry-proxy/") {
		prefix := "/v2/registry-proxy/" + proxyID
		return prefix, prefix
	}

	prefix := "/registry-proxies/" + proxyID + "/v2"
	return prefix, "/api" + prefix
}

func rewriteLocationHeader(location, locationPrefix string, r *http.Request) string {
	parsed, err := url.Parse(location)
	if err != nil {
		return location
	}

	path := parsed.Path
	if path == "" {
		path = location
	}

	if path != "/v2" && !strings.HasPrefix(path, "/v2/") {
		return location
	}

	newPath := locationPrefix + strings.TrimPrefix(path, "/v2")

	if parsed.IsAbs() {
		parsed.Scheme = requestScheme(r)
		parsed.Host = r.Host
	}

	parsed.Path = newPath

	return parsed.String()
}

func requestScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}

// registryV2Ping answers the /v2 base endpoint check performed by docker
// clients before any pull or push.
func (handler *Handler) registryV2Ping(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	if _, ok := identityFromContext(r); !ok {
		return httperror.Unauthorized("Authentication required", nil)
	}

	if r.URL.Path != "/v2" && r.URL.Path != "/v2/" {
		return httperror.NotFound("Image references must be prefixed with a registry proxy, e.g. <host>/registry-proxy/<id>/<image>:<tag>", nil)
	}

	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)

	return nil
}
