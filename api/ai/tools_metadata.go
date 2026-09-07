package ai

import (
	"context"
	"encoding/json"
	"fmt"

	portainer "github.com/portainer/portainer/api"
)

// endpointSummary is the minimal, redacted representation of an endpoint
// returned to the LLM.
type endpointSummary struct {
	ID     portainer.EndpointID `json:"id"`
	Name   string               `json:"name"`
	Type   string               `json:"type"`
	Status string               `json:"status"`
	Tags   []string             `json:"tags,omitempty"`
}

// endpointDetail extends the summary with a few extra non-sensitive fields.
type endpointDetail struct {
	endpointSummary
	URL          string `json:"url,omitempty"`
	LastCheckIn  int64  `json:"lastCheckIn,omitempty"`
	IsEdgeDevice bool   `json:"isEdgeDevice,omitempty"`
}

// stackSummary is the minimal representation of a stack returned to the LLM.
type stackSummary struct {
	ID          portainer.StackID      `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	EndpointID  portainer.EndpointID   `json:"endpointId"`
	Status      string                 `json:"status"`
	CreatedBy   string                 `json:"createdBy,omitempty"`
	CreationDate int64                 `json:"creationDate,omitempty"`
}

var endpointTypeNames = map[portainer.EndpointType]string{
	portainer.DockerEnvironment:                  "docker",
	portainer.AgentOnDockerEnvironment:           "agent_on_docker",
	portainer.AzureEnvironment:                   "azure",
	portainer.EdgeAgentOnDockerEnvironment:       "edge_agent_on_docker",
	portainer.KubernetesLocalEnvironment:         "kubernetes_local",
	portainer.AgentOnKubernetesEnvironment:       "agent_on_kubernetes",
	portainer.EdgeAgentOnKubernetesEnvironment:   "edge_agent_on_kubernetes",
}

var endpointStatusNames = map[portainer.EndpointStatus]string{
	portainer.EndpointStatusUp:   "up",
	portainer.EndpointStatusDown: "down",
}

var stackTypeNames = map[portainer.StackType]string{
	portainer.DockerSwarmStack:   "docker_swarm",
	portainer.DockerComposeStack: "docker_compose",
	portainer.KubernetesStack:    "kubernetes",
}

var stackStatusNames = map[portainer.StackStatus]string{
	portainer.StackStatusActive:    "active",
	portainer.StackStatusInactive:  "inactive",
	portainer.StackStatusDeploying: "deploying",
	portainer.StackStatusError:     "error",
}

func endpointTypeName(t portainer.EndpointType) string {
	if name, ok := endpointTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", t)
}

func endpointStatusName(s portainer.EndpointStatus) string {
	if name, ok := endpointStatusNames[s]; ok {
		return name
	}
	return "unknown"
}

func (tc *ToolContext) tagNames(tagIDs []portainer.TagID) []string {
	if len(tagIDs) == 0 {
		return nil
	}

	tags, err := tc.DataStore.Tag().ReadAll()
	if err != nil {
		return nil
	}

	byID := make(map[portainer.TagID]string, len(tags))
	for _, tag := range tags {
		byID[tag.ID] = tag.Name
	}

	names := make([]string, 0, len(tagIDs))
	for _, id := range tagIDs {
		if name, ok := byID[id]; ok {
			names = append(names, name)
		}
	}
	return names
}

func toEndpointSummary(tc *ToolContext, e portainer.Endpoint) endpointSummary {
	return endpointSummary{
		ID:     e.ID,
		Name:   e.Name,
		Type:   endpointTypeName(e.Type),
		Status: endpointStatusName(e.Status),
		Tags:   tc.tagNames(e.TagIDs),
	}
}

// --- get_portainer_state ---

type getPortainerStateTool struct{}

func (t *getPortainerStateTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_portainer_state",
		Description: "Get a high-level summary of the Portainer instance: version, instance ID, and counts of environments (endpoints) and stacks the user can access.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *getPortainerStateTool) Execute(_ context.Context, _ json.RawMessage, tc *ToolContext) (string, error) {
	endpoints, err := tc.AuthorizedEndpoints()
	if err != nil {
		return "", err
	}

	stacks, err := tc.DataStore.Stack().ReadAll()
	if err != nil {
		return "", err
	}

	authorizedEndpointIDs := make(map[portainer.EndpointID]bool, len(endpoints))
	for _, e := range endpoints {
		authorizedEndpointIDs[e.ID] = true
	}

	visibleStacks := 0
	for _, s := range stacks {
		if authorizedEndpointIDs[s.EndpointID] {
			visibleStacks++
		}
	}

	instanceID := ""
	if versionService := tc.DataStore.Version(); versionService != nil {
		if id, err := versionService.InstanceID(); err == nil {
			instanceID = id
		}
	}

	result := map[string]any{
		"version":        portainer.APIVersion,
		"instanceId":     instanceID,
		"endpointCount":  len(endpoints),
		"stackCount":     visibleStacks,
	}

	return marshalResult(result)
}

// --- get_endpoints ---

type getEndpointsTool struct{}

func (t *getEndpointsTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_endpoints",
		Description: "List the environments (endpoints) the current user is allowed to access, with id, name, type, status and tags.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *getEndpointsTool) Execute(_ context.Context, _ json.RawMessage, tc *ToolContext) (string, error) {
	endpoints, err := tc.AuthorizedEndpoints()
	if err != nil {
		return "", err
	}

	summaries := make([]endpointSummary, 0, len(endpoints))
	for _, e := range endpoints {
		summaries = append(summaries, toEndpointSummary(tc, e))
	}

	return marshalResult(map[string]any{"endpoints": summaries})
}

// --- get_endpoint ---

type getEndpointTool struct{}

func (t *getEndpointTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_endpoint",
		Description: "Get details for a single environment (endpoint) by its id.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"endpoint_id": map[string]any{
					"type":        "integer",
					"description": "The id of the environment (endpoint).",
				},
			},
			"required": []string{"endpoint_id"},
		},
	}
}

type getEndpointArgs struct {
	EndpointID portainer.EndpointID `json:"endpoint_id"`
}

func (t *getEndpointTool) Execute(_ context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getEndpointArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.AuthorizedEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	detail := endpointDetail{
		endpointSummary: toEndpointSummary(tc, *endpoint),
		URL:             sanitizeURL(endpoint.URL),
		LastCheckIn:     endpoint.LastCheckInDate,
		IsEdgeDevice:    endpoint.IsEdgeDevice,
	}

	return marshalResult(detail)
}

// --- get_stacks ---

type getStacksTool struct{}

func (t *getStacksTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_stacks",
		Description: "List stacks (deployed compose/swarm/k8s workloads) the user can access, optionally filtered by environment (endpoint) id.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"endpoint_id": map[string]any{
					"type":        "integer",
					"description": "Optional. Only return stacks deployed on this environment (endpoint).",
				},
			},
		},
	}
}

type getStacksArgs struct {
	EndpointID *portainer.EndpointID `json:"endpoint_id"`
}

func (t *getStacksTool) Execute(_ context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getStacksArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	authorized, err := tc.AuthorizedEndpoints()
	if err != nil {
		return "", err
	}

	authorizedIDs := make(map[portainer.EndpointID]bool, len(authorized))
	for _, e := range authorized {
		authorizedIDs[e.ID] = true
	}

	stacks, err := tc.DataStore.Stack().ReadAll()
	if err != nil {
		return "", err
	}

	summaries := make([]stackSummary, 0, len(stacks))
	for _, s := range stacks {
		if !authorizedIDs[s.EndpointID] {
			continue
		}
		if a.EndpointID != nil && s.EndpointID != *a.EndpointID {
			continue
		}
		summaries = append(summaries, stackSummary{
			ID:           s.ID,
			Name:         s.Name,
			Type:         stackTypeName(s.Type),
			EndpointID:   s.EndpointID,
			Status:       stackStatusName(s.Status),
			CreatedBy:    s.CreatedBy,
			CreationDate: s.CreationDate,
		})
	}

	return marshalResult(map[string]any{"stacks": summaries})
}

// --- get_stack ---

type getStackTool struct{}

func (t *getStackTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_stack",
		Description: "Get details for a single stack by its id.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"stack_id": map[string]any{
					"type":        "integer",
					"description": "The id of the stack.",
				},
			},
			"required": []string{"stack_id"},
		},
	}
}

type getStackArgs struct {
	StackID portainer.StackID `json:"stack_id"`
}

func (t *getStackTool) Execute(_ context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getStackArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	stack, err := tc.DataStore.Stack().Read(a.StackID)
	if err != nil {
		return "", fmt.Errorf("stack %d not found", a.StackID)
	}

	if _, err := tc.AuthorizedEndpoint(stack.EndpointID); err != nil {
		return "", err
	}

	return marshalResult(stackSummary{
		ID:           stack.ID,
		Name:         stack.Name,
		Type:         stackTypeName(stack.Type),
		EndpointID:   stack.EndpointID,
		Status:       stackStatusName(stack.Status),
		CreatedBy:    stack.CreatedBy,
		CreationDate: stack.CreationDate,
	})
}

func stackTypeName(t portainer.StackType) string {
	if name, ok := stackTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", t)
}

func stackStatusName(s portainer.StackStatus) string {
	if name, ok := stackStatusNames[s]; ok {
		return name
	}
	return "unknown"
}

func marshalResult(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool result: %w", err)
	}
	return string(data), nil
}
