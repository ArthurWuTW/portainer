package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceOperationInspect
// @summary Inspect a service instance operation
// @description Inspect a service instance operation and its per-target results.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance operation identifier"
// @success 200 {object} portainer.ServiceInstanceOperation "Success"
// @failure 400 "Invalid request"
// @failure 404 "Operation not found"
// @failure 500 "Server error"
// @router /service-instance-operations/{id} [get]
func (handler *Handler) serviceInstanceOperationInspect(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	id, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return httperror.BadRequest("Invalid service instance operation identifier route variable", err)
	}

	operation, err := handler.DataStore.ServiceInstanceOperation().Read(portainer.ServiceInstanceOperationID(id))
	if handler.DataStore.IsErrObjectNotFound(err) {
		return httperror.NotFound("Unable to find service instance operation", err)
	} else if err != nil {
		return httperror.InternalServerError("Unable to find service instance operation", err)
	}

	return response.JSON(w, operation)
}
