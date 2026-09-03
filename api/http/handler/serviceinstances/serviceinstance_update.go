package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"

	"github.com/pkg/errors"
)

type updateServiceInstancePayload struct {
	Name           string                              `json:"Name"`
	Description    string                              `json:"Description"`
	TargetType     portainer.ServiceInstanceTargetType `json:"TargetType"`
	GroupID        portainer.EndpointGroupID           `json:"GroupId"`
	EnvironmentIDs []portainer.EndpointID              `json:"EnvironmentIds"`
	ComposeFile    string                              `json:"ComposeFile"`
	Env            []portainer.Pair                    `json:"Env"`
}

func (payload *updateServiceInstancePayload) Validate(r *http.Request) error {
	if payload.TargetType != 0 {
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
	}

	return nil
}

// @id ServiceInstanceUpdate
// @summary Update a service instance
// @description Update an existing service instance.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @accept json
// @produce json
// @param id path int true "Service instance identifier"
// @param body body updateServiceInstancePayload true "Service instance configuration"
// @success 200 {object} portainer.ServiceInstance "Success"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 404 "Service instance not found"
// @failure 500 "Server error"
// @router /service-instances/{id} [put]
func (handler *Handler) serviceInstanceUpdate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	id, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid service instance identifier route variable", err)
	}

	instance, err := handler.Service.Read(portainer.ServiceInstanceID(id))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find service instance", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find service instance", err)
	}

	var payload updateServiceInstancePayload
	if err := request.DecodeAndValidateJSONPayload(r, &payload); err != nil {
		return httperror.BadRequest("Invalid request payload", err)
	}

	if payload.Name != "" {
		instance.Name = payload.Name
	}
	instance.Description = payload.Description
	if payload.TargetType != 0 {
		instance.TargetType = payload.TargetType
	}
	instance.GroupID = payload.GroupID
	instance.EnvironmentIDs = payload.EnvironmentIDs
	if payload.ComposeFile != "" {
		instance.ComposeFile = payload.ComposeFile
	}
	instance.Env = payload.Env

	// Validate the (possibly new) targets before persisting.
	targets, err := handler.Service.ResolveTargets(instance)
	if err != nil {
		return httperror.InternalServerError("Unable to resolve target environments", err)
	}
	if len(targets.Endpoints) == 0 {
		return httperror.BadRequest("No deployment targets", errors.New("no deployment targets"))
	}

	if err := handler.Service.Update(portainer.ServiceInstanceID(id), instance); err != nil {
		return httperror.InternalServerError("Unable to update service instance", err)
	}

	return response.JSON(w, instance)
}
