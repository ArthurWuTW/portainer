package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/internal/testhelpers"

	"github.com/stretchr/testify/require"
)

// fakeDockerClient implements DockerAPI with canned responses.
type fakeDockerClient struct {
	containers []container.Summary
	inspect    container.InspectResponse
	logs       string
	stats      string
	images     []image.Summary
	networks   []network.Inspect
	volumes    []*volume.Volume
	info       system.Info
}

func (f *fakeDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return f.containers, nil
}

func (f *fakeDockerClient) ContainerInspect(_ context.Context, _ string) (container.InspectResponse, error) {
	return f.inspect, nil
}

func (f *fakeDockerClient) ContainerLogs(_ context.Context, _ string, _ container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte(f.logs))), nil
}

func (f *fakeDockerClient) ContainerStats(_ context.Context, _ string, _ bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{Body: io.NopCloser(bytes.NewReader([]byte(f.stats)))}, nil
}

func (f *fakeDockerClient) ImageList(_ context.Context, _ image.ListOptions) ([]image.Summary, error) {
	return f.images, nil
}

func (f *fakeDockerClient) NetworkList(_ context.Context, _ network.ListOptions) ([]network.Inspect, error) {
	return f.networks, nil
}

func (f *fakeDockerClient) VolumeList(_ context.Context, _ volume.ListOptions) (volume.ListResponse, error) {
	return volume.ListResponse{Volumes: f.volumes}, nil
}

func (f *fakeDockerClient) Info(_ context.Context) (system.Info, error) {
	return f.info, nil
}

func dockerToolContext(t *testing.T, user *portainer.User, fake *fakeDockerClient) *ToolContext {
	t.Helper()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{
		endpointWithAccess(1, 1, 2),
	}))

	return &ToolContext{
		DataStore: ds,
		DockerClientProvider: func(_ *portainer.Endpoint) (DockerAPI, error) {
			return fake, nil
		},
		User: user,
	}
}

func TestGetEndpointsTool(t *testing.T) {
	t.Parallel()

	tc := dockerToolContext(t, adminUser(), &fakeDockerClient{})

	out, err := (&getEndpointsTool{}).Execute(context.Background(), nil, tc)
	require.NoError(t, err)

	var result struct {
		Endpoints []endpointSummary `json:"endpoints"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result.Endpoints, 1)
	require.Equal(t, portainer.EndpointID(1), result.Endpoints[0].ID)
	require.Equal(t, "docker", result.Endpoints[0].Type)
}

func TestGetContainersTool(t *testing.T) {
	t.Parallel()

	fake := &fakeDockerClient{
		containers: []container.Summary{
			{
				ID:    "abc123def456",
				Names: []string{"/nginx"},
				Image: "nginx:latest",
				State: container.StateRunning,
				Status: "Up 2 hours (healthy)",
			},
		},
	}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getContainersTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.NoError(t, err)

	var result struct {
		Containers []containerSummary `json:"containers"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result.Containers, 1)
	require.Equal(t, "abc123def456", result.Containers[0].ID)
	require.Equal(t, "nginx", result.Containers[0].Name)
	require.Equal(t, "healthy", result.Containers[0].Health)
}

