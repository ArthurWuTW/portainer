package registryproxies

import (
	"context"
	"net/http"
	"strings"
	"time"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
	"github.com/portainer/portainer/pkg/liboras"
)

func (handler *Handler) loadProxy(r *http.Request) (*portainer.RegistryProxy, *httperror.HandlerError) {
	proxyID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return nil, httperror.BadRequest("Invalid registry proxy identifier route variable", err)
	}

	registryProxy, err := handler.DataStore.RegistryProxy().Read(portainer.RegistryProxyID(proxyID))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return nil, httperror.NotFound("Unable to find a registry proxy with the specified identifier inside the database", err)
	} else if err != nil {
		return nil, httperror.InternalServerError("Unable to find a registry proxy with the specified identifier inside the database", err)
	}

	return registryProxy, nil
}

// @id RegistryProxyCatalog
// @summary List image repositories in the proxied registry
// @description Retrieve the catalog (list of image repositories) of the proxied local registry.
// @description **Access policy**: authenticated
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Registry proxy identifier"
// @success 200 {object} registryProxyCatalogResponse "Success"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id}/catalog [get]
type registryProxyCatalogResponse struct {
	Repositories []string `json:"repositories"`
}

func (handler *Handler) registryProxyCatalog(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	registryProxy, handlerErr := handler.loadProxy(r)
	if handlerErr != nil {
		return handlerErr
	}

	registry := toRegistry(registryProxy)

	registryClient, err := liboras.CreateClient(registry)
	if err != nil {
		return httperror.InternalServerError("Unable to create registry client", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	repositories, err := liboras.ListRepositories(ctx, &registry, registryClient)
	if err != nil {
		return httperror.InternalServerError("Unable to list repositories from the registry", err)
	}

	if repositories == nil {
		repositories = []string{}
	}

	return response.JSON(w, registryProxyCatalogResponse{Repositories: repositories})
}

// @id RegistryProxyTags
// @summary List image versions (tags) for a repository
// @description Retrieve the list of tags for a repository inside the proxied local registry.
// @description **Access policy**: authenticated
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Registry proxy identifier"
// @param repository query string true "Repository name"
// @success 200 {object} registryProxyTagsResponse "Success"
// @failure 400 "Invalid request"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id}/tags [get]
type registryProxyTagsResponse struct {
	Repository string   `json:"repository"`
	Tags       []string `json:"tags"`
}

func (handler *Handler) registryProxyTags(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	registryProxy, handlerErr := handler.loadProxy(r)
	if handlerErr != nil {
		return handlerErr
	}

	repository, err := request.RetrieveQueryParameter(r, "repository", false)
	if err != nil {
		return httperror.BadRequest("Invalid query parameter: repository", err)
	}

	registry := toRegistry(registryProxy)

	registryClient, err := liboras.CreateClient(registry)
	if err != nil {
		return httperror.InternalServerError("Unable to create registry client", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	tags, err := liboras.ListTags(ctx, registryClient, repository)
	if err != nil {
		return httperror.InternalServerError("Unable to list tags from the registry", err)
	}

	if tags == nil {
		tags = []string{}
	}

	return response.JSON(w, registryProxyTagsResponse{Repository: repository, Tags: tags})
}

type registryProxyTagAddPayload struct {
	// Repository name
	Repository string `json:"repository" example:"my-image" validate:"required"`
	// Existing tag or digest the manifest should be resolved from
	Source string `json:"source" example:"1.2.3" validate:"required"`
	// New tag pointing at the same manifest
	Target string `json:"target" example:"latest" validate:"required"`
}

func (payload *registryProxyTagAddPayload) Validate(r *http.Request) error {
	return nil
}

// @id RegistryProxyTagAdd
// @summary Tag an existing image manifest with a new version tag
// @description Point a new tag at an existing manifest inside the proxied local registry.
// @description **Access policy**: administrator
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @accept json
// @param id path int true "Registry proxy identifier"
// @param body body registryProxyTagAddPayload true "Tag details"
// @success 204 "Success"
// @failure 400 "Invalid request"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id}/tags [post]
func (handler *Handler) registryProxyTagAdd(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	registryProxy, handlerErr := handler.loadProxy(r)
	if handlerErr != nil {
		return handlerErr
	}

	var payload registryProxyTagAddPayload
	if err := request.DecodeAndValidateJSONPayload(r, &payload); err != nil {
		return httperror.BadRequest("Invalid request payload", err)
	}

	registry := toRegistry(registryProxy)

	registryClient, err := liboras.CreateClient(registry)
	if err != nil {
		return httperror.InternalServerError("Unable to create registry client", err)
	}

	if err := liboras.TagManifest(registryClient, payload.Repository, payload.Source, payload.Target); err != nil {
		return httperror.InternalServerError("Unable to tag the image manifest in the registry", err)
	}

	return response.Empty(w)
}

// @id RegistryProxyManifestDelete
// @summary Delete an image (manifest) from the proxied registry
// @description Delete a manifest by tag or digest inside the proxied local registry.
// @description Deleting by tag removes the tag, deleting by digest removes the manifest.
// @description **Access policy**: administrator
// @tags registry_proxies
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Registry proxy identifier"
// @param repository query string true "Repository name"
// @param reference query string true "Tag or digest of the manifest to delete"
// @success 204 "Success"
// @failure 400 "Invalid request"
// @failure 404 "Registry proxy not found"
// @failure 500 "Server error"
// @router /registry-proxies/{id}/manifest [delete]
func (handler *Handler) registryProxyManifestDelete(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	registryProxy, handlerErr := handler.loadProxy(r)
	if handlerErr != nil {
		return handlerErr
	}

	repository, err := request.RetrieveQueryParameter(r, "repository", false)
	if err != nil {
		return httperror.BadRequest("Invalid query parameter: repository", err)
	}

	reference, err := request.RetrieveQueryParameter(r, "reference", false)
	if err != nil {
		return httperror.BadRequest("Invalid query parameter: reference", err)
	}

	registry := toRegistry(registryProxy)

	registryClient, err := liboras.CreateClient(registry)
	if err != nil {
		return httperror.InternalServerError("Unable to create registry client", err)
	}

	// Deleting by digest removes the manifest (and every tag pointing to it),
	// which is the expected behavior. Deleting by tag must only untag that
	// single tag without affecting sibling tags pointing to the same manifest.
	if strings.Contains(reference, ":") {
		err = liboras.DeleteManifestByDigest(registryClient, repository, reference)
	} else {
		err = liboras.SafeDeleteTags(registryClient, repository, []string{reference})
	}
	if err != nil {
		return httperror.InternalServerError("Unable to delete the image from the registry", err)
	}

	return response.Empty(w)
}
