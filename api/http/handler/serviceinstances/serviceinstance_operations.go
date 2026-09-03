package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceOperations
// @summary List the operations of a service instance
// @description List all lifecycle operations of a service instance, newest first.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 200 {array} portainer.ServiceInstanceOperation "Success"
// @failure 400 "Invalid request"
// @failure 404 "Service instance not found"
// @failure 500 "Server error"
// @router /service-instances/{id}/operations [get]
func (handler *Handler) serviceInstanceOperations(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
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

	operations, err := handler.DataStore.ServiceInstanceOperation().ReadAllByServiceInstanceID(instance.ID)
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve service instance operations", err)
	}

	// newest first
	for i, j := 0, len(operations)-1; i < j; i, j = i+1, j-1 {
		operations[i], operations[j] = operations[j], operations[i]
	}

	return response.JSON(w, operations)
}
