package reconcile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

const (
	listLimit                       = 1000
	maxPollTimeout                  = 30 * time.Second
	deletionReasonCapacityReduction = "capacity_reduction"
	deletionReasonFleetDisabled     = "fleet_disabled"
)

type AdminClient interface {
	DescribeFleet(context.Context, string) (adminclient.Fleet, error)
	GetFleetCapacity(context.Context, string, string, int) (adminclient.Capacity, error)
	CreateRunner(
		context.Context,
		string,
		adminclient.CreateRunnerRequest,
	) (adminclient.CreateRunnerResponse, error)
	DescribeRunner(context.Context, string, string) (adminclient.Runner, error)
	ListRunners(context.Context, string, []string, int) ([]adminclient.Runner, error)
	DeleteRunner(context.Context, string, string) (adminclient.Runner, error)
}

type ArtifactResolver interface {
	Resolve(context.Context, string, string, string) (artifact.Artifact, error)
}

type Config struct {
	FleetID         string
	FleetManagerID  string
	WarmCapacity    int
	MaxCapacity     int
	OperatingSystem string
	Architecture    string
	PollTimeout     time.Duration
}

type RunConfig struct {
	Interval           time.Duration
	ErrorRetryInterval time.Duration
}

type Reconciler struct {
	admin      AdminClient
	artifacts  ArtifactResolver
	provider   provider.Provider
	config     Config
	log        *slog.Logger
	generation string
}

