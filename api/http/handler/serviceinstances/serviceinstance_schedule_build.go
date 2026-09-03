package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/serviceinstance"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"

	"github.com/pkg/errors"
)

type scheduleServiceInstanceBuildPayload struct {
	ComposeFile string `json:"ComposeFile" validate:"required"`
	DeployAt    int64  `json:"DeployAt" validate:"required,gt=0"`
}

func (payload *scheduleServiceInstanceBuildPayload) Validate(r *http.Request) error {
	return nil
}

// @id ServiceInstanceScheduleBuild
// @summary Schedule a build for a service instance
// @description Pulls the images referenced by the provided compose file on all
// @description target environments immediately, then deploys the compose file
// @description at the given timestamp.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @accept json
// @produce json
// @param id path int true "Service instance identifier"
// @param body body scheduleServiceInstanceBuildPayload true "Scheduled build configuration"
// @success 202 {object} portainer.ServiceInstanceScheduledBuild "Scheduled build created"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 404 "Service instance not found"
// @failure 500 "Server error"
// @router /service-instances/{id}/schedule-build [post]
func (handler *Handler) serviceInstanceScheduleBuild(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	id, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid service instance identifier route variable", err)
	}

	securityContext, err := security.RetrieveRestrictedRequestContext(r)
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve info from request context", err)
	}

	instance, err := handler.Service.Read(portainer.ServiceInstanceID(id))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find service instance", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find service instance", err)
	}

	var payload scheduleServiceInstanceBuildPayload
	if err := request.DecodeAndValidateJSONPayload(r, &payload); err != nil {
		return httperror.BadRequest("Invalid request payload", err)
	}

	// Re-check authorization on every operation: the user may have lost
	// access to one of the target environments since the instance was created.
	targets, err := handler.Service.ResolveTargets(instance)
	if err != nil {
		return httperror.InternalServerError("Unable to resolve target environments", err)
	}
	if len(targets.Endpoints) == 0 {
		return httperror.BadRequest("No deployment targets", serviceinstance.ErrNoDeploymentTargets)
	}

	for _, endpoint := range targets.Endpoints {
		if err := handler.requestBouncer.AuthorizedEndpointOperation(r, &endpoint); err != nil {
			return httperror.Forbidden("Permission denied to access environment", err)
		}
	}

	build, err := handler.Service.ScheduleBuild(r.Context(), instance.ID, payload.ComposeFile, payload.DeployAt, securityContext)
	if err != nil {
		switch {
		case errors.Is(err, serviceinstance.ErrNoDeploymentTargets):
			return httperror.BadRequest("No deployment targets", err)
		case errors.Is(err, serviceinstance.ErrInvalidComposeFile):
			return httperror.BadRequest("Invalid compose file", err)
		default:
			return httperror.InternalServerError("Unable to schedule build", err)
		}
	}

	return response.JSONWithStatus(w, build, http.StatusAccepted)
}
