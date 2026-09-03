package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceInspect
// @summary Inspect a service instance
// @description Inspect a service instance.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 200 {object} portainer.ServiceInstance "Success"
// @failure 400 "Invalid request"
// @failure 404 "Service instance not found"
// @failure 500 "Server error"
// @router /service-instances/{id} [get]
func (handler *Handler) serviceInstanceInspect(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
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

	return response.JSON(w, instance)
}