func New(
	admin AdminClient,
	artifacts ArtifactResolver,
	resourceProvider provider.Provider,
	config Config,
	log *slog.Logger,
) (*Reconciler, error) {
	config.FleetID = strings.TrimSpace(config.FleetID)
	config.OperatingSystem = strings.ToLower(strings.TrimSpace(config.OperatingSystem))
	config.Architecture = strings.ToLower(strings.TrimSpace(config.Architecture))
	switch {
	case admin == nil:
		return nil, fmt.Errorf("admin client is required")
	case artifacts == nil:
		return nil, fmt.Errorf("artifact resolver is required")
	case resourceProvider == nil:
		return nil, fmt.Errorf("provider is required")
	case config.FleetID == "":
		return nil, fmt.Errorf("fleet ID is required")
	case config.WarmCapacity < 0:
		return nil, fmt.Errorf("warm capacity must not be negative")
	case config.MaxCapacity < 0:
		return nil, fmt.Errorf("maximum capacity must not be negative")
	case config.MaxCapacity > 0 && config.WarmCapacity > config.MaxCapacity:
		return nil, fmt.Errorf("warm capacity must not exceed maximum capacity")
	case config.OperatingSystem == "":
		return nil, fmt.Errorf("operating system is required")
	case config.Architecture == "":
		return nil, fmt.Errorf("architecture is required")
	}
	if config.PollTimeout < 0 || config.PollTimeout > maxPollTimeout {
		return nil, fmt.Errorf(
			"poll timeout must be between 0 and %s",
			maxPollTimeout,
		)
	}
	if config.PollTimeout%time.Second != 0 {
		return nil, fmt.Errorf("poll timeout must use whole seconds")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Reconciler{
		admin:     admin,
		artifacts: artifacts,
		provider:  resourceProvider,
		config:    config,
		log:       log,
	}, nil
}

func (r *Reconciler) FleetID() string {
	return r.config.FleetID
}

func (r *Reconciler) logAction(message string, attributes ...slog.Attr) {
	args := make([]any, 0, len(attributes)+2)
	args = append(
		args,
		slog.String("fleet_id", r.config.FleetID),
		slog.String("provider", r.provider.Name()),
	)
	for _, attribute := range attributes {
		args = append(args, attribute)
	}
	r.log.Info(message, args...)
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	r.logAction("describing fleet")
	fleet, err := r.admin.DescribeFleet(ctx, r.config.FleetID)
	if err != nil {
		return fmt.Errorf("describe fleet %s: %w", r.config.FleetID, err)
	}
	if !fleet.Enabled {
		r.logAction("reconciling disabled fleet")
		r.generation = ""
		if err := r.reconcileDisabledFleet(ctx); err != nil {
			return err
		}
		r.logAction(
			"fleet reconciliation completed",
			slog.Bool("enabled", false),
		)
		return nil
	}
	r.logAction("validating fleet configuration")
	if err := r.validateFleet(fleet); err != nil {
		return err
	}

	previousGeneration := r.generation
	waitSeconds := r.capacityWaitSeconds()
	r.logAction(
		"getting fleet capacity",
		slog.String("generation", r.generation),
		slog.Int("wait_seconds", waitSeconds),
	)
	capacity, err := r.admin.GetFleetCapacity(
		ctx,
		r.config.FleetID,
		r.generation,
		waitSeconds,
	)
	if err != nil {
		return fmt.Errorf("get fleet %s capacity: %w", r.config.FleetID, err)
	}
	r.generation = capacity.Generation

	r.logAction("listing provider resources")
	resources, err := r.provider.List(ctx, r.config.FleetID)
	if err != nil {
		return fmt.Errorf("list %s resources for fleet %s: %w", r.provider.Name(), r.config.FleetID, err)
	}
	resourcesByRunner := groupResourcesByRunner(resources)

	var reconcileErrors []error
	r.logAction(
		"cleaning up terminated provider resources",
		slog.Int("provider_resources", len(resources)),
	)
	if err := r.deleteTerminatedResources(ctx, resourcesByRunner); err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}

	r.logAction("listing active runners")
	activeRunners, err := r.admin.ListRunners(
		ctx,
		r.config.FleetID,
		[]string{adminclient.RunnerStatePending, adminclient.RunnerStateIdle},
		listLimit,
	)
	if err != nil {
		reconcileErrors = append(
			reconcileErrors,
			fmt.Errorf("list active runners for fleet %s: %w", r.config.FleetID, err),
		)
		return errors.Join(reconcileErrors...)
	}

	r.logAction(
		"reconciling fleet capacity",
		slog.Int("active_runners", int(capacity.PendingRunners+capacity.IdleRunners)),
		slog.Int("active_runner_records", len(activeRunners)),
	)
	err = r.reconcileCapacity(ctx, fleet, capacity, activeRunners, resourcesByRunner)
	if err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}
	if err := errors.Join(reconcileErrors...); err != nil {
		return err
	}

	r.logAction(
		"fleet reconciliation completed",
		slog.Bool("enabled", true),
		slog.String("generation", capacity.Generation),
		slog.Bool("generation_changed", capacity.Generation != previousGeneration),
		slog.Int("capacity_wait_seconds", waitSeconds),
		slog.Int64("runnable_tasks", capacity.RunnableTasks),
		slog.Int64("pending_runners", capacity.PendingRunners),
		slog.Int64("idle_runners", capacity.IdleRunners),
		slog.Int64("busy_runners", capacity.BusyRunners),
		slog.Int64("terminated_runners", capacity.TerminatedRunners),
		slog.Int("provider_resources", len(resources)),
		slog.Int("max_capacity", r.config.MaxCapacity),
	)
	return nil
}

func (r *Reconciler) reconcileDisabledFleet(ctx context.Context) error {
	r.logAction("listing provider resources")
	resources, err := r.provider.List(ctx, r.config.FleetID)
	if err != nil {
		return fmt.Errorf("list resources for disabled fleet %s: %w", r.config.FleetID, err)
	}
	resourcesByRunner := groupResourcesByRunner(resources)
	var reconcileErrors []error
	r.logAction(
		"cleaning up terminated provider resources",
		slog.Int("provider_resources", len(resources)),
	)
	if err := r.deleteTerminatedResources(ctx, resourcesByRunner); err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}
	r.logAction("listing active runners")
	runners, err := r.admin.ListRunners(
		ctx,
		r.config.FleetID,
		[]string{adminclient.RunnerStatePending, adminclient.RunnerStateIdle},
		listLimit,
	)
	if err != nil {
		reconcileErrors = append(
			reconcileErrors,
			fmt.Errorf("list runners for disabled fleet %s: %w", r.config.FleetID, err),
		)
		return errors.Join(reconcileErrors...)
	}
	r.logAction(
		"terminating runners for disabled fleet",
		slog.Int("runner_count", len(runners)),
	)
	if err := r.terminateRunners(
		ctx,
		runners,
		resourcesByRunner,
		deletionReasonFleetDisabled,
	); err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}
	return errors.Join(reconcileErrors...)
}

