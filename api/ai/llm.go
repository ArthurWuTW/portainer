package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTemperature = 0.2
	defaultMaxTokens   = 2048
	llmRequestTimeout  = 5 * time.Minute
	maxErrorBodyBytes  = 4096
)

// LLMClient is the interface for calling an OpenAI-compatible LLM.
type LLMClient interface {
	// Chat performs a single (streamed) chat completion. Content deltas are
	// forwarded to onContentDelta as they arrive. The returned response holds
	// the accumulated content and any tool calls requested by the model.
	Chat(ctx context.Context, messages []ChatMessage, tools []ToolDefinition, onContentDelta func(string)) (*LLMResponse, error)
}

// LLMResponse is the accumulated result of a single (streamed) LLM call.
type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolCall is a single tool invocation requested by the LLM.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type openAIClient struct {
	config LLMConfig
	client *http.Client
}

// NewLLMClient returns an LLMClient that talks to an OpenAI-compatible endpoint.
func NewLLMClient(config LLMConfig) LLMClient {
	return &openAIClient{
		config: config,
		client: &http.Client{Timeout: llmRequestTimeout},
	}
}

// openAITool is the OpenAI-compatible wire format for a tool definition.
type openAITool struct {
	Type     string         `json:"type"`
	Function ToolDefinition `json:"function"`
}

func toOpenAITools(tools []ToolDefinition) []openAITool {
	out := make([]openAITool, 0, len(tools))
	for _, t := range tools {
		out = append(out, openAITool{Type: "function", Function: t})
	}
	return out
}

func (c *openAIClient) Chat(ctx context.Context, messages []ChatMessage, tools []ToolDefinition, onContentDelta func(string)) (*LLMResponse, error) {
	body := map[string]any{
		"model":    c.config.Model,
		"messages": messages,
		"stream":   true,
	}

	temperature := c.config.Temperature
	if temperature == 0 {
		temperature = defaultTemperature
	}
	body["temperature"] = temperature

	maxTokens := c.config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	body["max_tokens"] = maxTokens

	if len(tools) > 0 {
		body["tools"] = toOpenAITools(tools)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode LLM request: %w", err)
	}

	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readLLMError(resp)
	}

	return readStream(resp.Body, onContentDelta)
}

func readLLMError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	return fmt.Errorf("LLM request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
}

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

func readStream(body io.Reader, onContentDelta func(string)) (*LLMResponse, error) {
	result := &LLMResponse{}
	toolCalls := map[int]*ToolCall{}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				result.Content += choice.Delta.Content
				if onContentDelta != nil {
					onContentDelta(choice.Delta.Content)
				}
			}

			for _, tc := range choice.Delta.ToolCalls {
				existing, ok := toolCalls[tc.Index]
				if !ok {
					existing = &ToolCall{ID: tc.ID, Name: tc.Function.Name}
					toolCalls[tc.Index] = existing
				}

				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Function.Name != "" {
					existing.Name = tc.Function.Name
				}
				existing.Arguments += tc.Function.Arguments
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read LLM stream: %w", err)
	}

	for i := 0; i < len(toolCalls); i++ {
		if tc, ok := toolCalls[i]; ok {
			result.ToolCalls = append(result.ToolCalls, *tc)
		}
	}

	return result, nil
}
