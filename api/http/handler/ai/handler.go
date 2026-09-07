package ai

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	portainer "github.com/portainer/portainer/api"
	ai "github.com/portainer/portainer/api/ai"
	"github.com/portainer/portainer/api/dataservices"
	dockerclient "github.com/portainer/portainer/api/docker/client"
	"github.com/portainer/portainer/api/http/security"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
)

// Handler is the HTTP handler for the AI chat API.
type Handler struct {
	*mux.Router
	requestBouncer      security.BouncerService
	DataStore           dataservices.DataStore
	DockerClientFactory *dockerclient.ClientFactory
	Service             *ai.Service
}

// NewHandler creates a handler to manage AI chat operations.
func NewHandler(bouncer security.BouncerService) *Handler {
	h := &Handler{
		Router:         mux.NewRouter(),
		requestBouncer: bouncer,
		Service:        ai.NewService(),
	}

	h.Handle("/ai/chat",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.chat))).Methods(http.MethodPost)

	return h
}

// @id AIChat
// @summary Chat with the AI assistant
// @description Streams an AI chat response (SSE) for the authenticated user. The LLM endpoint
// @description configuration is provided in the request body and is never persisted.
// @description **Access policy**: authenticated
// @tags ai
// @security jwt
// @accept json
// @produce text/event-stream
// @param body body ChatRequest true "Chat request"
// @success 200 {string} string "Server-sent event stream"
// @failure 400 "Bad request"
// @failure 500 "Internal server error"
// @router /ai/chat [post]
func (h *Handler) chat(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	var req ai.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperror.BadRequest("Invalid request body", err)
	}

	if len(req.Messages) == 0 {
		return httperror.BadRequest("No messages provided", nil)
	}

	if req.LLMConfig.BaseURL == "" || req.LLMConfig.Model == "" {
		return httperror.BadRequest("LLM configuration requires baseUrl and model", nil)
	}

	securityContext, err := security.RetrieveRestrictedRequestContext(r)
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve user details from request context", err)
	}

	endpointGroups, err := h.DataStore.EndpointGroup().ReadAll()
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve endpoint groups", err)
	}

	toolContext := &ai.ToolContext{
		DataStore: h.DataStore,
		DockerClientProvider: func(endpoint *portainer.Endpoint) (ai.DockerAPI, error) {
			return h.DockerClientFactory.CreateClient(endpoint, "", nil)
		},
		User:            securityContext.User,
		TeamMemberships: securityContext.UserMemberships,
		EndpointGroups:  endpointGroups,
	}

	h.Service.RunChat(r.Context(), w, req, toolContext)

	return nil
}
