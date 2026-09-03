package serviceinstances

import (
	"net/http"

	"github.com/portainer/portainer/api/dataservices"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/serviceinstance"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"

	"github.com/gorilla/mux"
)

// Handler is the HTTP handler used to handle service instance operations.
type Handler struct {
	*mux.Router
	requestBouncer security.BouncerService
	DataStore      dataservices.DataStore
	Service        *serviceinstance.Service
}

// NewHandler creates a handler to manage service instance operations.
func NewHandler(bouncer security.BouncerService) *Handler {
	h := &Handler{
		Router:         mux.NewRouter(),
		requestBouncer: bouncer,
	}

	h.Handle("/service-instances",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceList))).Methods(http.MethodGet)
	h.Handle("/service-instances",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceCreate))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceInspect))).Methods(http.MethodGet)
	h.Handle("/service-instances/{id}",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceUpdate))).Methods(http.MethodPut)
	h.Handle("/service-instances/{id}",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceDelete))).Methods(http.MethodDelete)
	h.Handle("/service-instances/{id}/deploy",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceDeploy))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}/start",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceStart))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}/stop",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceStop))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}/redeploy",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceRedeploy))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}/refresh",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceRefresh))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}/schedule-build",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceScheduleBuild))).Methods(http.MethodPost)
	h.Handle("/service-instances/{id}/scheduled-builds",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceScheduledBuilds))).Methods(http.MethodGet)
	h.Handle("/service-instance-scheduled-builds/{id}",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceScheduledBuildCancel))).Methods(http.MethodDelete)
	h.Handle("/service-instances/{id}/targets",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceTargets))).Methods(http.MethodGet)
	h.Handle("/service-instances/{id}/operations",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceOperations))).Methods(http.MethodGet)
	h.Handle("/service-instance-operations/{id}",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.serviceInstanceOperationInspect))).Methods(http.MethodGet)

	return h
}
