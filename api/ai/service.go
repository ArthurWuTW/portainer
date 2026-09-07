package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// maxToolRounds bounds the number of LLM round-trips to prevent runaway loops.
	maxToolRounds = 8

	systemPrompt = `You are an AI assistant embedded in Portainer, a container management platform.
You help users understand and troubleshoot the current Portainer instance and the
environments (endpoints) it manages.

You have access to a set of read-only tools that query live data from Portainer:
- Portainer metadata (endpoints, stacks, instance state)
- Docker runtime state (containers, images, networks, volumes, logs, stats, system info)

Rules:
- Only use the provided tools to obtain data. Never invent or guess data.
- Always call a tool before answering a question about the current state.
- Tool results are JSON. Interpret them and answer concisely.
- You can only read data. You cannot modify, restart, stop, or delete anything.
- If a tool returns a permission error, tell the user they lack access to that resource.
- Keep answers focused and technical.`
)

// Service drives the tool-calling loop between the LLM and the Portainer tools.
type Service struct {
	llmClientFactory func(config LLMConfig) LLMClient
	tools            []Tool
}

// NewService returns a Service wired with the production LLM client factory and
// the default set of read-only tools.
func NewService() *Service {
	return &Service{
		llmClientFactory: NewLLMClient,
		tools:            NewTools(),
	}
}

// NewServiceWithLLMClient returns a Service that always uses the provided LLM
// client. Intended for testing.
func NewServiceWithLLMClient(client LLMClient) *Service {
	return &Service{
		llmClientFactory: func(_ LLMConfig) LLMClient { return client },
		tools:            NewTools(),
	}
}

// RunChat executes the tool-calling loop, streaming events to w as SSE.
func (s *Service) RunChat(ctx context.Context, w http.ResponseWriter, req ChatRequest, tc *ToolContext) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	client := s.llmClientFactory(req.LLMConfig)
	definitions := ToolDefinitions(s.tools)

	messages := append([]ChatMessage{{Role: "system", Content: systemPrompt}}, req.Messages...)

	for round := 0; round < maxToolRounds; round++ {
		resp, err := client.Chat(ctx, messages, definitions, func(delta string) {
			writeSSE(w, SSEEventToken, TokenEvent{Delta: delta})
		})
		if err != nil {
			writeSSE(w, SSEEventError, ErrorEvent{Message: err.Error()})
			return
		}

		if len(resp.ToolCalls) == 0 {
			writeSSE(w, SSEEventDone, map[string]any{})
			return
		}

		messages = append(messages, assistantMessageWithToolCalls(resp))

		for _, call := range resp.ToolCalls {
			writeSSE(w, SSEEventToolStart, ToolStartEvent{Name: call.Name, Arguments: call.Arguments})

			result, execErr := ExecuteTool(ctx, s.tools, call.Name, json.RawMessage(call.Arguments), tc)
			ok := execErr == nil
			if execErr != nil {
				result = fmt.Sprintf(`{"error": %q}`, execErr.Error())
			}

			writeSSE(w, SSEEventToolEnd, ToolEndEvent{Name: call.Name, OK: ok})

			messages = append(messages, ChatMessage{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	writeSSE(w, SSEEventError, ErrorEvent{Message: "maximum tool rounds exceeded"})
}

func assistantMessageWithToolCalls(resp *LLMResponse) ChatMessage {
	msg := ChatMessage{Role: "assistant", Content: resp.Content}
	for _, call := range resp.ToolCalls {
		tc := AssistantToolCall{ID: call.ID, Type: "function"}
		tc.Function.Name = call.Name
		tc.Function.Arguments = call.Arguments
		msg.ToolCalls = append(msg.ToolCalls, tc)
	}
	return msg
}
