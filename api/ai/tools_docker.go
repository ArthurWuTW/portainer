package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	portainer "github.com/portainer/portainer/api"
)

const (
	maxLogBytes   = 32 * 1024
	defaultTail   = "100"
	shortIDLength = 12
)

// isDockerEndpoint reports whether the endpoint type supports the Docker API.
func isDockerEndpoint(e *portainer.Endpoint) bool {
	switch e.Type {
	case portainer.DockerEnvironment, portainer.AgentOnDockerEnvironment, portainer.EdgeAgentOnDockerEnvironment:
		return true
	}
	return false
}

// dockerEndpoint loads an authorized endpoint and verifies it is Docker-based.
func (tc *ToolContext) dockerEndpoint(id portainer.EndpointID) (*portainer.Endpoint, error) {
	endpoint, err := tc.AuthorizedEndpoint(id)
	if err != nil {
		return nil, err
	}

	if !isDockerEndpoint(endpoint) {
		return nil, fmt.Errorf("endpoint %d is a %s environment; Docker tools only support Docker environments", id, endpointTypeName(endpoint.Type))
	}

	return endpoint, nil
}

func shortID(id string) string {
	if len(id) > shortIDLength {
		return id[:shortIDLength]
	}
	return id
}

func healthFromStatus(status string) string {
	switch {
	case strings.Contains(status, "(healthy)"):
		return "healthy"
	case strings.Contains(status, "(unhealthy)"):
		return "unhealthy"
	case strings.Contains(status, "(starting)"):
		return "starting"
	default:
		return ""
	}
}

// --- get_containers ---

type getContainersTool struct{}

func (t *getContainersTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_containers",
		Description: "List containers on a Docker environment (endpoint). Returns id, name, image, state, status and health for each container.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"endpoint_id": map[string]any{
					"type":        "integer",
					"description": "The id of the environment (endpoint).",
				},
				"all": map[string]any{
					"type":        "boolean",
					"description": "Include stopped containers. Defaults to true.",
				},
			},
			"required": []string{"endpoint_id"},
		},
	}
}

type getContainersArgs struct {
	EndpointID portainer.EndpointID `json:"endpoint_id"`
	All        *bool                `json:"all"`
}

type containerSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
	Health string `json:"health,omitempty"`
}

func (t *getContainersTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getContainersArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	all := true
	if a.All != nil {
		all = *a.All
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	summaries := make([]containerSummary, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		summaries = append(summaries, containerSummary{
			ID:     shortID(c.ID),
			Name:   name,
			Image:  c.Image,
			State:  string(c.State),
			Status: c.Status,
			Health: healthFromStatus(c.Status),
		})
	}

	return marshalResult(map[string]any{"endpointId": endpoint.ID, "containers": summaries})
}

// --- get_container ---

type getContainerTool struct{}

func (t *getContainerTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_container",
		Description: "Inspect a single container on a Docker environment (endpoint). Returns state, restart count, exit code, health, ports, and redacted environment variables and labels.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"endpoint_id": map[string]any{
					"type":        "integer",
					"description": "The id of the environment (endpoint).",
				},
				"container_id": map[string]any{
					"type":        "string",
					"description": "The container id or name.",
				},
			},
			"required": []string{"endpoint_id", "container_id"},
		},
	}
}

type getContainerArgs struct {
	EndpointID  portainer.EndpointID `json:"endpoint_id"`
	ContainerID string               `json:"container_id"`
}

