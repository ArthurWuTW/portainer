package ai

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	portainer "github.com/portainer/portainer/api"
	aiservice "github.com/portainer/portainer/api/ai"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/internal/testhelpers"

	"github.com/stretchr/testify/require"
)

// fakeLLM returns pre-scripted responses in order.
type fakeLLM struct {
	responses []aiservice.LLMResponse
	calls     int
}

func (f *fakeLLM) Chat(_ context.Context, _ []aiservice.ChatMessage, _ []aiservice.ToolDefinition, onDelta func(string)) (*aiservice.LLMResponse, error) {
	if f.calls >= len(f.responses) {
		return &aiservice.LLMResponse{}, nil
	}
	resp := f.responses[f.calls]
	f.calls++
	if onDelta != nil && resp.Content != "" {
		onDelta(resp.Content)
	}
	return &resp, nil
}

func newTestHandler(t *testing.T, llm aiservice.LLMClient) *Handler {
	t.Helper()

	h := NewHandler(testhelpers.NewTestRequestBouncer())
	h.DataStore = testhelpers.NewDatastore(
		testhelpers.WithEndpoints([]portainer.Endpoint{
			{ID: 1, Name: "prod", Type: portainer.DockerEnvironment, GroupID: 1},
		}),
		testhelpers.WithEndpointGroups([]portainer.EndpointGroup{
			{ID: 1, Name: "default"},
		}),
	)
	h.Service = aiservice.NewServiceWithLLMClient(llm)

	return h
}

func newChatRequest(t *testing.T, body string, admin bool) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/ai/chat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	role := portainer.StandardUserRole
	if admin {
		role = portainer.AdministratorRole
	}

	req = req.WithContext(security.StoreRestrictedRequestContext(req, &security.RestrictedRequestContext{
		IsAdmin: admin,
		UserID:  1,
		User:    &portainer.User{ID: 1, Role: role},
	}))

	return req
}

func TestChat_InvalidBody(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, &fakeLLM{})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, newChatRequest(t, "{invalid json", true))

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestChat_MissingMessages(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, &fakeLLM{})

	rr := httptest.NewRecorder()
	body := `{"llmConfig":{"baseUrl":"http://localhost:8000/v1","model":"qwen3"}}`
	h.ServeHTTP(rr, newChatRequest(t, body, true))

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestChat_MissingLLMConfig(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, &fakeLLM{})

	rr := httptest.NewRecorder()
	body := `{"messages":[{"role":"user","content":"hi"}]}`
	h.ServeHTTP(rr, newChatRequest(t, body, true))

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestChat_StreamsResponse(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{
		responses: []aiservice.LLMResponse{
			{Content: "Hello from the model"},
		},
	}
	h := newTestHandler(t, llm)

	rr := httptest.NewRecorder()
	body := `{"messages":[{"role":"user","content":"hi"}],"llmConfig":{"baseUrl":"http://localhost:8000/v1","model":"qwen3"}}`
	h.chat(rr, newChatRequest(t, body, true))

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	require.Contains(t, rr.Body.String(), "event: token")
	require.Contains(t, rr.Body.String(), `"delta":"Hello from the model"`)
	require.Contains(t, rr.Body.String(), "event: done")
	require.Equal(t, 1, llm.calls)
}

func TestChat_ToolCallThenAnswer(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{
		responses: []aiservice.LLMResponse{
			{
				ToolCalls: []aiservice.ToolCall{
					{ID: "call_1", Name: "get_endpoints", Arguments: "{}"},
				},
			},
			{Content: "There is one endpoint."},
		},
	}
	h := newTestHandler(t, llm)

	rr := httptest.NewRecorder()
	body := `{"messages":[{"role":"user","content":"list endpoints"}],"llmConfig":{"baseUrl":"http://localhost:8000/v1","model":"qwen3"}}`
	h.chat(rr, newChatRequest(t, body, true))

	require.Equal(t, http.StatusOK, rr.Code)
	out := rr.Body.String()
	require.Contains(t, out, "event: tool_start")
	require.Contains(t, out, `"name":"get_endpoints"`)
	require.Contains(t, out, "event: tool_end")
	require.Contains(t, out, `"ok":true`)
	require.Contains(t, out, "event: done")
	require.Equal(t, 2, llm.calls)
}
