package serviceinstance

import (
	"context"
	"fmt"
	"testing"
	"time"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/datastore"
	"github.com/portainer/portainer/api/filesystem"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/stacks/deployments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubStackDeployer implements deployments.StackDeployer for testing.
type stubStackDeployer struct {
	deployments.StackDeployer
	deployFunc     func(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint, registries []portainer.Registry, prune, forcePullImage, forceRecreate bool) error
	undeployFunc   func(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint) error
	deployedStacks []*portainer.Stack
}

func (s *stubStackDeployer) DeployComposeStack(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint, registries []portainer.Registry, prune, forcePullImage, forceRecreate bool) error {
	if s.deployFunc != nil {
		return s.deployFunc(ctx, stack, endpoint, registries, prune, forcePullImage, forceRecreate)
	}
	s.deployedStacks = append(s.deployedStacks, stack)
	return nil
}

func (s *stubStackDeployer) UndeployComposeStack(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint) error {
	if s.undeployFunc != nil {
		return s.undeployFunc(ctx, stack, endpoint)
	}
	return nil
}

func newTestService(t *testing.T) (*Service, *datastore.Store) {
	t.Helper()
	_, store := datastore.MustNewTestStore(t, true, false)

	fileService, err := filesystem.NewService(t.TempDir(), "")
	require.NoError(t, err)

	deployer := &stubStackDeployer{}
	svc := NewService(store, fileService, deployer)
	return svc, store
}

func createTestUser(t *testing.T, store *datastore.Store) *portainer.User {
	t.Helper()
	user := &portainer.User{
		ID:       1,
		Username: "testadmin",
		Role:     portainer.AdministratorRole,
	}
	require.NoError(t, store.User().Create(user))
	return user
}

func createTestEndpoints(t *testing.T, store *datastore.Store, groupID portainer.EndpointGroupID, count int) []portainer.Endpoint {
	t.Helper()
	endpoints := make([]portainer.Endpoint, 0, count)
	for i := 1; i <= count; i++ {
		endpoint := &portainer.Endpoint{
			ID:      portainer.EndpointID(i),
			Name:    fmt.Sprintf("endpoint-%d", i),
			Type:    portainer.DockerEnvironment,
			GroupID: groupID,
		}
		require.NoError(t, store.Endpoint().Create(endpoint))
		endpoints = append(endpoints, *endpoint)
	}
	return endpoints
}

func adminSecurityContext() *security.RestrictedRequestContext {
	return &security.RestrictedRequestContext{
		IsAdmin: true,
		UserID:  1,
		User:    &portainer.User{ID: 1, Username: "testadmin", Role: portainer.AdministratorRole},
	}
}

func TestStackName(t *testing.T) {
	tests := []struct {
		id   portainer.ServiceInstanceID
		name string
		want string
	}{
		{1, "production-web", "si-1-production-web"},
		{102, "My Service", "si-102-myservice"},
		{5, "test", "si-5-test"},
		{3, "a_b_c", "si-3-a_b_c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StackName(tt.id, tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAggregateStatus(t *testing.T) {
	tests := []struct {
		name     string
		statuses []portainer.StackStatus
		want     portainer.ServiceInstanceStatus
	}{
		{"empty", nil, portainer.ServiceInstanceStatusUnknown},
		{"all running", []portainer.StackStatus{portainer.StackStatusActive, portainer.StackStatusActive}, portainer.ServiceInstanceStatusRunning},
		{"all stopped", []portainer.StackStatus{portainer.StackStatusInactive, portainer.StackStatusInactive}, portainer.ServiceInstanceStatusStopped},
		{"all failed", []portainer.StackStatus{portainer.StackStatusError, portainer.StackStatusError}, portainer.ServiceInstanceStatusFailed},
		{"mixed running and stopped", []portainer.StackStatus{portainer.StackStatusActive, portainer.StackStatusInactive}, portainer.ServiceInstanceStatusPartial},
		{"deploying takes priority", []portainer.StackStatus{portainer.StackStatusActive, portainer.StackStatusDeploying}, portainer.ServiceInstanceStatusDeploying},
		{"failed and running", []portainer.StackStatus{portainer.StackStatusError, portainer.StackStatusActive}, portainer.ServiceInstanceStatusPartial},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateStatus(tt.statuses)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreate(t *testing.T) {
	svc, store := newTestService(t)
	createTestUser(t, store)

	instance := &portainer.ServiceInstance{
		Name:           "test-instance",
		Description:    "A test instance",
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{1},
		ComposeFile:    "services:\n  web:\n    image: nginx",
	}

	err := svc.Create(instance)
	require.NoError(t, err)
	assert.NotZero(t, instance.ID)
	assert.Equal(t, "si-1-test-instance", instance.StackName)
	assert.Equal(t, portainer.ServiceInstanceStatusUnknown, instance.Status)
	assert.NotZero(t, instance.CreatedAt)
	assert.NotZero(t, instance.UpdatedAt)

	// Verify it was persisted
	read, err := store.ServiceInstance().Read(instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-instance", read.Name)
}

func TestResolveTargetsGroup(t *testing.T) {
	svc, store := newTestService(t)
	createTestUser(t, store)

	// Create a group
	group := &portainer.EndpointGroup{
		Name:        "test-group",
		Description: "Test group",
	}
	require.NoError(t, store.EndpointGroup().Create(group))

	// Create endpoints in the group
	createTestEndpoints(t, store, group.ID, 3)

	instance := &portainer.ServiceInstance{
		TargetType: portainer.ServiceInstanceTargetGroup,
		GroupID:    group.ID,
	}

	targets, err := svc.ResolveTargets(instance)
	require.NoError(t, err)
	assert.Len(t, targets.Endpoints, 3)
	assert.Empty(t, targets.Missing)
}

func TestResolveTargetsEnvironments(t *testing.T) {
	svc, store := newTestService(t)
	createTestUser(t, store)

	endpoints := createTestEndpoints(t, store, 1, 2)

	instance := &portainer.ServiceInstance{
		TargetType:     portainer.ServiceInstanceTargetEnvironments,
		EnvironmentIDs: []portainer.EndpointID{endpoints[0].ID, endpoints[1].ID, 999},
	}

	targets, err := svc.ResolveTargets(instance)
	require.NoError(t, err)
	assert.Len(t, targets.Endpoints, 2)
	assert.Equal(t, []portainer.EndpointID{999}, targets.Missing)
}

func TestExecuteOperation_NoTargets(t *testing.T) {
	svc, store := newTestService(t)
	createTestUser(t, store)

	// Create an empty group
	group := &portainer.EndpointGroup{
		Name:        "empty-group",
		Description: "Empty group",
	}
	require.NoError(t, store.EndpointGroup().Create(group))

	instance := &portainer.ServiceInstance{
		Name:       "no-targets",
		TargetType: portainer.ServiceInstanceTargetGroup,
		GroupID:    group.ID,
	}
	require.NoError(t, svc.Create(instance))

	_, err := svc.ExecuteOperation(context.Background(), instance.ID, portainer.ServiceInstanceOperationDeploy, adminSecurityContext())
	assert.ErrorIs(t, err, ErrNoDeploymentTargets)
}

func TestExecuteOperation_ConcurrencyGuard(t *testing.T) {
	_, store := datastore.MustNewTestStore(t, true, false)
	createTestUser(t, store)

	fileService, err := filesystem.NewService(t.TempDir(), "")
	require.NoError(t, err)

	// Use a deployer that blocks on a channel
	blockChan := make(chan struct{})
	deployer := &stubStackDeployer{
		deployFunc: func(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint, registries []portainer.Registry, prune, forcePullImage, forceRecreate bool) error {
			<-blockChan
			return nil
		},
	}

	svc := NewService(store, fileService, deployer)

	group := &portainer.EndpointGroup{
		Name:        "guard-group",
		Description: "Guard test group",
	}
	require.NoError(t, store.EndpointGroup().Create(group))
	createTestEndpoints(t, store, group.ID, 1)

	instance := &portainer.ServiceInstance{
		Name:       "guard-test",
		TargetType: portainer.ServiceInstanceTargetGroup,
		GroupID:    group.ID,
	}
	require.NoError(t, svc.Create(instance))

	// Start first operation (will block in the deployer)
	_, err = svc.ExecuteOperation(context.Background(), instance.ID, portainer.ServiceInstanceOperationDeploy, adminSecurityContext())
	require.NoError(t, err)

	// Second operation should be rejected
	_, err = svc.ExecuteOperation(context.Background(), instance.ID, portainer.ServiceInstanceOperationDeploy, adminSecurityContext())
	assert.ErrorIs(t, err, ErrOperationInProgress)

	// Release the first operation
	close(blockChan)
	time.Sleep(100 * time.Millisecond)

	// Now a new operation should be allowed
	_, err = svc.ExecuteOperation(context.Background(), instance.ID, portainer.ServiceInstanceOperationDeploy, adminSecurityContext())
	assert.NoError(t, err)
}

func TestExecuteOperation_PartialFailure(t *testing.T) {
	_, store := datastore.MustNewTestStore(t, true, false)
	createTestUser(t, store)

	fileService, err := filesystem.NewService(t.TempDir(), "")
	require.NoError(t, err)

	// Fail on the second endpoint (ID 2)
	deployer := &stubStackDeployer{
		deployFunc: func(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint, registries []portainer.Registry, prune, forcePullImage, forceRecreate bool) error {
			if endpoint.ID == 2 {
				return fmt.Errorf("simulated deploy failure")
			}
			return nil
		},
	}

	svc := NewService(store, fileService, deployer)

	group := &portainer.EndpointGroup{
		Name:        "partial-group",
		Description: "Partial failure test group",
	}
	require.NoError(t, store.EndpointGroup().Create(group))
	createTestEndpoints(t, store, group.ID, 3)

	instance := &portainer.ServiceInstance{
		Name:       "partial-test",
		TargetType: portainer.ServiceInstanceTargetGroup,
		GroupID:    group.ID,
	}
	require.NoError(t, svc.Create(instance))

	operation, err := svc.ExecuteOperation(context.Background(), instance.ID, portainer.ServiceInstanceOperationDeploy, adminSecurityContext())
	require.NoError(t, err)

	// Wait for the operation to complete
	require.Eventually(t, func() bool {
		op, readErr := store.ServiceInstanceOperation().Read(operation.ID)
		return readErr == nil && op.Status != portainer.ServiceInstanceOperationStatusRunning
	}, 5*time.Second, 50*time.Millisecond)

	op, err := store.ServiceInstanceOperation().Read(operation.ID)
	require.NoError(t, err)
	assert.Equal(t, portainer.ServiceInstanceOperationStatusPartialSuccess, op.Status)
	require.Len(t, op.Results, 3)
	assert.Equal(t, portainer.ServiceInstanceTargetStatusSuccess, op.Results[0].Status)
	assert.Equal(t, portainer.ServiceInstanceTargetStatusFailed, op.Results[1].Status)
	assert.Contains(t, op.Results[1].Error, "simulated deploy failure")
	assert.Equal(t, portainer.ServiceInstanceTargetStatusSkipped, op.Results[2].Status)
}

func TestDelete(t *testing.T) {
	svc, store := newTestService(t)
	createTestUser(t, store)

	instance := &portainer.ServiceInstance{
		Name:       "delete-test",
		TargetType: portainer.ServiceInstanceTargetEnvironments,
	}
	require.NoError(t, svc.Create(instance))

	// Create an operation
	op := &portainer.ServiceInstanceOperation{
		ServiceInstanceID: instance.ID,
		Type:              portainer.ServiceInstanceOperationDeploy,
		Status:            portainer.ServiceInstanceOperationStatusSuccess,
	}
	require.NoError(t, store.ServiceInstanceOperation().Create(op))

	// Delete the instance
	err := svc.Delete(instance.ID)
	require.NoError(t, err)

	// Verify instance is gone
	_, err = store.ServiceInstance().Read(instance.ID)
	assert.Error(t, err)

	// Verify operation is gone
	_, err = store.ServiceInstanceOperation().Read(op.ID)
	assert.Error(t, err)
}
