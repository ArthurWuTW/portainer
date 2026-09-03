package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

type serviceInstanceTargetResponse struct {
	EnvironmentID portainer.EndpointID `json:"EnvironmentId"`
	Environment   *portainer.Endpoint  `json:"Environment,omitempty"`
	Stack         *portainer.Stack     `json:"Stack,omitempty"`
	Missing       bool                 `json:"Missing"`
}

// @id ServiceInstanceTargets
// @summary List the resolved targets of a service instance
// @description Resolves the current deployment targets of a service instance and
// @description reports the managed stack (if any) on each target environment.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 200 {array} object "Success"
// @failure 400 "Invalid request"
// @failure 404 "Service instance not found"
// @failure 500 "Server error"
// @router /service-instances/{id}/targets [get]
func (handler *Handler) serviceInstanceTargets(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
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

	targets, err := handler.Service.ResolveTargets(instance)
	if err != nil {
		return httperror.InternalServerError("Unable to resolve service instance targets", err)
	}

	responses := make([]serviceInstanceTargetResponse, 0, len(targets.Endpoints)+len(targets.Missing))
	for _, endpoint := range targets.Endpoints {
		item := serviceInstanceTargetResponse{
			EnvironmentID: endpoint.ID,
			Environment:   &endpoint,
		}
		if stack, err := handler.Service.FindStackOnEndpoint(instance, endpoint.ID); err == nil {
			item.Stack = stack
		}
		responses = append(responses, item)
	}
	for _, missingID := range targets.Missing {
		responses = append(responses, serviceInstanceTargetResponse{
			EnvironmentID: missingID,
			Missing:       true,
		})
	}

	return response.JSON(w, responses)
}
