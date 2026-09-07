// Package ai implements the AI context layer: a set of read-only tools that
// expose Portainer metadata and endpoint runtime state to an LLM, plus the
// tool-calling loop that drives an OpenAI-compatible model.
package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// LLMConfig describes an OpenAI-compatible LLM endpoint. It is provided by the
// frontend on every chat request and is never persisted server-side.
type LLMConfig struct {
	BaseURL     string  `json:"baseUrl"`
	APIKey      string  `json:"apiKey"`
	Model       string  `json:"model"`
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

// ChatMessage is a single conversation message in the OpenAI chat format.
type ChatMessage struct {
	Role       string              `json:"role"`
	Content    string              `json:"content,omitempty"`
	ToolCalls  []AssistantToolCall `json:"tool_calls,omitempty"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
}

// AssistantToolCall is a tool invocation requested by the model.
type AssistantToolCall struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatRequest is the body of POST /api/ai/chat.
type ChatRequest struct {
	Messages  []ChatMessage `json:"messages"`
	LLMConfig LLMConfig     `json:"llmConfig"`
}

// SSE event names streamed back to the frontend.
const (
	SSEEventToken     = "token"
	SSEEventToolStart = "tool_start"
	SSEEventToolEnd   = "tool_end"
	SSEEventDone      = "done"
	SSEEventError     = "error"
)

// TokenEvent is the payload of a "token" SSE event.
type TokenEvent struct {
	Delta string `json:"delta"`
}

// ToolStartEvent is the payload of a "tool_start" SSE event.
type ToolStartEvent struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolEndEvent is the payload of a "tool_end" SSE event.
type ToolEndEvent struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
}

// ErrorEvent is the payload of an "error" SSE event.
type ErrorEvent struct {
	Message string `json:"message"`
}

// writeSSE writes a single server-sent event to the response writer.
func writeSSE(w http.ResponseWriter, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(payload))

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
