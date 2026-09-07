package ai

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/internal/testhelpers"

	"github.com/stretchr/testify/require"
)

// fakeLLMClient returns pre-scripted responses in order.
type fakeLLMClient struct {
	responses []LLMResponse
	calls     int
}

func (f *fakeLLMClient) Chat(_ context.Context, _ []ChatMessage, _ []ToolDefinition, onContentDelta func(string)) (*LLMResponse, error) {
	if f.calls >= len(f.responses) {
		return &LLMResponse{}, nil
	}
	resp := f.responses[f.calls]
	f.calls++
	if onContentDelta != nil && resp.Content != "" {
		onContentDelta(resp.Content)
	}
	return &resp, nil
}

func newTestService(client LLMClient) *Service {
	return &Service{
		llmClientFactory: func(_ LLMConfig) LLMClient { return client },
		tools:            NewTools(),
	}
}

// testToolContext returns a ToolContext backed by a test datastore so that
// metadata tools (e.g. get_portainer_state) can run without a real DB.
func testToolContext() *ToolContext {
	ds := testhelpers.NewDatastore(
		testhelpers.WithEndpoints([]portainer.Endpoint{
			endpointWithAccess(1, 1, 2),
		}),
		testhelpers.WithStacks([]portainer.Stack{}),
	)

	return &ToolContext{
		DataStore: ds,
		User:      adminUser(),
	}
}

func TestRunChat_FinalAnswerWithoutTools(t *testing.T) {
	t.Parallel()

	client := &fakeLLMClient{
		responses: []LLMResponse{{Content: "Hello from the model"}},
	}
	svc := newTestService(client)

	rr := httptest.NewRecorder()
	svc.RunChat(context.Background(), rr, ChatRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "hi"}},
		LLMConfig: LLMConfig{BaseURL: "http://localhost:8000/v1", Model: "qwen3"},
	}, &ToolContext{})

	body := rr.Body.String()
	require.Contains(t, body, "event: token")
	require.Contains(t, body, `"delta":"Hello from the model"`)
	require.Contains(t, body, "event: done")
	require.NotContains(t, body, "event: error")
	require.Equal(t, 1, client.calls)
}

func TestRunChat_ToolCallThenAnswer(t *testing.T) {
	t.Parallel()

	client := &fakeLLMClient{
		responses: []LLMResponse{
			{
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "get_portainer_state", Arguments: "{}"},
				},
			},
			{Content: "There are 2 endpoints."},
		},
	}
	svc := newTestService(client)

	rr := httptest.NewRecorder()
	svc.RunChat(context.Background(), rr, ChatRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "how many endpoints?"}},
		LLMConfig: LLMConfig{BaseURL: "http://localhost:8000/v1", Model: "qwen3"},
	}, testToolContext())

	body := rr.Body.String()
	require.Contains(t, body, "event: tool_start")
	require.Contains(t, body, `"name":"get_portainer_state"`)
	require.Contains(t, body, "event: tool_end")
	require.Contains(t, body, `"ok":true`)
	require.Contains(t, body, "event: token")
	require.Contains(t, body, "event: done")
	require.Equal(t, 2, client.calls)
}

func TestRunChat_ToolErrorIsReportedToModel(t *testing.T) {
	t.Parallel()

	client := &fakeLLMClient{
		responses: []LLMResponse{
			{
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "get_endpoint", Arguments: `{"endpoint_id":999}`},
				},
			},
			{Content: "That endpoint does not exist."},
		},
	}
	svc := newTestService(client)

	rr := httptest.NewRecorder()
	svc.RunChat(context.Background(), rr, ChatRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "show endpoint 999"}},
		LLMConfig: LLMConfig{BaseURL: "http://localhost:8000/v1", Model: "qwen3"},
	}, testToolContext())

	body := rr.Body.String()
	require.Contains(t, body, `"ok":false`)
	require.Contains(t, body, "event: done")
}

func TestRunChat_MaxRoundsExceeded(t *testing.T) {
	t.Parallel()

	responses := make([]LLMResponse, 0, maxToolRounds)
	for i := 0; i < maxToolRounds; i++ {
		responses = append(responses, LLMResponse{
			ToolCalls: []ToolCall{{ID: "call", Name: "get_portainer_state", Arguments: "{}"}},
		})
	}

	client := &fakeLLMClient{responses: responses}
	svc := newTestService(client)

	rr := httptest.NewRecorder()
	svc.RunChat(context.Background(), rr, ChatRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "loop"}},
		LLMConfig: LLMConfig{BaseURL: "http://localhost:8000/v1", Model: "qwen3"},
	}, testToolContext())

	body := rr.Body.String()
	require.Contains(t, body, "maximum tool rounds exceeded")
	require.Equal(t, maxToolRounds, client.calls)
}

func TestRunChat_SSEHeaders(t *testing.T) {
	t.Parallel()

	client := &fakeLLMClient{
		responses: []LLMResponse{{Content: "ok"}},
	}
	svc := newTestService(client)

	rr := httptest.NewRecorder()
	svc.RunChat(context.Background(), rr, ChatRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "hi"}},
		LLMConfig: LLMConfig{BaseURL: "http://localhost:8000/v1", Model: "qwen3"},
	}, &ToolContext{})

	require.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
	require.Equal(t, "no", rr.Header().Get("X-Accel-Buffering"))
	require.True(t, strings.HasPrefix(rr.Body.String(), "event: "))
}
