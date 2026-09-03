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
	dockerclient "github.com/portainer/portainer/api/docker/client"
	"github.com/portainer/portainer/api/docker/images"
	"github.com/portainer/portainer/api/filesystem"
	"github.com/portainer/portainer/api/http/security"
	"github.com/portainer/portainer/api/logs"
	"github.com/portainer/portainer/api/stacks/deployments"
	"github.com/portainer/portainer/api/stacks/stackutils"

	composeloader "github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
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
	// ErrInvalidComposeFile is returned when the provided compose file cannot be parsed.
	ErrInvalidComposeFile = errors.New("invalid compose file")
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
	clientFactory *dockerclient.ClientFactory

	mu              sync.Mutex
	running         map[portainer.ServiceInstanceID]bool
	scheduledBuilds map[portainer.ServiceInstanceScheduledBuildID]context.CancelFunc
}

// NewService creates a new Service.
func NewService(dataStore dataservices.DataStore, fileService portainer.FileService, stackDeployer deployments.StackDeployer, clientFactory *dockerclient.ClientFactory) *Service {
	return &Service{
		dataStore:       dataStore,
		fileService:     fileService,
		stackDeployer:   stackDeployer,
		clientFactory:   clientFactory,
		running:         make(map[portainer.ServiceInstanceID]bool),
		scheduledBuilds: make(map[portainer.ServiceInstanceScheduledBuildID]context.CancelFunc),
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

// ScheduleBuild schedules a build for a service instance: the images referenced
// by the provided compose file are pulled on all target environments immediately,
// and the compose file is deployed at the given unix timestamp.
func (s *Service) ScheduleBuild(ctx context.Context, instanceID portainer.ServiceInstanceID, composeFile string, deployAt int64, securityContext *security.RestrictedRequestContext) (*portainer.ServiceInstanceScheduledBuild, error) {
	if _, err := parseComposeFile(composeFile); err != nil {
		return nil, err
	}

	instance, err := s.dataStore.ServiceInstance().Read(instanceID)
	if err != nil {
		return nil, err
	}

	targets, err := s.ResolveTargets(instance)
	if err != nil {
		return nil, err
	}
	if len(targets.Endpoints) == 0 {
		return nil, ErrNoDeploymentTargets
	}

	results := make([]portainer.ServiceInstanceScheduledBuildTargetResult, 0, len(targets.Endpoints))
	for _, endpoint := range targets.Endpoints {
		results = append(results, portainer.ServiceInstanceScheduledBuildTargetResult{
			EnvironmentID: endpoint.ID,
			Status:        portainer.ServiceInstanceScheduledBuildTargetStatusPending,
		})
	}

	build := &portainer.ServiceInstanceScheduledBuild{
		ID:                portainer.ServiceInstanceScheduledBuildID(s.dataStore.ServiceInstanceScheduledBuild().GetNextIdentifier()),
		ServiceInstanceID: instanceID,
		ComposeFile:       composeFile,
		DeployAt:          deployAt,
		Status:            portainer.ServiceInstanceScheduledBuildStatusPending,
		UserID:            securityContext.UserID,
		CreatedAt:         time.Now().Unix(),
		Results:           results,
	}

	if err := s.dataStore.ServiceInstanceScheduledBuild().Create(build); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.scheduledBuilds[build.ID] = cancel
	s.mu.Unlock()

	go s.runScheduledBuild(runCtx, build, instance, targets.Endpoints, securityContext)

	return build, nil
}

// CancelScheduledBuild cancels a pending or pulling scheduled build.
func (s *Service) CancelScheduledBuild(id portainer.ServiceInstanceScheduledBuildID) error {
	build, err := s.dataStore.ServiceInstanceScheduledBuild().Read(id)
	if err != nil {
		return err
	}

	switch build.Status {
	case portainer.ServiceInstanceScheduledBuildStatusPending, portainer.ServiceInstanceScheduledBuildStatusPulling:
	default:
		return errors.New("only pending or pulling scheduled builds can be cancelled")
	}

	build.Status = portainer.ServiceInstanceScheduledBuildStatusCancelled
	now := time.Now().Unix()
	build.FinishedAt = &now
	if err := s.dataStore.ServiceInstanceScheduledBuild().Update(id, build); err != nil {
		return err
	}

	s.mu.Lock()
	if cancel, ok := s.scheduledBuilds[build.ID]; ok {
		cancel()
	}
	s.mu.Unlock()

	return nil
}

// ListScheduledBuilds returns all scheduled builds for the given service instance.
func (s *Service) ListScheduledBuilds(instanceID portainer.ServiceInstanceID) ([]portainer.ServiceInstanceScheduledBuild, error) {
	return s.dataStore.ServiceInstanceScheduledBuild().ReadAllByServiceInstanceID(instanceID)
}

// ReadScheduledBuild returns a scheduled build by ID.
func (s *Service) ReadScheduledBuild(id portainer.ServiceInstanceScheduledBuildID) (*portainer.ServiceInstanceScheduledBuild, error) {
	return s.dataStore.ServiceInstanceScheduledBuild().Read(id)
}

// RecoverScheduledBuilds re-schedules any scheduled build that was pending or
// pulling when the server was restarted.
func (s *Service) RecoverScheduledBuilds() {
	builds, err := s.dataStore.ServiceInstanceScheduledBuild().ReadAll(func(build portainer.ServiceInstanceScheduledBuild) bool {
		return build.Status == portainer.ServiceInstanceScheduledBuildStatusPending ||
			build.Status == portainer.ServiceInstanceScheduledBuildStatusPulling ||
			build.Status == portainer.ServiceInstanceScheduledBuildStatusImageReady
	})
	if err != nil {
		log.Warn().Err(err).Msg("unable to read scheduled builds for recovery")
		return
	}

	for _, build := range builds {
		instance, err := s.dataStore.ServiceInstance().Read(build.ServiceInstanceID)
		if err != nil {
			log.Warn().Err(err).Int("build_id", int(build.ID)).Msg("unable to recover scheduled build: service instance not found")
			continue
		}

		targets, err := s.ResolveTargets(instance)
		if err != nil || len(targets.Endpoints) == 0 {
			build.Status = portainer.ServiceInstanceScheduledBuildStatusFailed
			now := time.Now().Unix()
			build.FinishedAt = &now
			build.Error = "no deployment targets"
			if err := s.dataStore.ServiceInstanceScheduledBuild().Update(build.ID, &build); err != nil {
				log.Warn().Err(err).Int("build_id", int(build.ID)).Msg("unable to update failed scheduled build")
			}
			continue
		}

		securityContext := &security.RestrictedRequestContext{
			UserID:  build.UserID,
			IsAdmin: true,
		}

		runCtx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.scheduledBuilds[build.ID] = cancel
		s.mu.Unlock()

		go s.runScheduledBuild(runCtx, &build, instance, targets.Endpoints, securityContext)
	}
}

func (s *Service) runScheduledBuild(ctx context.Context, build *portainer.ServiceInstanceScheduledBuild, instance *portainer.ServiceInstance, endpoints []portainer.Endpoint, securityContext *security.RestrictedRequestContext) {
	defer s.removeScheduledBuildCancel(build.ID)

	build.Status = portainer.ServiceInstanceScheduledBuildStatusPulling
	s.persistScheduledBuild(build)

	imageNames, err := extractComposeImages(build.ComposeFile)
	if err != nil {
		s.failScheduledBuild(build, err)
		return
	}

	failed := false
	for i, endpoint := range endpoints {
		if ctx.Err() != nil {
			return
		}
		if failed {
			build.Results[i].Status = portainer.ServiceInstanceScheduledBuildTargetStatusSkipped
			s.persistScheduledBuild(build)
			continue
		}

		build.Results[i].Status = portainer.ServiceInstanceScheduledBuildTargetStatusPulling
		s.persistScheduledBuild(build)

		if err := s.pullImagesOnEndpoint(ctx, &endpoint, imageNames); err != nil {
			build.Results[i].Status = portainer.ServiceInstanceScheduledBuildTargetStatusFailed
			build.Results[i].Error = err.Error()
			failed = true
		} else {
			build.Results[i].Status = portainer.ServiceInstanceScheduledBuildTargetStatusImageReady
		}
		s.persistScheduledBuild(build)
	}

	if failed {
		s.failScheduledBuild(build, errors.New("image pull failed"))
		return
	}

	build.Status = portainer.ServiceInstanceScheduledBuildStatusImageReady
	s.persistScheduledBuild(build)

	if err := s.waitForDeployTime(ctx, build.DeployAt); err != nil {
		return
	}

	instance.ComposeFile = build.ComposeFile
	if err := s.dataStore.ServiceInstance().Update(instance.ID, instance); err != nil {
		s.failScheduledBuild(build, err)
		return
	}

	_, err = s.ExecuteOperation(ctx, instance.ID, portainer.ServiceInstanceOperationDeploy, securityContext, true)
	if err != nil {
		s.failScheduledBuild(build, err)
		return
	}

	for i := range build.Results {
		build.Results[i].Status = portainer.ServiceInstanceScheduledBuildTargetStatusDeployed
	}
	build.Status = portainer.ServiceInstanceScheduledBuildStatusDeployed
	now := time.Now().Unix()
	build.FinishedAt = &now
	s.persistScheduledBuild(build)
}

func (s *Service) removeScheduledBuildCancel(id portainer.ServiceInstanceScheduledBuildID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.scheduledBuilds, id)
}

func (s *Service) persistScheduledBuild(build *portainer.ServiceInstanceScheduledBuild) {
	if err := s.dataStore.ServiceInstanceScheduledBuild().Update(build.ID, build); err != nil {
		log.Warn().Err(err).Int("build_id", int(build.ID)).Msg("unable to persist service instance scheduled build")
	}
}

func (s *Service) failScheduledBuild(build *portainer.ServiceInstanceScheduledBuild, err error) {
	build.Status = portainer.ServiceInstanceScheduledBuildStatusFailed
	build.Error = err.Error()
	now := time.Now().Unix()
	build.FinishedAt = &now
	for i := range build.Results {
		switch build.Results[i].Status {
		case portainer.ServiceInstanceScheduledBuildTargetStatusPending,
			portainer.ServiceInstanceScheduledBuildTargetStatusPulling:
			build.Results[i].Status = portainer.ServiceInstanceScheduledBuildTargetStatusSkipped
		}
	}
	s.persistScheduledBuild(build)
}

func (s *Service) pullImagesOnEndpoint(ctx context.Context, endpoint *portainer.Endpoint, imageNames []string) error {
	if s.clientFactory == nil {
		return errors.New("docker client factory is not available")
	}

	cli, err := s.clientFactory.CreateClient(endpoint, "", nil)
	if err != nil {
		return errors.Wrapf(err, "unable to create docker client for environment %d", int(endpoint.ID))
	}
	defer logs.CloseAndLogErr(cli)

	puller := images.NewPuller(cli, images.NewRegistryClient(s.dataStore), s.dataStore)

	for _, name := range imageNames {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		img, err := images.ParseImage(images.ParseImageOptions{Name: name})
		if err != nil {
			return errors.Wrapf(err, "unable to parse image %q", name)
		}
		if err := puller.Pull(ctx, img); err != nil {
			return errors.Wrapf(err, "unable to pull image %q on environment %d", name, int(endpoint.ID))
		}
	}

	return nil
}

func (s *Service) waitForDeployTime(ctx context.Context, deployAt int64) error {
	wait := time.Until(time.Unix(deployAt, 0))
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseComposeFile(composeFile string) (*composetypes.Project, error) {
	composeConfig, err := composeloader.LoadWithContext(context.Background(), composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{{Content: []byte(composeFile)}},
	}, composeloader.WithSkipValidation)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidComposeFile, err.Error())
	}

	return composeConfig, nil
}

