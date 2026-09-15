package registryproxies

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id RegistryProxyUpdate
// @summary Update a registry proxy
// @description Update a registry proxy. An empty password keeps the existing one.
// @description **Access policy**: administrator
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @accept json
// @produce json
// @param id path int true "Registry proxy identifier"
// @param body body registryProxyPayload true "Registry proxy details"
// @success 200 {object} portainer.RegistryProxy "Success"
// @failure 400 "Invalid request"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id} [put]
func (handler *Handler) registryProxyUpdate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	proxyID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid registry proxy identifier route variable", err)
	}

	var payload registryProxyPayload
	if err := request.DecodeAndValidateJSONPayload(r, &payload); err != nil {
		return httperror.BadRequest("Invalid request payload", err)
	}

	url, err := normalizeRegistryURL(payload.URL)
	if err != nil {
		return httperror.BadRequest("Invalid registry URL", err)
	}

	registryProxy, err := handler.DataStore.RegistryProxy().Read(portainer.RegistryProxyID(proxyID))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find a registry proxy with the specified identifier inside the database", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find a registry proxy with the specified identifier inside the database", err)
	}

	if payload.Authentication && payload.Password == "" {
		payload.Password = registryProxy.Password
		payload.Username = firstNonEmpty(payload.Username, registryProxy.Username)
	}

	registryProxy.Name = payload.Name
	registryProxy.URL = url
	registryProxy.TLS = payload.TLS
	registryProxy.TLSSkipVerify = payload.TLSSkipVerify
	registryProxy.Authentication = payload.Authentication
	registryProxy.Username = payload.Username
	registryProxy.Password = payload.Password

	if !registryProxy.Authentication {
		registryProxy.Username = ""
		registryProxy.Password = ""
	}

	if err := handler.DataStore.RegistryProxy().Update(portainer.RegistryProxyID(proxyID), registryProxy); err != nil {
		return httperror.InternalServerError("Unable to persist the registry proxy changes inside the database", err)
	}

	hideFields(registryProxy)

	return response.JSON(w, registryProxy)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