func (r *Reconciler) validateFleet(fleet adminclient.Fleet) error {
	if !strings.EqualFold(fleet.Spec.OperatingSystem, r.config.OperatingSystem) {
		return fmt.Errorf(
			"fleet %s operating system is %q, provider expects %q",
			r.config.FleetID,
			fleet.Spec.OperatingSystem,
			r.config.OperatingSystem,
		)
	}
	if !strings.EqualFold(fleet.Spec.Architecture, r.config.Architecture) {
		return fmt.Errorf(
			"fleet %s architecture is %q, provider expects %q",
			r.config.FleetID,
			fleet.Spec.Architecture,
			r.config.Architecture,
		)
	}
	return nil
}

func (r *Reconciler) reconcileCapacity(
	ctx context.Context,
	fleet adminclient.Fleet,
	capacity adminclient.Capacity,
	activeRunners []adminclient.Runner,
	resourcesByRunner map[string][]provider.Resource,
) error {
	target := targetActiveRunners(capacity, r.config.WarmCapacity, r.config.MaxCapacity)
	active := int(capacity.PendingRunners + capacity.IdleRunners)
	if active < target {
		r.logAction(
			"increasing fleet capacity",
			slog.Int("active_runners", active),
			slog.Int("target_runners", target),
			slog.Int("runner_count", target-active),
		)
		var provisionErrors []error
		for range target - active {
			if err := r.provision(
				ctx,
				fleet,
				genericIdempotencyKey(r.config.FleetID),
				resourcesByRunner,
			); err != nil {
				provisionErrors = append(provisionErrors, err)
			}
		}
		return errors.Join(provisionErrors...)
	}
	if active == target {
		r.logAction(
			"fleet capacity matches target",
			slog.Int("active_runners", active),
			slog.Int("target_runners", target),
		)
		return nil
	}
	r.logAction(
		"decreasing fleet capacity",
		slog.Int("active_runners", active),
		slog.Int("target_runners", target),
		slog.Int("runner_count", active-target),
	)
	return r.terminateRunners(
		ctx,
		oldestRunners(activeRunners, active-target),
		resourcesByRunner,
		deletionReasonCapacityReduction,
	)
}

func targetActiveRunners(
	capacity adminclient.Capacity,
	warmCapacity, maxCapacity int,
) int {
	target := int(capacity.RunnableTasks) + warmCapacity
	if maxCapacity == 0 {
		return target
	}
	available := maxCapacity - int(capacity.BusyRunners)
	if available <= 0 {
		return 0
	}
	return min(target, available)
}