type containerDetail struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Image       string   `json:"image"`
	State       string   `json:"state"`
	Running     bool     `json:"running"`
	Restarting  bool     `json:"restarting,omitempty"`
	OOMKilled   bool     `json:"oomKilled,omitempty"`
	RestartCount int     `json:"restartCount"`
	ExitCode    int      `json:"exitCode,omitempty"`
	StartedAt   string   `json:"startedAt,omitempty"`
	FinishedAt  string   `json:"finishedAt,omitempty"`
	Health      string   `json:"health,omitempty"`
	Ports       []string `json:"ports,omitempty"`
	Env         []string `json:"env,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func (t *getContainerTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getContainerArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	inspect, err := cli.ContainerInspect(ctx, a.ContainerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	detail := containerDetail{
		ID:   shortID(inspect.ID),
		Name: strings.TrimPrefix(inspect.Name, "/"),
		Image: inspect.Image,
	}

	if inspect.State != nil {
		detail.State = string(inspect.State.Status)
		detail.Running = inspect.State.Running
		detail.Restarting = inspect.State.Restarting
		detail.OOMKilled = inspect.State.OOMKilled
		detail.ExitCode = inspect.State.ExitCode
		detail.StartedAt = inspect.State.StartedAt
		detail.FinishedAt = inspect.State.FinishedAt
		if inspect.State.Health != nil {
			detail.Health = string(inspect.State.Health.Status)
		}
	}

	detail.RestartCount = inspect.RestartCount
	if inspect.Config != nil {
		detail.Env = redactEnv(inspect.Config.Env)
		detail.Labels = redactLabels(inspect.Config.Labels)
	}

	if inspect.NetworkSettings != nil {
		for port, bindings := range inspect.NetworkSettings.Ports {
			for _, b := range bindings {
				detail.Ports = append(detail.Ports, fmt.Sprintf("%s -> %s:%s", port, b.HostIP, b.HostPort))
			}
		}
	}

	return marshalResult(detail)
}

// --- get_container_logs ---

type getContainerLogsTool struct{}

func (t *getContainerLogsTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_container_logs",
		Description: "Get the recent log output of a container on a Docker environment (endpoint).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"endpoint_id": map[string]any{
					"type":        "integer",
					"description": "The id of the environment (endpoint).",
				},
				"container_id": map[string]any{
					"type":        "string",
					"description": "The container id or name.",
				},
				"tail": map[string]any{
					"type":        "string",
					"description": "Number of lines to show from the end of the logs. Defaults to 100.",
				},
			},
			"required": []string{"endpoint_id", "container_id"},
		},
	}
}

type getContainerLogsArgs struct {
	EndpointID  portainer.EndpointID `json:"endpoint_id"`
	ContainerID string               `json:"container_id"`
	Tail        string               `json:"tail"`
}

func (t *getContainerLogsTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getContainerLogsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if a.Tail == "" {
		a.Tail = defaultTail
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	logs, err := cli.ContainerLogs(ctx, a.ContainerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       a.Tail,
	})
	if err != nil {
		return "", fmt.Errorf("failed to read container logs: %w", err)
	}
	defer logs.Close()

	data, err := io.ReadAll(io.LimitReader(logs, maxLogBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read container logs: %w", err)
	}

	return marshalResult(map[string]any{
		"endpointId":   endpoint.ID,
		"containerId":  shortID(a.ContainerID),
		"tail":         a.Tail,
		"truncated":    len(data) >= maxLogBytes,
		"logs":         string(data),
	})
}

// --- get_container_stats ---

type getContainerStatsTool struct{}

func (t *getContainerStatsTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_container_stats",
		Description: "Get current CPU and memory usage of a container on a Docker environment (endpoint).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"endpoint_id": map[string]any{
					"type":        "integer",
					"description": "The id of the environment (endpoint).",
				},
				"container_id": map[string]any{
					"type":        "string",
					"description": "The container id or name.",
				},
			},
			"required": []string{"endpoint_id", "container_id"},
		},
	}
}

type getContainerStatsArgs struct {
	EndpointID  portainer.EndpointID `json:"endpoint_id"`
	ContainerID string               `json:"container_id"`
}

type statsPayload struct {
	CPUStats struct {
		CPUUsage       struct {
			TotalUsage uint64 `json:"cpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage       struct {
			TotalUsage uint64 `json:"cpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
}

func (t *getContainerStatsTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getContainerStatsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	stats, err := cli.ContainerStats(ctx, a.ContainerID, false)
	if err != nil {
		return "", fmt.Errorf("failed to read container stats: %w", err)
	}
	defer stats.Body.Close()

	var p statsPayload
	if err := json.NewDecoder(io.LimitReader(stats.Body, 64*1024)).Decode(&p); err != nil {
		return "", fmt.Errorf("failed to decode container stats: %w", err)
	}

	cpuPercent := 0.0
	cpuDelta := float64(p.CPUStats.CPUUsage.TotalUsage - p.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(p.CPUStats.SystemCPUUsage - p.PreCPUStats.SystemCPUUsage)
	if cpuDelta > 0 && systemDelta > 0 {
		onlineCPUs := float64(p.CPUStats.OnlineCPUs)
		if onlineCPUs == 0 {
			onlineCPUs = 1
		}
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100
	}

	memoryPercent := 0.0
	if p.MemoryStats.Limit > 0 {
		memoryPercent = float64(p.MemoryStats.Usage) / float64(p.MemoryStats.Limit) * 100
	}

	return marshalResult(map[string]any{
		"endpointId":    endpoint.ID,
		"containerId":   shortID(a.ContainerID),
		"cpuPercent":    round2(cpuPercent),
		"memoryUsage":   p.MemoryStats.Usage,
		"memoryLimit":   p.MemoryStats.Limit,
		"memoryPercent": round2(memoryPercent),
	})
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// --- get_images ---

type getImagesTool struct{}

func (t *getImagesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_images",
		Description: "List images on a Docker environment (endpoint).",
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

type getImagesArgs struct {
	EndpointID portainer.EndpointID `json:"endpoint_id"`
}

type imageSummary struct {
	ID       string   `json:"id"`
	Tags     []string `json:"tags,omitempty"`
	Size     int64    `json:"size"`
	Containers int    `json:"containers"`
}

func (t *getImagesTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getImagesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list images: %w", err)
	}

	summaries := make([]imageSummary, 0, len(images))
	for _, img := range images {
		summaries = append(summaries, imageSummary{
			ID:         shortID(img.ID),
			Tags:       img.RepoTags,
			Size:       img.Size,
			Containers: int(img.Containers),
		})
	}

	return marshalResult(map[string]any{"endpointId": endpoint.ID, "images": summaries})
}

// --- get_networks ---

type getNetworksTool struct{}

func (t *getNetworksTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_networks",
		Description: "List networks on a Docker environment (endpoint).",
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

type getNetworksArgs struct {
	EndpointID portainer.EndpointID `json:"endpoint_id"`
}

type networkSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
}

func (t *getNetworksTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getNetworksArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	summaries := make([]networkSummary, 0, len(networks))
	for _, n := range networks {
		summaries = append(summaries, networkSummary{
			ID:     shortID(n.ID),
			Name:   n.Name,
			Driver: n.Driver,
			Scope:  n.Scope,
		})
	}

	return marshalResult(map[string]any{"endpointId": endpoint.ID, "networks": summaries})
}

// --- get_volumes ---

type getVolumesTool struct{}

func (t *getVolumesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_volumes",
		Description: "List volumes on a Docker environment (endpoint).",
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

type getVolumesArgs struct {
	EndpointID portainer.EndpointID `json:"endpoint_id"`
}

type volumeSummary struct {
	Name        string `json:"name"`
	Driver      string `json:"driver"`
	MountPoint  string `json:"mountPoint"`
}

func (t *getVolumesTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getVolumesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	volumes, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list volumes: %w", err)
	}

	summaries := make([]volumeSummary, 0, len(volumes.Volumes))
	for _, v := range volumes.Volumes {
		summaries = append(summaries, volumeSummary{
			Name:       v.Name,
			Driver:     v.Driver,
			MountPoint: v.Mountpoint,
		})
	}

	return marshalResult(map[string]any{"endpointId": endpoint.ID, "volumes": summaries})
}

// --- get_system_info ---

type getSystemInfoTool struct{}

func (t *getSystemInfoTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "get_system_info",
		Description: "Get Docker system information for a Docker environment (endpoint): OS, kernel, architecture, CPU count, memory, Docker version, and swarm status.",
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

type getSystemInfoArgs struct {
	EndpointID portainer.EndpointID `json:"endpoint_id"`
}

func (t *getSystemInfoTool) Execute(ctx context.Context, args json.RawMessage, tc *ToolContext) (string, error) {
	var a getSystemInfoArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	endpoint, err := tc.dockerEndpoint(a.EndpointID)
	if err != nil {
		return "", err
	}

	cli, err := tc.dockerClient(endpoint)
	if err != nil {
		return "", err
	}

	info, err := cli.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get system info: %w", err)
	}

	swarmStatus := "inactive"
	if info.Swarm.ControlAvailable {
		swarmStatus = "active"
	}

	result := map[string]any{
		"endpointId":      endpoint.ID,
		"operatingSystem": info.OperatingSystem,
		"kernelVersion":   info.KernelVersion,
		"architecture":    info.Architecture,
		"ncpu":            info.NCPU,
		"memTotal":        info.MemTotal,
		"dockerVersion":   info.ServerVersion,
		"driver":          info.Driver,
		"swarm":           swarmStatus,
		"containers":      info.Containers,
		"containersRunning": info.ContainersRunning,
		"images":          info.Images,
	}

	return marshalResult(result)
}
