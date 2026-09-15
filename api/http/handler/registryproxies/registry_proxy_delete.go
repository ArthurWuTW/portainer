package registryproxies

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id RegistryProxyDelete
// @summary Delete a registry proxy
// @description Remove a registry proxy. The proxied registry itself is not modified.
// @description **Access policy**: administrator
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Registry proxy identifier"
// @success 204 "Success"
// @failure 400 "Invalid request"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id} [delete]
func (handler *Handler) registryProxyDelete(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	proxyID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid registry proxy identifier route variable", err)
	}

	if err := handler.DataStore.RegistryProxy().Delete(portainer.RegistryProxyID(proxyID)); err != nil {
		if handler.DataStore.IsErrObjectNotFound(err) {
			return httperror.NotFound("Unable to find a registry proxy with the specified identifier inside the database", err)
		}

		return httperror.InternalServerError("Unable to remove the registry proxy from the database", err)
	}

	return response.Empty(w)
}