func (r *Reconciler) provision(
	ctx context.Context,
	fleet adminclient.Fleet,
	idempotencyKey string,
	resourcesByRunner map[string][]provider.Resource,
) error {
	r.logAction("creating logical runner")
	created, err := r.admin.CreateRunner(ctx, r.config.FleetID, adminclient.CreateRunnerRequest{
		IdempotencyKey: idempotencyKey,
		Ephemeral:      true,
	})
	if err != nil {
		return fmt.Errorf("create logical runner for fleet %s: %w", r.config.FleetID, err)
	}
	if created.Runner.State != adminclient.RunnerStatePending {
		return fmt.Errorf(
			"created runner %s is %s, expected pending",
			created.Runner.ID,
			created.Runner.State,
		)
	}
	if len(resourcesByRunner[created.Runner.ID]) > 0 {
		r.logAction(
			"using existing provider resource",
			slog.String("runner_id", created.Runner.ID),
			slog.Int("resource_count", len(resourcesByRunner[created.Runner.ID])),
		)
		return nil
	}

	r.logAction(
		"resolving runner artifact",
		slog.String("runner_id", created.Runner.ID),
		slog.String("runner_version", created.Runner.RunnerVersion),
	)
	resolved, err := r.artifacts.Resolve(
		ctx,
		created.Runner.RunnerVersion,
		fleet.Spec.OperatingSystem,
		fleet.Spec.Architecture,
	)
	if err != nil {
		return r.rollbackProvision(ctx, created.Runner.ID, fmt.Errorf("resolve runner artifact: %w", err))
	}
	r.logAction(
		"building runner bootstrap",
		slog.String("runner_id", created.Runner.ID),
	)
	bootstrap, err := r.provider.BuildBootstrap(provider.RunnerBootstrap{
		RunnerID:          created.Runner.ID,
		FleetID:           r.config.FleetID,
		RunnerAPIURL:      runnerBaseURL(created.RunnerAPIURL),
		RegistrationToken: created.RegistrationToken,
		Artifact:          resolved,
		Tags:              map[string]string{"fleet_manager_id": r.config.FleetManagerID},
	})
	if err != nil {
		return r.rollbackProvision(ctx, created.Runner.ID, fmt.Errorf("build runner bootstrap: %w", err))
	}
	r.logAction(
		"creating provider runner",
		slog.String("runner_id", created.Runner.ID),
	)
	resource, err := r.provider.Create(ctx, provider.CreateRequest{
		RunnerID:      created.Runner.ID,
		FleetID:       r.config.FleetID,
		RunnerVersion: created.Runner.RunnerVersion,
		Bootstrap:     bootstrap,
	})
	if err != nil {
		return r.rollbackProvision(ctx, created.Runner.ID, fmt.Errorf("create provider runner: %w", err))
	}
	resourcesByRunner[created.Runner.ID] = append(resourcesByRunner[created.Runner.ID], resource)
	r.logAction(
		"provisioned runner",
		slog.String("runner_id", created.Runner.ID),
		slog.String("resource_id", resource.ID),
	)
	return nil
}

func (r *Reconciler) rollbackProvision(
	ctx context.Context,
	runnerID string,
	provisionError error,
) error {
	r.logAction(
		"terminating logical runner after provisioning failure",
		slog.String("runner_id", runnerID),
	)
	_, rollbackError := r.admin.DeleteRunner(ctx, r.config.FleetID, runnerID)
	if rollbackError != nil {
		return errors.Join(
			provisionError,
			fmt.Errorf("terminate logical runner %s after provisioning failure: %w", runnerID, rollbackError),
		)
	}
	return provisionError
}

func (r *Reconciler) deleteTerminatedResources(
	ctx context.Context,
	resourcesByRunner map[string][]provider.Resource,
) error {
	var deleteErrors []error
	for runnerID, resources := range resourcesByRunner {
		r.logAction(
			"describing runner for resource cleanup",
			slog.String("runner_id", runnerID),
			slog.Int("resource_count", len(resources)),
		)
		runner, err := r.admin.DescribeRunner(ctx, r.config.FleetID, runnerID)
		if err != nil && !adminclient.IsStatus(err, http.StatusNotFound) {
			deleteErrors = append(deleteErrors, fmt.Errorf(
				"describe runner %s for resource cleanup: %w",
				runnerID,
				err,
			))
			continue
		}
		if err == nil && runner.State != adminclient.RunnerStateTerminated {
			continue
		}

		deleted := true
		for _, resource := range resources {
			r.logAction(
				"deleting provider resource",
				slog.String("runner_id", runnerID),
				slog.String("resource_id", resource.ID),
				slog.String("reason", "runner_terminated_or_missing"),
			)
			if err := r.provider.Delete(ctx, resource); err != nil {
				deleted = false
				deleteErrors = append(deleteErrors, fmt.Errorf(
					"delete provider resource %s for runner %s: %w",
					resource.ID,
					runnerID,
					err,
				))
			}
		}
		if deleted {
			delete(resourcesByRunner, runnerID)
		}
	}
	return errors.Join(deleteErrors...)
}

