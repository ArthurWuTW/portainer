// Package serviceinstance implements the Service Instance orchestration layer.
//
// A Service Instance is a logical object that groups a set of target
// environments (either an environment group or an explicit list of
// environments) and deploys a shared compose definition to all of them by
// orchestrating the existing Portainer stack deployment machinery.
package serviceinstance

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
	dserrors "github.com/portainer/portainer/api/dataservices/errors"
	"github.com/portainer/portainer/api/filesystem"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/stacks/deployments"
	"github.com/portainer/portainer/api/stacks/stackutils"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrOperationInProgress is returned when a lifecycle operation is already
	// running for the same service instance.
	ErrOperationInProgress = errors.New("a service instance operation is already in progress")
	// ErrNoDeploymentTargets is returned when the resolved target list is empty.
	ErrNoDeploymentTargets = errors.New("no deployment targets")
	// ErrStackOwnershipConflict is returned when a stack with the target name
	// already exists on an environment but is not managed by this service instance.
	ErrStackOwnershipConflict = errors.New("a stack with the same name exists and is not managed by this service instance")
)

// ResolvedTargets is the result of resolving a service instance's deployment
// targets. Missing lists the environment IDs that could not be resolved (e.g.
// the environment was deleted).
type ResolvedTargets struct {
	Endpoints []portainer.Endpoint
	Missing   []portainer.EndpointID
}

// Service manages service instances and their lifecycle operations.
type Service struct {
	dataStore     dataservices.DataStore
	fileService   portainer.FileService
	stackDeployer deployments.StackDeployer

	mu      sync.Mutex
	running map[portainer.ServiceInstanceID]bool
}

// NewService creates a new Service.
func NewService(dataStore dataservices.DataStore, fileService portainer.FileService, stackDeployer deployments.StackDeployer) *Service {
	return &Service{
		dataStore:     dataStore,
		fileService:   fileService,
		stackDeployer: stackDeployer,
		running:       make(map[portainer.ServiceInstanceID]bool),
	}
}

// stackNameSanitizeRegex matches the normalization used by the compose/swarm
// stack managers (see exec.normalizeStackName): lowercase and drop any
// character that is not a letter, digit, dash or underscore.
var stackNameSanitizeRegex = regexp.MustCompile(`[^-_a-z0-9]+`)

// StackName returns the deterministic stack name used on each target
// environment for the given service instance.
func StackName(id portainer.ServiceInstanceID, name string) string {
	sanitized := stackNameSanitizeRegex.ReplaceAllString(strings.ToLower(name), "")
	return fmt.Sprintf("si-%d-%s", int(id), sanitized)
}

// Create validates and persists a new service instance.
func (s *Service) Create(instance *portainer.ServiceInstance) error {
	if instance.ID == 0 {
		instance.ID = portainer.ServiceInstanceID(s.dataStore.ServiceInstance().GetNextIdentifier())
	}

	if instance.StackName == "" {
		instance.StackName = StackName(instance.ID, instance.Name)
	}

	now := time.Now().Unix()
	instance.CreatedAt = now
	instance.UpdatedAt = now
	instance.Status = portainer.ServiceInstanceStatusUnknown

	return s.dataStore.ServiceInstance().Create(instance)
}

// Update persists changes to an existing service instance.
func (s *Service) Update(id portainer.ServiceInstanceID, instance *portainer.ServiceInstance) error {
	instance.ID = id
	instance.UpdatedAt = time.Now().Unix()

	return s.dataStore.ServiceInstance().Update(id, instance)
}

// Delete removes a service instance and its operation history. Stacks that
// were deployed on target environments are left in place (they become
// regular stacks) to avoid destructive side effects.
func (s *Service) Delete(id portainer.ServiceInstanceID) error {
	operations, err := s.dataStore.ServiceInstanceOperation().ReadAllByServiceInstanceID(id)
	if err != nil {
		return err
	}

	for _, operation := range operations {
		if err := s.dataStore.ServiceInstanceOperation().Delete(operation.ID); err != nil {
			log.Warn().Err(err).Int("operation_id", int(operation.ID)).Msg("unable to delete service instance operation")
		}
	}

	return s.dataStore.ServiceInstance().Delete(id)
}

// Read returns a service instance by ID.
func (s *Service) Read(id portainer.ServiceInstanceID) (*portainer.ServiceInstance, error) {
	return s.dataStore.ServiceInstance().Read(id)
}

