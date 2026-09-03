package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"

	"github.com/pkg/errors"
)

type createServiceInstancePayload struct {
	Name           string                              `json:"Name" validate:"required"`
	Description    string                              `json:"Description"`
	TargetType     portainer.ServiceInstanceTargetType `json:"TargetType" validate:"required,gt=0"`
	GroupID        portainer.EndpointGroupID           `json:"GroupId"`
	EnvironmentIDs []portainer.EndpointID              `json:"EnvironmentIds"`
	ComposeFile    string                              `json:"ComposeFile" validate:"required"`
	Env            []portainer.Pair                    `json:"Env"`
}

func (payload *createServiceInstancePayload) Validate(r *http.Request) error {
	if len(payload.Name) == 0 {
		return errors.New("invalid service instance name")
	}

	if len(payload.ComposeFile) == 0 {
		return errors.New("invalid compose file content")
	}

	switch payload.TargetType {
	case portainer.ServiceInstanceTargetGroup:
		if payload.GroupID == 0 {
			return errors.New("group ID is required when target type is group")
		}
	case portainer.ServiceInstanceTargetEnvironments:
		if len(payload.EnvironmentIDs) == 0 {
			return errors.New("at least one environment is required when target type is environments")
		}
	default:
		return errors.New("invalid target type")
	}

	return nil
}

// @id ServiceInstanceCreate
// @summary Create a service instance
// @description Create a new service instance.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @accept json
// @produce json
// @param body body createServiceInstancePayload true "Service instance configuration"
// @success 200 {object} portainer.ServiceInstance "Success"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 500 "Server error"
// @router /service-instances [post]
func (handler *Handler) serviceInstanceCreate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	var payload createServiceInstancePayload
	err := request.DecodeAndValidateJSONPayload(r, &payload)
	if err != nil {
		return httperror.BadRequest("Invalid request payload", err)
	}

	securityContext, err := security.RetrieveRestrictedRequestContext(r)
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve info from request context", err)
	}

	// Validate that the user has access to all target environments
	targets, err := handler.resolveTargetsFromPayload(&payload)
	if err != nil {
		return httperror.InternalServerError("Unable to resolve target environments", err)
	}
	if len(targets) == 0 {
		return httperror.BadRequest("No deployment targets", errors.New("no deployment targets"))
	}

	for _, endpoint := range targets {
		if err := handler.requestBouncer.AuthorizedEndpointOperation(r, &endpoint); err != nil {
			return httperror.Forbidden("Permission denied to access environment", err)
		}
	}

	instance := &portainer.ServiceInstance{
		Name:           payload.Name,
		Description:    payload.Description,
		TargetType:     payload.TargetType,
		GroupID:        payload.GroupID,
		EnvironmentIDs: payload.EnvironmentIDs,
		ComposeFile:    payload.ComposeFile,
		Env:            payload.Env,
		CreatedBy:      securityContext.User.Username,
	}

	if err := handler.Service.Create(instance); err != nil {
		return httperror.InternalServerError("Unable to create service instance", err)
	}

	return response.JSON(w, instance)
}

// resolveTargetsFromPayload resolves the target endpoints from the create payload.
func (handler *Handler) resolveTargetsFromPayload(payload *createServiceInstancePayload) ([]portainer.Endpoint, error) {
	switch payload.TargetType {
	case portainer.ServiceInstanceTargetGroup:
		return handler.DataStore.Endpoint().ReadAll(func(e portainer.Endpoint) bool {
			return e.GroupID == payload.GroupID
		})
	case portainer.ServiceInstanceTargetEnvironments:
		endpoints := make([]portainer.Endpoint, 0, len(payload.EnvironmentIDs))
		for _, id := range payload.EnvironmentIDs {
			endpoint, err := handler.DataStore.Endpoint().Endpoint(id)
			if err != nil {
				if handler.DataStore.IsErrObjectNotFound(err) {
					continue
				}
				return nil, err
			}
			endpoints = append(endpoints, *endpoint)
		}
		return endpoints, nil
	default:
		return nil, errors.New("unknown target type")
	}
}
