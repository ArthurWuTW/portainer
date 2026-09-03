package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceScheduledBuildCancel
// @summary Cancel a service instance scheduled build
// @description Cancels a scheduled build that is still pending or pulling.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Scheduled build identifier"
// @success 204 "Success"
// @failure 400 "Invalid request"
// @failure 404 "Scheduled build not found"
// @failure 500 "Server error"
// @router /service-instance-scheduled-builds/{id} [delete]
func (handler *Handler) serviceInstanceScheduledBuildCancel(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	id, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid scheduled build identifier route variable", err)
	}

	err = handler.Service.CancelScheduledBuild(portainer.ServiceInstanceScheduledBuildID(id))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find scheduled build", err)
	} else if err != nil {
		return httperror.BadRequest("Unable to cancel scheduled build", err)
	}

	return response.Empty(w)
}
