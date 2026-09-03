package serviceinstances

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/datastore"
	"github.com/portainer/portainer/api/filesystem"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/internal/testhelpers"
	"github.com/portainer/portainer/api/serviceinstance"
	"github.com/portainer/portainer/api/stacks/deployments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStackDeployer struct {
	deployments.StackDeployer
	deployErr   error
	undeployErr error
}

func (s *stubStackDeployer) DeployComposeStack(_ context.Context, _ *portainer.Stack, _ *portainer.Endpoint, _ []portainer.Registry, _, _, _ bool) error {
	return s.deployErr
}

func (s *stubStackDeployer) UndeployComposeStack(_ context.Context, _ *portainer.Stack, _ *portainer.Endpoint) error {
	return s.undeployErr
}

func newTestHandler(t *testing.T) (*Handler, *datastore.Store, *stubStackDeployer) {
	t.Helper()
	_, store := datastore.MustNewTestStore(t, true, false)

	user := &portainer.User{
		ID:       1,
		Username: "testadmin",
		Role:     portainer.AdministratorRole,
	}
	require.NoError(t, store.User().Create(user))

	endpoint := &portainer.Endpoint{
		ID:   1,
		Name: "test-endpoint",
		Type: portainer.DockerEnvironment,
	}
	require.NoError(t, store.Endpoint().Create(endpoint))

	fileService, err := filesystem.NewService(t.TempDir(), "")
	require.NoError(t, err)

	deployer := &stubStackDeployer{}
	svc := serviceinstance.NewService(store, fileService, deployer, nil)

	h := NewHandler(testhelpers.NewTestRequestBouncer())
	h.DataStore = store
	h.Service = svc

	return h, store, deployer
}

func newRequestWithSecurityContext(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	ctx := security.StoreRestrictedRequestContext(req, &security.RestrictedRequestContext{
		IsAdmin: true,
		UserID:  1,
		User:    &portainer.User{ID: 1, Username: "testadmin", Role: portainer.AdministratorRole},
	})
	return req.WithContext(ctx)
}

func createInstancePayload() []byte {
	payload := map[string]any{
		"Name":           "production-web",
		"Description":    "production web service",
		"TargetType":     2,
		"EnvironmentIds": []int{1},
		"ComposeFile":    "services:\n  web:\n    image: nginx:latest",
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestServiceInstanceCreate(t *testing.T) {
	h, store, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, "/service-instances", bytes.NewReader(createInstancePayload())))

	require.Equal(t, http.StatusOK, w.Code)

	var instance portainer.ServiceInstance
	err := json.Unmarshal(w.Body.Bytes(), &instance)
	require.NoError(t, err)
	assert.Equal(t, "production-web", instance.Name)
	assert.Equal(t, "si-1-production-web", instance.StackName)

	_, err = store.ServiceInstance().Read(instance.ID)
	require.NoError(t, err)
}

func TestServiceInstanceCreate_InvalidPayload(t *testing.T) {
	h, _, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, "/service-instances", bytes.NewReader([]byte(`{"Name": ""}`))))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceInstanceList(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "list-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, "/service-instances", nil))

	require.Equal(t, http.StatusOK, w.Code)

	var instances []portainer.ServiceInstance
	err := json.Unmarshal(w.Body.Bytes(), &instances)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, "list-test", instances[0].Name)
}

func TestServiceInstanceInspect(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "inspect-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, fmt.Sprintf("/service-instances/%d", instance.ID), nil))

	require.Equal(t, http.StatusOK, w.Code)

	var got portainer.ServiceInstance
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, instance.ID, got.ID)
}

func TestServiceInstanceInspect_NotFound(t *testing.T) {
	h, _, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, "/service-instances/999", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServiceInstanceUpdate(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "update-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	payload := map[string]any{
		"Name":           "updated-name",
		"Description":    "updated",
		"TargetType":     2,
		"EnvironmentIds": []int{1},
		"ComposeFile":    "services:\n  web:\n    image: nginx:latest",
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPut, fmt.Sprintf("/service-instances/%d", instance.ID), bytes.NewReader(b)))

	require.Equal(t, http.StatusOK, w.Code)

	updated, err := store.ServiceInstance().Read(instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", updated.Name)
	assert.Equal(t, "updated", updated.Description)
}