func (r *Reconciler) terminateRunners(
	ctx context.Context,
	runners []adminclient.Runner,
	resourcesByRunner map[string][]provider.Resource,
	deletionReason string,
) error {
	var terminateErrors []error
	for _, runner := range runners {
		r.logAction(
			"terminating logical runner",
			slog.String("runner_id", runner.ID),
		)
		_, err := r.admin.DeleteRunner(ctx, r.config.FleetID, runner.ID)
		if adminclient.IsStatus(err, http.StatusConflict) {
			continue
		}
		if err != nil {
			terminateErrors = append(terminateErrors, fmt.Errorf(
				"terminate logical runner %s: %w",
				runner.ID,
				err,
			))
			continue
		}
		for _, resource := range resourcesByRunner[runner.ID] {
			r.logAction(
				"deleting provider resource",
				slog.String("runner_id", runner.ID),
				slog.String("resource_id", resource.ID),
				slog.String("reason", deletionReason),
			)
			if err := r.provider.Delete(ctx, resource); err != nil {
				terminateErrors = append(terminateErrors, fmt.Errorf(
					"delete resource %s for runner %s: %w",
					resource.ID,
					runner.ID,
					err,
				))
			}
		}
	}
	return errors.Join(terminateErrors...)
}

func (r *Reconciler) capacityWaitSeconds() int {
	if r.generation == "" {
		return 0
	}
	return int(r.config.PollTimeout / time.Second)
}

func Run(
	ctx context.Context,
	log *slog.Logger,
	config RunConfig,
	reconcilers []*Reconciler,
) {
	if log == nil {
		log = slog.Default()
	}
	if config.Interval <= 0 {
		config.Interval = 15 * time.Second
	}
	if config.ErrorRetryInterval <= 0 {
		config.ErrorRetryInterval = 15 * time.Second
	}
	for _, current := range reconcilers {
		go runOne(ctx, log, config, current)
	}
}

func runOne(
	ctx context.Context,
	log *slog.Logger,
	config RunConfig,
	reconciler *Reconciler,
) {
	for {
		err := reconciler.Reconcile(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			if reconciler.config.PollTimeout == 0 || reconciler.generation == "" {
				if !waitForNextReconciliation(ctx, config.Interval) {
					return
				}
			}
			continue
		}
		log.Error(
			"fleet reconciliation failed",
			slog.String("fleet_id", reconciler.FleetID()),
			slog.Any("error", err),
		)
		if !waitForNextReconciliation(ctx, config.ErrorRetryInterval) {
			return
		}
	}
}

func waitForNextReconciliation(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func groupResourcesByRunner(resources []provider.Resource) map[string][]provider.Resource {
	grouped := make(map[string][]provider.Resource)
	for _, resource := range resources {
		if resource.RunnerID != "" {
			grouped[resource.RunnerID] = append(grouped[resource.RunnerID], resource)
		}
	}
	return grouped
}

func oldestRunners(runners []adminclient.Runner, count int) []adminclient.Runner {
	if count <= 0 {
		return nil
	}
	sorted := append([]adminclient.Runner(nil), runners...)
	sort.Slice(sorted, func(left, right int) bool {
		return sorted[left].CreatedAt.Before(sorted[right].CreatedAt)
	})
	if count > len(sorted) {
		count = len(sorted)
	}
	return sorted[:count]
}

func genericIdempotencyKey(fleetID string) string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("fleet-manager/%s/generic/%d", fleetID, time.Now().UnixNano())
	}
	return "fleet-manager/" + fleetID + "/generic/" + hex.EncodeToString(random)
}

func runnerBaseURL(apiURL string) string {
	return strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(apiURL), "/"), "/runner/v1")
}
