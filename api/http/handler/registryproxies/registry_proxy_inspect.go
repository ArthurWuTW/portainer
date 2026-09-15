package registryproxies

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id RegistryProxyInspect
// @summary Inspect a registry proxy
// @description Retrieve details for a registry proxy.
// @description **Access policy**: authenticated
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @produce json
// @param id path int true "Registry proxy identifier"
// @success 200 {object} portainer.RegistryProxy "Success"
// @failure 400 "Invalid request"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id} [get]
func (handler *Handler) registryProxyInspect(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	proxyID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid registry proxy identifier route variable", err)
	}

	if _, err := security.RetrieveRestrictedRequestContext(r); err != nil {
		return httperror.InternalServerError("Unable to retrieve info from request context", err)
	}

	registryProxy, err := handler.DataStore.RegistryProxy().Read(portainer.RegistryProxyID(proxyID))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find a registry proxy with the specified identifier inside the database", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find a registry proxy with the specified identifier inside the database", err)
	}

	hideFields(registryProxy)

	return response.JSON(w, registryProxy)
}
