package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
	"github.com/portainer/portainer/api/http/security"
)

// ToolContext carries the authenticated user's context and the dependencies
// needed by the AI tools.
type ToolContext struct {
	DataStore            dataservices.DataStore
	DockerClientProvider DockerClientProvider
	User                 *portainer.User
	TeamMemberships      []portainer.TeamMembership
	EndpointGroups       []portainer.EndpointGroup
}

// DockerClientProvider creates a Docker client for an endpoint. It is a
// function type so the production factory and test fakes both fit.
type DockerClientProvider func(endpoint *portainer.Endpoint) (DockerAPI, error)

// DockerAPI is the subset of the Docker client used by the AI tools.
type DockerAPI interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	NetworkList(ctx context.Context, options network.ListOptions) ([]network.Inspect, error)
	VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error)
	Info(ctx context.Context) (system.Info, error)
}

// CanAccessEndpoint reports whether the user may access the given endpoint.
func (tc *ToolContext) CanAccessEndpoint(endpoint *portainer.Endpoint) bool {
	if tc.User.Role == portainer.AdministratorRole {
		return true
	}

	// AuthorizedEndpointAccess dereferences the group, so fall back to an
	// empty group when the endpoint's group is not loaded.
	group := portainer.EndpointGroup{}
	for i := range tc.EndpointGroups {
		if tc.EndpointGroups[i].ID == endpoint.GroupID {
			group = tc.EndpointGroups[i]
			break
		}
	}

	return security.AuthorizedEndpointAccess(endpoint, &group, tc.User.ID, tc.TeamMemberships)
}

// AuthorizedEndpoints returns the endpoints the user is allowed to see.
func (tc *ToolContext) AuthorizedEndpoints() ([]portainer.Endpoint, error) {
	endpoints, err := tc.DataStore.Endpoint().Endpoints()
	if err != nil {
		return nil, err
	}

	ctx := &security.RestrictedRequestContext{
		IsAdmin:         tc.User.Role == portainer.AdministratorRole,
		UserID:          tc.User.ID,
		UserMemberships: tc.TeamMemberships,
		User:            tc.User,
	}

	return security.FilterEndpoints(endpoints, tc.EndpointGroups, ctx), nil
}

// AuthorizedEndpoint loads an endpoint and verifies the user can access it.
func (tc *ToolContext) AuthorizedEndpoint(id portainer.EndpointID) (*portainer.Endpoint, error) {
	endpoint, err := tc.DataStore.Endpoint().Endpoint(id)
	if err != nil {
		return nil, fmt.Errorf("environment %d not found", id)
	}

	if !tc.CanAccessEndpoint(endpoint) {
		return nil, fmt.Errorf("permission denied to access environment %d", id)
	}

	return endpoint, nil
}

// dockerClient returns a Docker client for the endpoint, or an error if the
// endpoint type is not Docker-based.
func (tc *ToolContext) dockerClient(endpoint *portainer.Endpoint) (DockerAPI, error) {
	if tc.DockerClientProvider == nil {
		return nil, fmt.Errorf("docker client provider is not configured")
	}
	return tc.DockerClientProvider(endpoint)
}
