package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/serviceinstance"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/request"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceDeploy
// @summary Deploy a service instance
// @description Starts an asynchronous deploy operation against all target environments.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 202 {object} portainer.ServiceInstanceOperation "Operation started"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 404 "Service instance not found"
// @failure 409 "Operation already in progress"
// @failure 500 "Server error"
// @router /service-instances/{id}/deploy [post]
func (handler *Handler) serviceInstanceDeploy(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	return handler.startOperation(w, r, portainer.ServiceInstanceOperationDeploy)
}

// @id ServiceInstanceStart
// @summary Start a service instance
// @description Starts an asynchronous start operation against all target environments.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 202 {object} portainer.ServiceInstanceOperation "Operation started"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 404 "Service instance not found"
// @failure 409 "Operation already in progress"
// @failure 500 "Server error"
// @router /service-instances/{id}/start [post]
func (handler *Handler) serviceInstanceStart(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	return handler.startOperation(w, r, portainer.ServiceInstanceOperationStart)
}

// @id ServiceInstanceStop
// @summary Stop a service instance
// @description Starts an asynchronous stop operation against all target environments.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 202 {object} portainer.ServiceInstanceOperation "Operation started"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 404 "Service instance not found"
// @failure 409 "Operation already in progress"
// @failure 500 "Server error"
// @router /service-instances/{id}/stop [post]
func (handler *Handler) serviceInstanceStop(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	return handler.startOperation(w, r, portainer.ServiceInstanceOperationStop)
}

// @id ServiceInstanceRedeploy
// @summary Redeploy a service instance
// @description Starts an asynchronous redeploy operation against all target environments.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @param id path int true "Service instance identifier"
// @success 202 {object} portainer.ServiceInstanceOperation "Operation started"
// @failure 400 "Invalid request"
// @failure 403 "Permission denied"
// @failure 404 "Service instance not found"
// @failure 409 "Operation already in progress"
// @failure 500 "Server error"
// @router /service-instances/{id}/redeploy [post]
func (handler *Handler) serviceInstanceRedeploy(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	return handler.startOperation(w, r, portainer.ServiceInstanceOperationRedeploy)
}

// startOperation is the shared implementation for the async lifecycle
// endpoints (deploy, start, stop, redeploy).
func (handler *Handler) startOperation(w http.ResponseWriter, r *http.Request, operationType portainer.ServiceInstanceOperationType) *httperror.HandlerError {
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

	operation, err := handler.Service.ExecuteOperation(r.Context(), instance.ID, operationType, securityContext)
	if err != nil {
		switch err {
		case serviceinstance.ErrOperationInProgress:
			return httperror.Conflict("A service instance operation is already in progress", err)
		case serviceinstance.ErrNoDeploymentTargets:
			return httperror.BadRequest("No deployment targets", err)
		default:
			return httperror.InternalServerError("Unable to start service instance operation", err)
		}
	}

	return response.JSONWithStatus(w, operation, http.StatusAccepted)
}
