package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolDefinition describes a tool that can be exposed to the LLM.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Tool is a read-only operation the LLM can invoke.
type Tool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error)
}

// NewTools returns the full set of read-only AI tools.
func NewTools() []Tool {
	return []Tool{
		&getPortainerStateTool{},
		&getEndpointsTool{},
		&getEndpointTool{},
		&getStacksTool{},
		&getStackTool{},
		&getContainersTool{},
		&getContainerTool{},
		&getContainerLogsTool{},
		&getContainerStatsTool{},
		&getImagesTool{},
		&getNetworksTool{},
		&getVolumesTool{},
		&getSystemInfoTool{},
	}
}

// ToolDefinitions returns the tool definitions for the LLM request.
func ToolDefinitions(tools []Tool) []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// ExecuteTool finds and executes the named tool.
func ExecuteTool(ctx context.Context, tools []Tool, name string, args json.RawMessage, tc *ToolContext) (string, error) {
	for _, t := range tools {
		if t.Definition().Name == name {
			return t.Execute(ctx, args, tc)
		}
	}
	return "", fmt.Errorf("unknown tool %q", name)
}
