package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceDelete
// @summary Delete a service instance
// @description Delete a service instance. Stacks deployed on target environments
// @description are left in place and become regular stacks.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 204 "Success"
// @failure 400 "Invalid request"
// @failure 404 "Service instance not found"
// @failure 500 "Server error"
// @router /service-instances/{id} [delete]
func (handler *Handler) serviceInstanceDelete(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
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

	if err := handler.Service.Delete(instance.ID); err != nil {
		return httperror.InternalServerError("Unable to delete service instance", err)
	}

	return response.Empty(w)
}