// ReadAll returns all service instances.
func (s *Service) ReadAll() ([]portainer.ServiceInstance, error) {
	return s.dataStore.ServiceInstance().ReadAll()
}

// ResolveTargets resolves the deployment targets of a service instance.
// For group targets the resolution is dynamic: the current group membership
// is used. For explicit environment targets, missing environments are
// reported in ResolvedTargets.Missing instead of failing the whole operation.
func (s *Service) ResolveTargets(instance *portainer.ServiceInstance) (*ResolvedTargets, error) {
	switch instance.TargetType {
	case portainer.ServiceInstanceTargetGroup:
		endpoints, err := s.dataStore.Endpoint().ReadAll(func(e portainer.Endpoint) bool {
			return e.GroupID == instance.GroupID
		})
		if err != nil {
			return nil, err
		}
		return &ResolvedTargets{Endpoints: endpoints}, nil
	case portainer.ServiceInstanceTargetEnvironments:
		resolved := &ResolvedTargets{}
		for _, id := range instance.EnvironmentIDs {
			endpoint, err := s.dataStore.Endpoint().Endpoint(id)
			if err != nil {
				if dataservices.IsErrObjectNotFound(err) {
					resolved.Missing = append(resolved.Missing, id)
					continue
				}
				return nil, err
			}
			resolved.Endpoints = append(resolved.Endpoints, *endpoint)
		}
		return resolved, nil
	default:
		return nil, errors.New("unknown service instance target type")
	}
}

// AggregateStatus computes the service instance status from the per-target
// stack statuses.
func AggregateStatus(statuses []portainer.StackStatus) portainer.ServiceInstanceStatus {
	if len(statuses) == 0 {
		return portainer.ServiceInstanceStatusUnknown
	}

	deploying := false
	running := 0
	stopped := 0
	failed := 0

	for _, status := range statuses {
		switch status {
		case portainer.StackStatusDeploying:
			deploying = true
		case portainer.StackStatusActive:
			running++
		case portainer.StackStatusInactive:
			stopped++
		case portainer.StackStatusError:
			failed++
		}
	}

	if deploying {
		return portainer.ServiceInstanceStatusDeploying
	}

	total := len(statuses)
	switch {
	case running == total:
		return portainer.ServiceInstanceStatusRunning
	case stopped == total:
		return portainer.ServiceInstanceStatusStopped
	case failed == total:
		return portainer.ServiceInstanceStatusFailed
	default:
		return portainer.ServiceInstanceStatusPartial
	}
}