func TestServiceInstanceDelete(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "delete-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodDelete, fmt.Sprintf("/service-instances/%d", instance.ID), nil))

	require.Equal(t, http.StatusNoContent, w.Code)

	_, err := store.ServiceInstance().Read(instance.ID)
	assert.Error(t, err)
}

func TestServiceInstanceDeploy(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:           "deploy-test",
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{1},
		ComposeFile:    "services:\n  web:\n    image: nginx:latest",
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, fmt.Sprintf("/service-instances/%d/deploy", instance.ID), nil))

	require.Equal(t, http.StatusAccepted, w.Code)

	var operation portainer.ServiceInstanceOperation
	err := json.Unmarshal(w.Body.Bytes(), &operation)
	require.NoError(t, err)
	assert.Equal(t, instance.ID, operation.ServiceInstanceID)
	assert.Equal(t, portainer.ServiceInstanceOperationDeploy, operation.Type)
}

func TestServiceInstanceDeploy_NoTargets(t *testing.T) {
	h, store, _ := newTestHandler(t)

	group := &portainer.EndpointGroup{
		Name:        "empty-group",
		Description: "empty",
	}
	require.NoError(t, store.EndpointGroup().Create(group))

	instance := &portainer.ServiceInstance{
		Name:       "no-targets",
		TargetType: portainer.ServiceInstanceTargetGroup,
		GroupID:    group.ID,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, fmt.Sprintf("/service-instances/%d/deploy", instance.ID), nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceInstanceStart(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:           "start-test",
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{1},
		ComposeFile:    "services:\n  web:\n    image: nginx:latest",
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	stack := &portainer.Stack{
		ID:                1,
		Name:              instance.StackName,
		Type:              portainer.DockerComposeStack,
		EndpointID:        1,
		EntryPoint:        "docker-compose.yml",
		ServiceInstanceID: instance.ID,
	}
	require.NoError(t, store.Stack().Create(stack))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, fmt.Sprintf("/service-instances/%d/start", instance.ID), nil))

	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestServiceInstanceStop(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:           "stop-test",
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{1},
		ComposeFile:    "services:\n  web:\n    image: nginx:latest",
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	stack := &portainer.Stack{
		ID:                1,
		Name:              instance.StackName,
		Type:              portainer.DockerComposeStack,
		EndpointID:        1,
		EntryPoint:        "docker-compose.yml",
		ServiceInstanceID: instance.ID,
	}
	require.NoError(t, store.Stack().Create(stack))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, fmt.Sprintf("/service-instances/%d/stop", instance.ID), nil))

	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestServiceInstanceTargets(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:           "targets-test",
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{1},
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, fmt.Sprintf("/service-instances/%d/targets", instance.ID), nil))

	require.Equal(t, http.StatusOK, w.Code)

	var targets []serviceInstanceTargetResponse
	err := json.Unmarshal(w.Body.Bytes(), &targets)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, portainer.EndpointID(1), targets[0].EnvironmentID)
	assert.False(t, targets[0].Missing)
}

func TestServiceInstanceOperations(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "operations-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	op := &portainer.ServiceInstanceOperation{
		ServiceInstanceID: instance.ID,
		Type:              portainer.ServiceInstanceOperationDeploy,
		Status:            portainer.ServiceInstanceOperationStatusSuccess,
	}
	require.NoError(t, store.ServiceInstanceOperation().Create(op))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, fmt.Sprintf("/service-instances/%d/operations", instance.ID), nil))

	require.Equal(t, http.StatusOK, w.Code)

	var operations []portainer.ServiceInstanceOperation
	err := json.Unmarshal(w.Body.Bytes(), &operations)
	require.NoError(t, err)
	require.Len(t, operations, 1)
	assert.Equal(t, op.ID, operations[0].ID)
}

