package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/pkg/liboras"
)

const (
	registryToolTimeout = 30 * time.Second
	maxRegistryItems    = 200
)

// --- list_registry_proxies ---

type listRegistryProxiesTool struct{}

func (t *listRegistryProxiesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "list_registry_proxies",
		Description: "List the local Docker registry proxies registered in Portainer (id, name, url and whether authentication/TLS are enabled).",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

type registryProxySummary struct {
	ID             portainer.RegistryProxyID `json:"id"`
	Name           string                    `json:"name"`
	URL            string                    `json:"url"`
	TLS            bool                      `json:"tls"`
	Authentication bool                      `json:"authentication"`
}

func (t *listRegistryProxiesTool) Execute(_ context.Context, _ json.RawMessage, tc *ToolContext) (string, error) {
	if tc.User.Role != portainer.AdministratorRole {
		return "", errors.New("permission denied: only administrators can list registry proxies")
	}

	proxies, err := tc.DataStore.RegistryProxy().ReadAll()
	if err != nil {
		return "", err
	}

	summaries := make([]registryProxySummary, 0, len(proxies))
	for _, p := range proxies {
		summaries = append(summaries, registryProxySummary{
			ID:             p.ID,
			Name:           p.Name,
			URL:            sanitizeURL(p.URL),
			TLS:            p.TLS,
			Authentication: p.Authentication,
		})
	}

	return marshalResult(map[string]any{"registryProxies": summaries})
}

// --- get_registry_proxy_images ---

type getRegistryProxyImagesTool struct{}

func (t *getRegistryProxyImagesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_registry_proxy_images",
		Description: "List the image repositories stored in a local registry proxy, or the tags (image versions) of one repository when a repository name is provided.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"proxy_id": map[string]any{
					"type":        "integer",
					"description": "The id of the registry proxy.",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Optional. Repository name to list its tags (image versions) instead of the repository list.",
				},
			},
			"required": []string{"proxy_id"},
		},
	}
}

type getRegistryProxyImagesArgs struct {
	ProxyID    portainer.RegistryProxyID `json:"proxy_id"`
	Repository string                    `json:"repository"`
}

func (t *getRegistryProxyImagesTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	if tc.User.Role != portainer.AdministratorRole {
		return "", errors.New("permission denied: only administrators can query registry proxies")
	}

	var a getRegistryProxyImagesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	proxy, err := tc.DataStore.RegistryProxy().Read(a.ProxyID)
	if err != nil {
		return "", fmt.Errorf("registry proxy %d not found", a.ProxyID)
	}

	registry := portainer.Registry{
		ID:             portainer.RegistryID(proxy.ID),
		Name:           proxy.Name,
		Type:           portainer.CustomRegistry,
		URL:            proxy.URL,
		Authentication: proxy.Authentication,
		Username:       proxy.Username,
		Password:       proxy.Password,
		ManagementConfiguration: &portainer.RegistryManagementConfiguration{
			Type: portainer.CustomRegistry,
			TLSConfig: portainer.TLSConfiguration{
				TLS:           proxy.TLS,
				TLSSkipVerify: proxy.TLSSkipVerify,
			},
		},
	}

	registryClient, err := liboras.CreateClient(registry)
	if err != nil {
		return "", fmt.Errorf("failed to create registry client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, registryToolTimeout)
	defer cancel()

	if a.Repository != "" {
		tags, err := liboras.ListTags(ctx, registryClient, a.Repository)
		if err != nil {
			return "", fmt.Errorf("failed to list tags: %w", err)
		}

		return marshalResult(map[string]any{
			"registryProxy": proxy.Name,
			"repository":    a.Repository,
			"tags":          capStrings(tags, maxRegistryItems),
			"truncated":     len(tags) > maxRegistryItems,
		})
	}

	repositories, err := liboras.ListRepositories(ctx, &registry, registryClient)
	if err != nil {
		return "", fmt.Errorf("failed to list repositories: %w", err)
	}

	// Hide repositories left empty by tag deletions, matching the UI catalog
	repositories, err = liboras.FilterRepositoriesWithTags(ctx, registryClient, repositories)
	if err != nil {
		return "", fmt.Errorf("failed to filter empty repositories: %w", err)
	}

	return marshalResult(map[string]any{
		"registryProxy": proxy.Name,
		"repositories":  capStrings(repositories, maxRegistryItems),
		"truncated":     len(repositories) > maxRegistryItems,
	})
}

func capStrings(values []string, max int) []string {
	if len(values) > max {
		return values[:max]
	}
	return values
}