// RefreshStatus recomputes the aggregated status of the service instance from
// the current status of its target stacks and persists it. It is synchronous
// and lightweight (no deployment is performed).
func (s *Service) RefreshStatus(instanceID portainer.ServiceInstanceID) (*portainer.ServiceInstance, error) {
	instance, err := s.dataStore.ServiceInstance().Read(instanceID)
	if err != nil {
		return nil, err
	}

	statuses, err := s.targetStackStatuses(instance)
	if err != nil {
		return nil, err
	}

	instance.Status = AggregateStatus(statuses)
	instance.UpdatedAt = time.Now().Unix()

	if err := s.dataStore.ServiceInstance().Update(instance.ID, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// HasRunningOperation reports whether a lifecycle operation is currently
// running for the given service instance.
func (s *Service) HasRunningOperation(id portainer.ServiceInstanceID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[id]
}

// ExecuteOperation starts a lifecycle operation (deploy, start, stop,
// redeploy, refresh) against all resolved targets of a service instance.
//
// The operation is executed asynchronously: a record is created and returned
// immediately, and the per-target execution happens in a background
// goroutine. Targets are processed sequentially and the operation stops at
// the first failure (fail-fast); remaining targets are marked as skipped.
func (s *Service) ExecuteOperation(ctx context.Context, instanceID portainer.ServiceInstanceID, operationType portainer.ServiceInstanceOperationType, securityContext *security.RestrictedRequestContext) (*portainer.ServiceInstanceOperation, error) {
	s.mu.Lock()
	if s.running[instanceID] {
		s.mu.Unlock()
		return nil, ErrOperationInProgress
	}
	s.running[instanceID] = true
	s.mu.Unlock()

	instance, err := s.dataStore.ServiceInstance().Read(instanceID)
	if err != nil {
		s.releaseOperation(instanceID)
		return nil, err
	}

	targets, err := s.ResolveTargets(instance)
	if err != nil {
		s.releaseOperation(instanceID)
		return nil, err
	}
	if len(targets.Endpoints) == 0 {
		s.releaseOperation(instanceID)
		return nil, ErrNoDeploymentTargets
	}

	operation := &portainer.ServiceInstanceOperation{
		ID:                portainer.ServiceInstanceOperationID(s.dataStore.ServiceInstanceOperation().GetNextIdentifier()),
		ServiceInstanceID: instanceID,
		Type:              operationType,
		Status:            portainer.ServiceInstanceOperationStatusRunning,
		UserID:            securityContext.UserID,
		StartedAt:         time.Now().Unix(),
		Results:           make([]portainer.ServiceInstanceTargetResult, 0, len(targets.Endpoints)),
	}

	for _, endpoint := range targets.Endpoints {
		operation.Results = append(operation.Results, portainer.ServiceInstanceTargetResult{
			EnvironmentID: endpoint.ID,
			Status:        portainer.ServiceInstanceTargetStatusPending,
		})
	}

	if err := s.dataStore.ServiceInstanceOperation().Create(operation); err != nil {
		s.releaseOperation(instanceID)
		return nil, err
	}

	go s.runOperation(instance, operation, targets.Endpoints, securityContext)

	return operation, nil
}

func (s *Service) releaseOperation(instanceID portainer.ServiceInstanceID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, instanceID)
}

// runOperation executes the operation against each target sequentially with
// fail-fast semantics and persists the results as they happen.
func (s *Service) runOperation(instance *portainer.ServiceInstance, operation *portainer.ServiceInstanceOperation, endpoints []portainer.Endpoint, securityContext *security.RestrictedRequestContext) {
	defer s.releaseOperation(instance.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	failed := false
	for i, endpoint := range endpoints {
		if failed {
			operation.Results[i].Status = portainer.ServiceInstanceTargetStatusSkipped
			continue
		}

		operation.Results[i].Status = portainer.ServiceInstanceTargetStatusRunning
		s.persistOperation(operation)

		err := s.executeOnTarget(ctx, instance, &endpoint, operation.Type, securityContext)
		if err != nil {
			operation.Results[i].Status = portainer.ServiceInstanceTargetStatusFailed
			operation.Results[i].Error = err.Error()
			failed = true
		} else {
			operation.Results[i].Status = portainer.ServiceInstanceTargetStatusSuccess
		}

		s.persistOperation(operation)
	}

	s.finalizeOperation(instance, operation)
}

func (s *Service) persistOperation(operation *portainer.ServiceInstanceOperation) {
	if err := s.dataStore.ServiceInstanceOperation().Update(operation.ID, operation); err != nil {
		log.Warn().Err(err).Int("operation_id", int(operation.ID)).Msg("unable to persist service instance operation")
	}
}

func (s *Service) finalizeOperation(instance *portainer.ServiceInstance, operation *portainer.ServiceInstanceOperation) {
	successCount := 0
	failureCount := 0
	for _, result := range operation.Results {
		switch result.Status {
		case portainer.ServiceInstanceTargetStatusSuccess:
			successCount++
		case portainer.ServiceInstanceTargetStatusFailed:
			failureCount++
		}
	}

	switch {
	case failureCount == 0:
		operation.Status = portainer.ServiceInstanceOperationStatusSuccess
	case successCount > 0:
		operation.Status = portainer.ServiceInstanceOperationStatusPartialSuccess
	default:
		operation.Status = portainer.ServiceInstanceOperationStatusFailed
	}

	now := time.Now().Unix()
	operation.FinishedAt = &now

	if err := s.dataStore.ServiceInstanceOperation().Update(operation.ID, operation); err != nil {
		log.Warn().Err(err).Int("operation_id", int(operation.ID)).Msg("unable to persist final service instance operation status")
	}

	// Refresh the aggregated instance status from the target stacks.
	statuses, err := s.targetStackStatuses(instance)
	if err == nil {
		instance.Status = AggregateStatus(statuses)
		instance.UpdatedAt = now
		if err := s.dataStore.ServiceInstance().Update(instance.ID, instance); err != nil {
			log.Warn().Err(err).Int("service_instance_id", int(instance.ID)).Msg("unable to update service instance status")
		}
	}

	log.Info().
		Int("service_instance_id", int(instance.ID)).
		Int("operation_id", int(operation.ID)).
		Str("operation_status", operationStatusName(operation.Status)).
		Msg("service instance operation finished")
}

// targetStackStatuses returns the current status of the stacks managed by the
// service instance on each of its target environments.
func (s *Service) targetStackStatuses(instance *portainer.ServiceInstance) ([]portainer.StackStatus, error) {
	targets, err := s.ResolveTargets(instance)
	if err != nil {
		return nil, err
	}

	statuses := make([]portainer.StackStatus, 0, len(targets.Endpoints))
	for _, endpoint := range targets.Endpoints {
		stack, err := s.findStackOnEndpoint(instance, endpoint.ID)
		if err != nil {
			if errors.Is(err, dserrors.ErrObjectNotFound) {
				// Stack not deployed on this target yet: treat as stopped.
				statuses = append(statuses, portainer.StackStatusInactive)
				continue
			}
			return nil, err
		}
		statuses = append(statuses, stack.Status)
	}

	return statuses, nil
}

// FindStackOnEndpoint returns the stack managed by the service instance on
// the given environment. It returns an ownership conflict error when a stack
// with the same name exists but is not managed by this service instance.
func (s *Service) FindStackOnEndpoint(instance *portainer.ServiceInstance, endpointID portainer.EndpointID) (*portainer.Stack, error) {
	return s.findStackOnEndpoint(instance, endpointID)
}

// findStackOnEndpoint returns the stack managed by the service instance on
// the given environment. It returns an ownership conflict error when a stack
// with the same name exists but is not managed by this service instance.
func (s *Service) findStackOnEndpoint(instance *portainer.ServiceInstance, endpointID portainer.EndpointID) (*portainer.Stack, error) {
	stacks, err := s.dataStore.Stack().StacksByName(instance.StackName)
	if err != nil {
		return nil, err
	}

	for i := range stacks {
		stack := &stacks[i]
		if stack.EndpointID != endpointID {
			continue
		}
		if stack.ServiceInstanceID != instance.ID {
			return nil, ErrStackOwnershipConflict
		}
		return stack, nil
	}

	return nil, dserrors.ErrObjectNotFound
}

// executeOnTarget runs a single operation against one target environment by
// orchestrating the existing stack deployment machinery.
func (s *Service) executeOnTarget(ctx context.Context, instance *portainer.ServiceInstance, endpoint *portainer.Endpoint, operationType portainer.ServiceInstanceOperationType, securityContext *security.RestrictedRequestContext) error {
	switch operationType {
	case portainer.ServiceInstanceOperationDeploy, portainer.ServiceInstanceOperationRedeploy:
		return s.deployTarget(ctx, instance, endpoint, securityContext)
	case portainer.ServiceInstanceOperationStart:
		stack, err := s.findStackOnEndpoint(instance, endpoint.ID)
		if err != nil {
			return err
		}
		return s.deployStack(ctx, stack, endpoint, securityContext, false)
	case portainer.ServiceInstanceOperationStop:
		stack, err := s.findStackOnEndpoint(instance, endpoint.ID)
		if err != nil {
			return err
		}
		return s.undeployStack(ctx, stack, endpoint)
	case portainer.ServiceInstanceOperationRefresh:
		_, err := s.findStackOnEndpoint(instance, endpoint.ID)
		return err
	default:
		return errors.New("unknown service instance operation type")
	}
}

// deployTarget writes the desired compose definition to disk, ensures a stack
// record exists on the target environment, and deploys it.
func (s *Service) deployTarget(ctx context.Context, instance *portainer.ServiceInstance, endpoint *portainer.Endpoint, securityContext *security.RestrictedRequestContext) error {
	stack, err := s.findStackOnEndpoint(instance, endpoint.ID)
	if err != nil && !errors.Is(err, dserrors.ErrObjectNotFound) {
		return err
	}

	if stack == nil {
		stack, err = s.createStackRecord(instance, endpoint, securityContext)
		if err != nil {
			return err
		}
	}

	if err := s.storeComposeFile(instance, stack); err != nil {
		return err
	}

	return s.deployStack(ctx, stack, endpoint, securityContext, true)
}

// createStackRecord creates the stack record on the target environment.
func (s *Service) createStackRecord(instance *portainer.ServiceInstance, endpoint *portainer.Endpoint, securityContext *security.RestrictedRequestContext) (*portainer.Stack, error) {
	stack := &portainer.Stack{
		ID:                portainer.StackID(s.dataStore.Stack().GetNextIdentifier()),
		Name:              instance.StackName,
		Type:              portainer.DockerComposeStack,
		EndpointID:        endpoint.ID,
		EntryPoint:        filesystem.ComposeFileDefaultName,
		Env:               instance.Env,
		ServiceInstanceID: instance.ID,
	}
	if securityContext.User != nil {
		stack.CreatedBy = securityContext.User.Username
	}
	stack.CreationDate = time.Now().Unix()
	stackutils.PrepareStackStatusForDeployment(stack)

	if err := s.dataStore.Stack().Create(stack); err != nil {
		return nil, fmt.Errorf("unable to create stack record: %w", err)
	}

	return stack, nil
}

// storeComposeFile writes the desired compose definition to the stack project
// path on disk.
func (s *Service) storeComposeFile(instance *portainer.ServiceInstance, stack *portainer.Stack) error {
	stackFolder := fmt.Sprintf("%d", int(stack.ID))
	var projectPath string
	var err error
	if stack.ProjectPath == "" {
		projectPath, err = s.fileService.StoreStackFileFromBytes(stackFolder, stack.EntryPoint, []byte(instance.ComposeFile))
	} else {
		projectPath, err = s.fileService.UpdateStoreStackFileFromBytes(stackFolder, stack.EntryPoint, []byte(instance.ComposeFile))
	}
	if err != nil {
		return fmt.Errorf("unable to store compose file: %w", err)
	}
	stack.ProjectPath = projectPath

	return s.dataStore.Stack().Update(stack.ID, stack)
}

// deployStack deploys (or redeploys) the stack on the target environment using
// the existing compose stack deployment machinery. The deployment config is
// built inside a write transaction (it may refresh ECR registry tokens) but
// the actual deployment runs outside of it so the BoltDB write lock is not
// held for the duration of the deployment.
func (s *Service) deployStack(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint, securityContext *security.RestrictedRequestContext, pullImage bool) error {
	var config *deployments.ComposeStackDeploymentConfig
	err := s.dataStore.UpdateTx(func(tx dataservices.DataStoreTx) error {
		var err error
		config, err = deployments.CreateComposeStackDeploymentConfigTx(tx, securityContext, stack, endpoint, s.fileService, s.stackDeployer, false, pullImage, false)
		return err
	})
	if err != nil {
		return err
	}

	deployErr := config.Deploy(ctx)
	if deployErr != nil {
		stackutils.UpdateStackStatusFromDeploymentResult(stack, deployErr)
		if err := s.dataStore.Stack().Update(stack.ID, stack); err != nil {
			log.Warn().Err(err).Int("stack_id", int(stack.ID)).Msg("unable to update stack status after failed deployment")
		}
		return deployErr
	}

	stack.Status = portainer.StackStatusActive
	stack.DeploymentStatus = []portainer.StackDeploymentStatus{
		{Status: portainer.StackStatusActive, Time: time.Now().Unix()},
	}
	return s.dataStore.Stack().Update(stack.ID, stack)
}

// undeployStack stops the stack on the target environment using the existing
// compose stack deployment machinery.
func (s *Service) undeployStack(ctx context.Context, stack *portainer.Stack, endpoint *portainer.Endpoint) error {
	if err := s.stackDeployer.UndeployComposeStack(ctx, stack, endpoint); err != nil {
		return err
	}

	stackutils.UpdateStackStatusFromUndeploymentResult(stack, nil)
	return s.dataStore.Stack().Update(stack.ID, stack)
}

func operationStatusName(status portainer.ServiceInstanceOperationStatus) string {
	switch status {
	case portainer.ServiceInstanceOperationStatusSuccess:
		return "success"
	case portainer.ServiceInstanceOperationStatusPartialSuccess:
		return "partial_success"
	case portainer.ServiceInstanceOperationStatusFailed:
		return "failed"
	case portainer.ServiceInstanceOperationStatusCancelled:
		return "cancelled"
	case portainer.ServiceInstanceOperationStatusRunning:
		return "running"
	default:
		return "pending"
	}
}