func TestServiceInstanceOperationInspect(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "op-inspect-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	op := &portainer.ServiceInstanceOperation{
		ServiceInstanceID: instance.ID,
		Type:              portainer.ServiceInstanceOperationDeploy,
		Status:            portainer.ServiceInstanceOperationStatusSuccess,
	}
	require.NoError(t, store.ServiceInstanceOperation().Create(op))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, fmt.Sprintf("/service-instance-operations/%d", op.ID), nil))

	require.Equal(t, http.StatusOK, w.Code)

	var got portainer.ServiceInstanceOperation
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, op.ID, got.ID)
}

func TestServiceInstanceScheduleBuild(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:           "schedule-test",
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{1},
		ComposeFile:    "services:\n  web:\n    image: nginx:latest",
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	payload := map[string]any{
		"ComposeFile": "services:\n  web:\n    image: nginx:latest",
		"DeployAt":    time.Now().Add(time.Hour).Unix(),
	}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, fmt.Sprintf("/service-instances/%d/schedule-build", instance.ID), bytes.NewReader(b)))

	require.Equal(t, http.StatusAccepted, w.Code)

	var build portainer.ServiceInstanceScheduledBuild
	err := json.Unmarshal(w.Body.Bytes(), &build)
	require.NoError(t, err)
	assert.Equal(t, instance.ID, build.ServiceInstanceID)
	assert.NotZero(t, build.ID)

	_, err = store.ServiceInstanceScheduledBuild().Read(build.ID)
	require.NoError(t, err)
}

func TestServiceInstanceScheduleBuild_InvalidPayload(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "schedule-invalid",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodPost, fmt.Sprintf("/service-instances/%d/schedule-build", instance.ID), bytes.NewReader([]byte(`{"ComposeFile": ""}`))))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceInstanceScheduledBuilds(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "scheduled-builds-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	build := &portainer.ServiceInstanceScheduledBuild{
		ServiceInstanceID: instance.ID,
		ComposeFile:       "services:\n  web:\n    image: nginx:latest",
		DeployAt:          time.Now().Add(time.Hour).Unix(),
		Status:            portainer.ServiceInstanceScheduledBuildStatusPending,
	}
	require.NoError(t, store.ServiceInstanceScheduledBuild().Create(build))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodGet, fmt.Sprintf("/service-instances/%d/scheduled-builds", instance.ID), nil))

	require.Equal(t, http.StatusOK, w.Code)

	var builds []portainer.ServiceInstanceScheduledBuild
	err := json.Unmarshal(w.Body.Bytes(), &builds)
	require.NoError(t, err)
	require.Len(t, builds, 1)
	assert.Equal(t, build.ID, builds[0].ID)
}

func TestServiceInstanceScheduledBuildCancel(t *testing.T) {
	h, store, _ := newTestHandler(t)

	instance := &portainer.ServiceInstance{
		Name:       "cancel-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, store.ServiceInstance().Create(instance))

	build := &portainer.ServiceInstanceScheduledBuild{
		ServiceInstanceID: instance.ID,
		ComposeFile:       "services:\n  web:\n    image: nginx:latest",
		DeployAt:          time.Now().Add(time.Hour).Unix(),
		Status:            portainer.ServiceInstanceScheduledBuildStatusPending,
	}
	require.NoError(t, store.ServiceInstanceScheduledBuild().Create(build))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequestWithSecurityContext(http.MethodDelete, fmt.Sprintf("/service-instance-scheduled-builds/%d", build.ID), nil))

	require.Equal(t, http.StatusNoContent, w.Code)

	updated, err := store.ServiceInstanceScheduledBuild().Read(build.ID)
	require.NoError(t, err)
	assert.Equal(t, portainer.ServiceInstanceScheduledBuildStatusCancelled, updated.Status)
}