func TestGetContainerTool_RedactsEnv(t *testing.T) {
	t.Parallel()

	fake := &fakeDockerClient{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:           "abc123def456",
				Name:         "/nginx",
				Image:        "sha256:deadbeef",
				RestartCount: 5,
				State: &container.State{
					Status:   "running",
					Running:  true,
					ExitCode: 0,
					Health:   &container.Health{Status: "healthy"},
				},
			},
			Config: &container.Config{
				Env: []string{
					"PATH=/usr/local/sbin:/usr/local/bin",
					"DATABASE_PASSWORD=supersecret",
					"API_KEY=abc123",
					"NGINX_PORT=80",
				},
				Labels: map[string]string{
					"com.docker.traffic": "80",
					"db_password":        "hunter2",
				},
			},
		},
	}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getContainerTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1,"container_id":"nginx"}`), tc)
	require.NoError(t, err)

	var result containerDetail
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Equal(t, "nginx", result.Name)
	require.Equal(t, 5, result.RestartCount)
	require.Equal(t, "healthy", result.Health)
	require.Equal(t, []string{
		"PATH=/usr/local/sbin:/usr/local/bin",
		"DATABASE_PASSWORD=" + redactedValue,
		"API_KEY=" + redactedValue,
		"NGINX_PORT=80",
	}, result.Env)
	require.Equal(t, redactedValue, result.Labels["db_password"])
	require.Equal(t, "80", result.Labels["com.docker.traffic"])
}

func TestGetContainerLogsTool(t *testing.T) {
	t.Parallel()

	fake := &fakeDockerClient{logs: "line1\nline2\nline3"}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getContainerLogsTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1,"container_id":"nginx"}`), tc)
	require.NoError(t, err)

	var result struct {
		Logs      string `json:"logs"`
		Truncated bool   `json:"truncated"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Equal(t, "line1\nline2\nline3", result.Logs)
	require.False(t, result.Truncated)
}

func TestGetContainerStatsTool(t *testing.T) {
	t.Parallel()

	statsJSON := `{
		"cpu_stats": {"cpu_usage": {"cpu_usage": 200}, "system_cpu_usage": 1000, "online_cpus": 4},
		"precpu_stats": {"cpu_usage": {"cpu_usage": 100}, "system_cpu_usage": 500},
		"memory_stats": {"usage": 104857600, "limit": 2147483648}
	}`

	fake := &fakeDockerClient{stats: statsJSON}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getContainerStatsTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1,"container_id":"nginx"}`), tc)
	require.NoError(t, err)

	var result struct {
		CPUPercent    float64 `json:"cpuPercent"`
		MemoryUsage   uint64  `json:"memoryUsage"`
		MemoryLimit   uint64  `json:"memoryLimit"`
		MemoryPercent float64 `json:"memoryPercent"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	// cpuDelta=100, systemDelta=500, onlineCPUs=4 -> 100/500*4*100 = 80%
	require.Equal(t, 80.0, result.CPUPercent)
	require.Equal(t, uint64(104857600), result.MemoryUsage)
	require.Equal(t, uint64(2147483648), result.MemoryLimit)
	require.Equal(t, 4.88, result.MemoryPercent)
}

func TestGetImagesTool(t *testing.T) {
	t.Parallel()

	fake := &fakeDockerClient{
		images: []image.Summary{
			{ID: "sha256:abc123def456", RepoTags: []string{"nginx:latest"}, Size: 180000000, Containers: 2},
		},
	}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getImagesTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.NoError(t, err)

	var result struct {
		Images []imageSummary `json:"images"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result.Images, 1)
	require.Equal(t, []string{"nginx:latest"}, result.Images[0].Tags)
}

func TestGetNetworksTool(t *testing.T) {
	t.Parallel()

	fake := &fakeDockerClient{
		networks: []network.Inspect{
			{ID: "net123", Name: "bridge", Driver: "bridge", Scope: "local"},
		},
	}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getNetworksTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.NoError(t, err)

	var result struct {
		Networks []networkSummary `json:"networks"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result.Networks, 1)
	require.Equal(t, "bridge", result.Networks[0].Name)
}

func TestGetVolumesTool(t *testing.T) {
	t.Parallel()

	vol := volume.Volume{Name: "data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/data"}
	fake := &fakeDockerClient{
		volumes: []*volume.Volume{&vol},
	}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getVolumesTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.NoError(t, err)

	var result struct {
		Volumes []volumeSummary `json:"volumes"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result.Volumes, 1)
	require.Equal(t, "data", result.Volumes[0].Name)
}

func TestGetSystemInfoTool(t *testing.T) {
	t.Parallel()

	fake := &fakeDockerClient{
		info: system.Info{
			OperatingSystem: "Ubuntu 24.04",
			KernelVersion:   "6.8.0",
			Architecture:    "x86_64",
			NCPU:            8,
			MemTotal:        16000000000,
			ServerVersion:   "27.1.0",
			Driver:          "overlay2",
			Containers:      10,
			ContainersRunning: 5,
			Images:          20,
		},
	}

	tc := dockerToolContext(t, adminUser(), fake)

	out, err := (&getSystemInfoTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Equal(t, "Ubuntu 24.04", result["operatingSystem"])
	require.Equal(t, float64(8), result["ncpu"])
	require.Equal(t, "27.1.0", result["dockerVersion"])
	require.Equal(t, "inactive", result["swarm"])
}

func TestDockerTool_DeniedForStandardUser(t *testing.T) {
	t.Parallel()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{
		endpointWithAccess(1, 3), // only user 3
	}))

	tc := &ToolContext{
		DataStore: ds,
		DockerClientProvider: func(_ *portainer.Endpoint) (DockerAPI, error) {
			t.Fatal("docker client should not be created for unauthorized user")
			return nil, nil
		},
		User: standardUser(),
	}

	_, err := (&getContainersTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestDockerTool_NonDockerEndpoint(t *testing.T) {
	t.Parallel()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{
		{
			ID:   1,
			Name: "k8s",
			Type: portainer.KubernetesLocalEnvironment,
		},
	}))

	tc := &ToolContext{
		DataStore: ds,
		DockerClientProvider: func(_ *portainer.Endpoint) (DockerAPI, error) {
			t.Fatal("docker client should not be created for kubernetes endpoint")
			return nil, nil
		},
		User: adminUser(),
	}

	_, err := (&getContainersTool{}).Execute(context.Background(), []byte(`{"endpoint_id":1}`), tc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Docker tools only support Docker environments")
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	t.Parallel()

	_, err := ExecuteTool(context.Background(), NewTools(), "does_not_exist", nil, &ToolContext{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown tool")
}
