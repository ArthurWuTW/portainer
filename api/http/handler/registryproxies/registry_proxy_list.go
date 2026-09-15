package registryproxies

import (
	"net/http"

	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id RegistryProxyList
// @summary List registry proxies
// @description List all registry proxies.
// @description **Access policy**: administrator
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @produce json
// @success 200 {array} portainer.RegistryProxy "Success"
// @failure 500 "Server error"
// @router /registry-proxies [get]
func (handler *Handler) registryProxyList(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	registryProxies, err := handler.DataStore.RegistryProxy().ReadAll()
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve registry proxies from the database", err)
	}

	for idx := range registryProxies {
		hideFields(&registryProxies[idx])
	}

	return response.JSON(w, registryProxies)
}
