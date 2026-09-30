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

const listLimit = 1000

type AdminClient interface {
	DescribeFleet(context.Context, string) (adminclient.Fleet, error)
	GetFleetCapacity(context.Context, string, string, int) (adminclient.Capacity, error)
	CreateRunner(
		context.Context,
		string,
		adminclient.CreateRunnerRequest,
	) (adminclient.CreateRunnerResponse, error)
	ListRunners(context.Context, string, []string, int) ([]adminclient.Runner, error)
	DeleteRunner(context.Context, string, string) (adminclient.Runner, error)
}

type ArtifactResolver interface {
	Resolve(context.Context, string, string, string) (artifact.Artifact, error)
}

type Config struct {
	FleetID             string
	WarmCapacity        int
	OperatingSystem     string
	Architecture        string
	CapacityWaitSeconds int
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
	case config.OperatingSystem == "":
		return nil, fmt.Errorf("operating system is required")
	case config.Architecture == "":
		return nil, fmt.Errorf("architecture is required")
	}
	if config.CapacityWaitSeconds < 0 || config.CapacityWaitSeconds > 30 {
		return nil, fmt.Errorf("capacity wait must be between 0 and 30 seconds")
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

func (r *Reconciler) Reconcile(ctx context.Context) error {
	fleet, err := r.admin.DescribeFleet(ctx, r.config.FleetID)
	if err != nil {
		return fmt.Errorf("describe fleet %s: %w", r.config.FleetID, err)
	}
	if !fleet.Enabled {
		r.generation = ""
		if err := r.reconcileDisabledFleet(ctx); err != nil {
			return err
		}
		r.log.Info(
			"fleet reconciliation completed",
			slog.String("fleet_id", r.config.FleetID),
			slog.String("provider", r.provider.Name()),
			slog.Bool("enabled", false),
		)
		return nil
	}
	if err := r.validateFleet(fleet); err != nil {
		return err
	}

	previousGeneration := r.generation
	waitSeconds := r.capacityWaitSeconds()
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

	resources, err := r.provider.List(ctx, r.config.FleetID)
	if err != nil {
		return fmt.Errorf("list %s resources for fleet %s: %w", r.provider.Name(), r.config.FleetID, err)
	}
	resourcesByRunner := groupResourcesByRunner(resources)

	var reconcileErrors []error
	if err := r.deleteTerminatedResources(ctx, resourcesByRunner); err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}

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

	err = r.reconcileCapacity(ctx, fleet, capacity, activeRunners, resourcesByRunner)
	if err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}
	if err := errors.Join(reconcileErrors...); err != nil {
		return err
	}

	r.log.Info(
		"fleet reconciliation completed",
		slog.String("fleet_id", r.config.FleetID),
		slog.String("provider", r.provider.Name()),
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
	)
	return nil
}

func (r *Reconciler) reconcileDisabledFleet(ctx context.Context) error {
	resources, err := r.provider.List(ctx, r.config.FleetID)
	if err != nil {
		return fmt.Errorf("list resources for disabled fleet %s: %w", r.config.FleetID, err)
	}
	resourcesByRunner := groupResourcesByRunner(resources)
	var reconcileErrors []error
	if err := r.deleteTerminatedResources(ctx, resourcesByRunner); err != nil {
		reconcileErrors = append(reconcileErrors, err)
	}
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
	if err := r.terminateRunners(ctx, runners, resourcesByRunner); err != nil {
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
	target := int(capacity.RunnableTasks) + r.config.WarmCapacity
	if len(activeRunners) < target {
		var provisionErrors []error
		for range target - len(activeRunners) {
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
	if len(activeRunners) == target {
		return nil
	}
	return r.terminateRunners(
		ctx,
		oldestRunners(activeRunners, len(activeRunners)-target),
		resourcesByRunner,
	)
}

func (r *Reconciler) provision(
	ctx context.Context,
	fleet adminclient.Fleet,
	idempotencyKey string,
	resourcesByRunner map[string][]provider.Resource,
) error {
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
		return nil
	}

	resolved, err := r.artifacts.Resolve(
		ctx,
		created.Runner.RunnerVersion,
		fleet.Spec.OperatingSystem,
		fleet.Spec.Architecture,
	)
	if err != nil {
		return r.rollbackProvision(ctx, created.Runner.ID, fmt.Errorf("resolve runner artifact: %w", err))
	}
	bootstrap, err := r.provider.BuildBootstrap(provider.RunnerBootstrap{
		RunnerID:          created.Runner.ID,
		FleetID:           r.config.FleetID,
		RunnerAPIURL:      runnerBaseURL(created.RunnerAPIURL),
		RegistrationToken: created.RegistrationToken,
		Artifact:          resolved,
	})
	if err != nil {
		return r.rollbackProvision(ctx, created.Runner.ID, fmt.Errorf("build runner bootstrap: %w", err))
	}
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
	r.log.Info(
		"provisioned runner",
		slog.String("fleet_id", r.config.FleetID),
		slog.String("runner_id", created.Runner.ID),
		slog.String("provider", r.provider.Name()),
		slog.String("resource_id", resource.ID),
	)
	return nil
}

func (r *Reconciler) rollbackProvision(
	ctx context.Context,
	runnerID string,
	provisionError error,
) error {
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
	terminated, err := r.admin.ListRunners(
		ctx,
		r.config.FleetID,
		[]string{adminclient.RunnerStateTerminated},
		listLimit,
	)
	if err != nil {
		return fmt.Errorf("list terminated runners for fleet %s: %w", r.config.FleetID, err)
	}
	var deleteErrors []error
	for _, runner := range terminated {
		for _, resource := range resourcesByRunner[runner.ID] {
			if err := r.provider.Delete(ctx, resource); err != nil {
				deleteErrors = append(deleteErrors, fmt.Errorf(
					"delete provider resource %s for terminated runner %s: %w",
					resource.ID,
					runner.ID,
					err,
				))
			}
		}
		delete(resourcesByRunner, runner.ID)
	}
	return errors.Join(deleteErrors...)
}

func (r *Reconciler) terminateRunners(
	ctx context.Context,
	runners []adminclient.Runner,
	resourcesByRunner map[string][]provider.Resource,
) error {
	var terminateErrors []error
	for _, runner := range runners {
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
	return r.config.CapacityWaitSeconds
}

func Run(
	ctx context.Context,
	log *slog.Logger,
	errorRetryInterval time.Duration,
	reconcilers []*Reconciler,
) {
	if log == nil {
		log = slog.Default()
	}
	if errorRetryInterval <= 0 {
		errorRetryInterval = 15 * time.Second
	}
	for _, current := range reconcilers {
		go runOne(ctx, log, errorRetryInterval, current)
	}
}

func runOne(
	ctx context.Context,
	log *slog.Logger,
	errorRetryInterval time.Duration,
	reconciler *Reconciler,
) {
	for {
		err := reconciler.Reconcile(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			if reconciler.config.CapacityWaitSeconds == 0 || reconciler.generation == "" {
				if !waitForRetry(ctx, errorRetryInterval) {
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
		if !waitForRetry(ctx, errorRetryInterval) {
			return
		}
	}
}

func waitForRetry(ctx context.Context, interval time.Duration) bool {
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
