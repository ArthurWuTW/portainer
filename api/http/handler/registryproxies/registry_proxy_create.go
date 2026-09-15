package registryproxies

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

type registryProxyPayload struct {
	// Name of the registry proxy
	Name string `json:"Name" example:"my-local-registry" validate:"required"`
	// Host and optional port of the local Docker registry (without scheme)
	URL string `json:"URL" example:"registry.local:5000" validate:"required"`
	// Use TLS to contact the local registry
	TLS bool `json:"TLS" example:"false"`
	// Skip the verification of the local registry TLS certificate
	TLSSkipVerify bool `json:"TLSSkipVerify" example:"false"`
	// Is authentication against the local registry enabled
	Authentication bool `json:"Authentication" example:"false"`
	// Username used to authenticate against the local registry
	Username string `json:"Username" example:"registry user"`
	// Password used to authenticate against the local registry
	Password string `json:"Password,omitempty" example:"registry_password"`
}

func (payload *registryProxyPayload) Validate(r *http.Request) error {
	if _, err := normalizeRegistryURL(payload.URL); err != nil {
		return err
	}

	return nil
}

// @id RegistryProxyCreate
// @summary Create a registry proxy
// @description Register a local Docker registry to be exposed through Portainer.
// @description Proxying to external Docker Hub is not supported.
// @description **Access policy**: administrator
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @accept json
// @produce json
// @param body body registryProxyPayload true "Registry proxy details"
// @success 200 {object} portainer.RegistryProxy "Success"
// @failure 400 "Invalid request"
// @failure 409 "Registry proxy already exists"
// @failure 500 "Server error"
// @router /registry-proxies [post]
func (handler *Handler) registryProxyCreate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	var payload registryProxyPayload
	if err := request.DecodeAndValidateJSONPayload(r, &payload); err != nil {
		return httperror.BadRequest("Invalid request payload", err)
	}

	url, err := normalizeRegistryURL(payload.URL)
	if err != nil {
		return httperror.BadRequest("Invalid registry URL", err)
	}

	registryProxies, err := handler.DataStore.RegistryProxy().ReadAll()
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve registry proxies from the database", err)
	}

	for _, existing := range registryProxies {
		if existing.Name == payload.Name {
			return httperror.Conflict("Registry proxy with the same name already exists", nil)
		}
	}

	registryProxy := &portainer.RegistryProxy{
		Name:           payload.Name,
		URL:            url,
		TLS:            payload.TLS,
		TLSSkipVerify:  payload.TLSSkipVerify,
		Authentication: payload.Authentication,
		Username:       payload.Username,
		Password:       payload.Password,
	}

	if !registryProxy.Authentication {
		registryProxy.Username = ""
		registryProxy.Password = ""
	}

	if err := handler.DataStore.RegistryProxy().Create(registryProxy); err != nil {
		return httperror.InternalServerError("Unable to persist the registry proxy inside the database", err)
	}

	hideFields(registryProxy)

	return response.JSON(w, registryProxy)
}