func extractComposeImages(composeFile string) ([]string, error) {
	composeConfig, err := parseComposeFile(composeFile)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	imageNames := make([]string, 0, len(composeConfig.Services))
	for _, service := range composeConfig.Services {
		if service.Image == "" || seen[service.Image] {
			continue
		}
		seen[service.Image] = true
		imageNames = append(imageNames, service.Image)
	}

	return imageNames, nil
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
// goroutine. When parallel is false, targets are processed sequentially and
// the operation stops at the first failure (fail-fast); remaining targets are
// marked as skipped. When parallel is true, all targets are processed
// concurrently and each reports its own success or failure.
func (s *Service) ExecuteOperation(ctx context.Context, instanceID portainer.ServiceInstanceID, operationType portainer.ServiceInstanceOperationType, securityContext *security.RestrictedRequestContext, parallel bool) (*portainer.ServiceInstanceOperation, error) {
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

	go s.runOperation(instance, operation, targets.Endpoints, securityContext, parallel)

	return operation, nil
}

func (s *Service) releaseOperation(instanceID portainer.ServiceInstanceID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, instanceID)
}

// runOperation executes the operation against each target and persists the
// results as they happen. When parallel is false, targets are processed
// sequentially with fail-fast semantics. When parallel is true, all targets
// are processed concurrently and each reports its own success or failure.
func (s *Service) runOperation(instance *portainer.ServiceInstance, operation *portainer.ServiceInstanceOperation, endpoints []portainer.Endpoint, securityContext *security.RestrictedRequestContext, parallel bool) {
	defer s.releaseOperation(instance.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if parallel {
		s.runOperationParallel(ctx, instance, operation, endpoints, securityContext)
		return
	}

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

// runOperationParallel executes the operation against all targets
// concurrently. Each target independently reports success or failure; the
// shared operation record is guarded by a mutex so results are persisted
// safely.
func (s *Service) runOperationParallel(ctx context.Context, instance *portainer.ServiceInstance, operation *portainer.ServiceInstanceOperation, endpoints []portainer.Endpoint, securityContext *security.RestrictedRequestContext) {
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, endpoint := range endpoints {
		wg.Add(1)
		go func(i int, endpoint portainer.Endpoint) {
			defer wg.Done()

			mu.Lock()
			operation.Results[i].Status = portainer.ServiceInstanceTargetStatusRunning
			s.persistOperation(operation)
			mu.Unlock()

			err := s.executeOnTarget(ctx, instance, &endpoint, operation.Type, securityContext)

			mu.Lock()
			if err != nil {
				operation.Results[i].Status = portainer.ServiceInstanceTargetStatusFailed
				operation.Results[i].Error = err.Error()
			} else {
				operation.Results[i].Status = portainer.ServiceInstanceTargetStatusSuccess
			}
			s.persistOperation(operation)
			mu.Unlock()
		}(i, endpoint)
	}

	wg.Wait()

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
	case portainer.ServiceInstanceOperationRestart:
		stack, err := s.findStackOnEndpoint(instance, endpoint.ID)
		if err != nil {
			return err
		}
		if err := s.undeployStack(ctx, stack, endpoint); err != nil {
			return err
		}
		return s.deployStack(ctx, stack, endpoint, securityContext, false)
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
