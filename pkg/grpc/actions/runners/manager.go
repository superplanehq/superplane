package runners

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/registration"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	runnerRegistrationTTL     = 10 * time.Minute
	defaultListLimit          = 200
	maxListLimit              = 1000
	maxIdempotencyKeyLength   = 255
	maxCapacityLongPoll       = 30 * time.Second
	capacityLongPollFrequency = time.Second
)

var errRunnerStatePrecondition = errors.New("runner state does not match the requested states")

// RunnerManager manages runner registration, capacity, and termination for resolved fleets.
// Callers authorize the fleet before invoking these operations.
type RunnerManager struct {
	signer       *jwt.Signer
	runnerAPIURL string
}

func NewRunnerManager(signer *jwt.Signer, baseURL string) *RunnerManager {
	return &RunnerManager{signer: signer, runnerAPIURL: strings.TrimRight(baseURL, "/")}
}

type CreateRunnerOptions struct {
	TaskID         *string
	Ephemeral      bool
	IdempotencyKey string
	FleetManagerID string
	Provider       string
}

type CreatedRunner struct {
	Runner                *models.Runner
	RegistrationToken     string
	RegistrationExpiresAt time.Time
	RunnerAPIURL          string
	DisplayName           string
}

type UpdateFleetOptions struct {
	RunnerVersion *string
	Enabled       *bool
	Spec          *models.RunnerFleetSpec
}

type CapacityOptions struct {
	Generation  *string
	WaitSeconds int32
}

type FleetCapacity struct {
	RunnableTasks     int64
	PendingRunners    int64
	IdleRunners       int64
	BusyRunners       int64
	TerminatedRunners int64
	Generation        string
}

type ListRunnersOptions struct {
	States  []string
	Limit   int32
	AfterID string
}

type TerminateRunnerOptions struct {
	RunnerID string
	States   []string
}

type ProviderDeletionOptions struct {
	RunnerID       string
	FleetManagerID string
	Provider       string
}

func (m *RunnerManager) UpdateFleet(ctx context.Context, fleet *models.RunnerFleet, options UpdateFleetOptions) (*models.RunnerFleet, error) {
	if options.RunnerVersion == nil && options.Enabled == nil && options.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "at least one fleet field is required")
	}
	if options.RunnerVersion != nil && strings.TrimSpace(*options.RunnerVersion) == "" {
		return nil, grpcerrors.InvalidArgument(nil, "runner version is required")
	}

	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if fleet.ScopeType == models.RunnerFleetScopeOrganization {
			if fleet.ScopeID == nil {
				return models.ErrRunnerFleetNotFound
			}
			if err := models.RequireOrganizationRunnerFeature(tx, *fleet.ScopeID); err != nil {
				return err
			}
		}
		// Take the write lock directly: concurrent writers must not upgrade
		// shared fleet locks while holding each other off.
		current, err := models.FindRunnerFleet(tx.Clauses(clause.Locking{Strength: "UPDATE"}), fleet.ID)
		if err != nil {
			return err
		}
		fleet = current
		if options.RunnerVersion != nil {
			if err := fleet.PinRunnerVersion(tx, *options.RunnerVersion); err != nil {
				return err
			}
		}
		if options.Enabled != nil {
			if err := fleet.SetEnabled(tx, *options.Enabled); err != nil {
				return err
			}
		}
		if options.Spec != nil {
			fleet.Spec = datatypes.NewJSONType(*options.Spec)
			fleet.UpdatedAt = time.Now()
			if err := tx.Model(fleet).Updates(map[string]any{
				"spec":       fleet.Spec,
				"updated_at": fleet.UpdatedAt,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if errors.Is(err, models.ErrRunnerFleetNotFound) {
		return nil, grpcerrors.NotFound(err, "fleet not found")
	}
	if errors.Is(err, models.ErrOrganizationRunnersDisabled) {
		return nil, grpcerrors.FailedPrecondition(err, err.Error())
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update fleet")
	}
	return fleet, nil
}

func (m *RunnerManager) GetFleetCapacity(ctx context.Context, fleet *models.RunnerFleet, options CapacityOptions) (*FleetCapacity, error) {
	wait := time.Duration(options.WaitSeconds) * time.Second
	if wait < 0 {
		return nil, grpcerrors.InvalidArgument(nil, "wait seconds must not be negative")
	}
	if wait > maxCapacityLongPoll {
		wait = maxCapacityLongPoll
	}

	deadline := time.Now().Add(wait)
	for {
		capacity, capacityErr := loadFleetCapacity(ctx, fleet)
		if capacityErr != nil {
			return nil, capacityErr
		}
		if options.Generation == nil || *options.Generation != capacity.Generation || wait == 0 {
			return capacity, nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return capacity, nil
		}
		pause := min(capacityLongPollFrequency, remaining)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pause):
		}
	}
}

func (m *RunnerManager) CreateRunner(ctx context.Context, fleet *models.RunnerFleet, options CreateRunnerOptions) (*CreatedRunner, error) {
	fleetID := fleet.Slug
	taskID, err := optionalID(options.TaskID, "task ID")
	if err != nil {
		return nil, err
	}

	idempotencyKey := strings.TrimSpace(options.IdempotencyKey)
	// Generic capacity can tolerate creating another pending runner after a
	// lost response. A task-specific request cannot: its reservation must be
	// recoverable by retrying the same logical create request.
	if taskID != nil && idempotencyKey == "" {
		return nil, grpcerrors.InvalidArgument(nil, "idempotency key is required for a task-specific runner")
	}
	if len(idempotencyKey) > maxIdempotencyKeyLength {
		return nil, grpcerrors.InvalidArgument(nil, "idempotency key is too long")
	}
	fleetManagerID := strings.TrimSpace(options.FleetManagerID)
	providerName := strings.TrimSpace(options.Provider)
	if (fleetManagerID == "") != (providerName == "") {
		return nil, grpcerrors.InvalidArgument(nil, "fleet manager ID and provider must be supplied together")
	}
	requestHash := ""
	if idempotencyKey != "" {
		requestHash = createRunnerRequestHash(fleetID, taskID, options.Ephemeral)
		if fleetManagerID != "" {
			// Preserve hashes for older clients that did not supply ownership.
			sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%q:%q", requestHash, fleetManagerID, providerName)))
			requestHash = hex.EncodeToString(sum[:])
		}
	}
	tags := map[string]string{}
	if fleetManagerID != "" {
		tags[models.RunnerTagFleetManagerID] = fleetManagerID
		tags[models.RunnerTagProvider] = providerName
	}

	runner, registration, err := m.createRunner(
		ctx,
		fleet,
		taskID,
		options.Ephemeral,
		idempotencyKey,
		requestHash,
		tags,
	)
	if err != nil {
		return nil, err
	}

	token, err := runnerapi.MintRegistrationToken(m.signer, runner, registration, fleetID, taskID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create runner registration token")
	}

	return &CreatedRunner{
		Runner:                runner,
		RegistrationToken:     token,
		RegistrationExpiresAt: registration.ExpiresAt,
		RunnerAPIURL:          m.runnerAPIURL,
		DisplayName:           "runner-" + runner.ID.String(),
	}, nil
}

func (m *RunnerManager) ListRunners(ctx context.Context, fleet *models.RunnerFleet, options ListRunnersOptions) (*models.RunnerListPage, error) {
	if err := validateRunnerStates(options.States); err != nil {
		return nil, err
	}
	afterID, err := parseOptionalID(options.AfterID, "after ID")
	if err != nil {
		return nil, err
	}

	page, err := fleet.ListRunners(database.DB(ctx), models.ListPage{
		States:  options.States,
		Limit:   listLimit(options.Limit),
		AfterID: afterID,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list runners")
	}

	return &page, nil
}

func (m *RunnerManager) DescribeRunner(ctx context.Context, fleet *models.RunnerFleet, runnerID string) (*models.Runner, error) {
	runner, err := findFleetRunner(ctx, fleet, runnerID)
	if err != nil {
		return nil, err
	}
	return runner, nil
}

func (m *RunnerManager) TerminateRunner(ctx context.Context, fleet *models.RunnerFleet, options TerminateRunnerOptions) (*models.Runner, error) {
	if err := validateRunnerStates(options.States); err != nil {
		return nil, err
	}
	runnerID, err := parseID(options.RunnerID, "runner ID")
	if err != nil {
		return nil, err
	}

	var runner *models.Runner
	var releasedTask *models.RunnerTask
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		runnerQuery := tx
		if len(options.States) > 0 {
			runnerQuery = runnerQuery.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		runner, err = fleet.FindRunner(runnerQuery, runnerID)
		if err != nil {
			return err
		}
		if len(options.States) > 0 && !slices.Contains(options.States, runner.State) {
			return errRunnerStatePrecondition
		}
		if runner.State == models.RunnerStateBusy {
			if !runner.Ephemeral {
				return models.ErrRunnerBusy
			}
			task, taskErr := runner.FindActiveTask(tx)
			if errors.Is(taskErr, models.ErrRunnerTaskNotFound) {
				return models.ErrRunnerBusy
			}
			if taskErr != nil {
				return taskErr
			}
			return task.RequestCancel(tx, time.Now())
		}
		releasedTask, err = runner.FindActiveTask(tx)
		if err != nil && !errors.Is(err, models.ErrRunnerTaskNotFound) {
			return err
		}
		return runner.Terminate(tx, models.RunnerTerminationRequested)
	})
	if errors.Is(err, models.ErrRunnerNotFound) {
		return nil, grpcerrors.NotFound(err, "runner not found")
	}
	if errors.Is(err, errRunnerStatePrecondition) {
		return nil, grpcerrors.Conflict(err, "runner state does not match the requested states")
	}
	if errors.Is(err, models.ErrRunnerBusy) ||
		errors.Is(err, models.ErrRunnerTaskNotCompletable) {
		return nil, grpcerrors.Conflict(err, "busy runner cannot be terminated")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to terminate runner")
	}
	if releasedTask != nil {
		messages.PublishPlanningRunnerStatus(database.DB(ctx), releasedTask)
	}
	_ = runnercontrol.Publish(runnercontrol.Notification{RunnerID: runner.ID.String()})
	return runner, nil
}

func (m *RunnerManager) MarkRunnerProviderDeleted(ctx context.Context, fleet *models.RunnerFleet, options ProviderDeletionOptions) (*models.Runner, error) {
	runnerID, err := parseID(options.RunnerID, "runner ID")
	if err != nil {
		return nil, err
	}

	var runner *models.Runner
	var logins []models.FactoryCodexConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		runner, err = fleet.FindRunner(tx, runnerID)
		if err != nil {
			return err
		}
		tags := runner.Tags.Data()
		directOrganizationRunner := fleet.ScopeType == models.RunnerFleetScopeOrganization &&
			tags[models.RunnerTagFleetManagerID] == "" && tags[models.RunnerTagProvider] == ""
		if (!directOrganizationRunner && (options.FleetManagerID == "" || options.Provider == "")) ||
			tags[models.RunnerTagFleetManagerID] != options.FleetManagerID ||
			tags[models.RunnerTagProvider] != options.Provider {
			return fmt.Errorf("runner provider ownership does not match the confirming manager")
		}
		logins, err = models.ListCodexSetupConnections(tx)
		if err != nil {
			return err
		}
		return runner.MarkProviderDeleted(tx, time.Now())
	})
	if errors.Is(err, models.ErrRunnerNotFound) {
		return nil, grpcerrors.NotFound(err, "runner not found")
	}
	if err != nil {
		return nil, grpcerrors.FailedPrecondition(err, "runner VM is not confirmed deleted")
	}
	messages.PublishCodexConnectionChanges(database.DB(ctx), logins)
	return runner, nil
}

func (m *RunnerManager) createRunner(
	ctx context.Context,
	fleet *models.RunnerFleet,
	taskID *uuid.UUID,
	ephemeral bool,
	idempotencyKey, requestHash string,
	tags map[string]string,
) (*models.Runner, *models.RunnerRegistration, error) {
	var runner *models.Runner
	var registration *models.RunnerRegistration
	var reservedTask *models.RunnerTask
	fleetID := fleet.Slug

	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var findErr error
		fleet, findErr = models.FindRunnerFleet(tx, fleet.ID)
		if findErr != nil {
			return findErr
		}
		if err := fleet.LockAdmission(tx); err != nil {
			return err
		}
		if fleet.ScopeType == models.RunnerFleetScopeOrganization && !fleet.Enabled {
			return models.ErrRunnerFleetDisabled
		}
		if idempotencyKey != "" {
			if releaseErr := models.ReleaseTerminatedRunnerCreationIdempotencyKey(
				tx,
				fleet.ID,
				idempotencyKey,
			); releaseErr != nil {
				return releaseErr
			}
			existingRunner, findErr := loadIdempotentRunner(tx, fleet.ID, idempotencyKey, requestHash)
			if findErr == nil {
				runner = existingRunner
				registration, findErr = models.FindRunnerRegistration(tx, runner.ID)
				return findErr
			}
			if !errors.Is(findErr, models.ErrRunnerNotFound) {
				return findErr
			}
		}
		if !fleet.Enabled {
			return models.ErrRunnerFleetDisabled
		}

		// Match PostgreSQL precision before signing the persisted grant. A
		// retry must not cross a JWT second boundary after database rounding.
		now := time.Now().Truncate(time.Microsecond)
		runner = &models.Runner{
			ID:            uuid.New(),
			FleetID:       fleet.ID,
			State:         models.RunnerStatePending,
			RunnerVersion: fleet.RunnerVersion,
			Ephemeral:     ephemeral,
			Tags:          datatypes.NewJSONType(tags),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if idempotencyKey != "" {
			// Store the key on the runner before reserving a task. The unique
			// index fences concurrent retries, while the hash rejects reuse of
			// the same fleet-local key for different runner or task inputs.
			runner.CreationIdempotencyKey = &idempotencyKey
			runner.CreationRequestHash = &requestHash
		}
		if createErr := tx.Create(runner).Error; createErr != nil {
			return createErr
		}

		if taskID != nil {
			task, taskErr := models.FindRunnerTask(tx, *taskID)
			if taskErr != nil {
				return taskErr
			}
			if task.FleetID != fleet.ID {
				return models.ErrRunnerTaskFleetMismatch
			}
			if fleet.ScopeType == models.RunnerFleetScopeOrganization &&
				(fleet.ScopeID == nil || task.OrganizationID != *fleet.ScopeID) {
				return models.ErrRunnerTaskFleetMismatch
			}
			if task.Backend != models.RunnerTaskBackendIntegrated {
				return models.ErrRunnerTaskBackendMismatch
			}
			if taskErr := task.Reserve(tx, runner.ID); taskErr != nil {
				return taskErr
			}
			reservedTask = task
		}

		registration = &models.RunnerRegistration{
			JTI:       uuid.New(),
			RunnerID:  runner.ID,
			ExpiresAt: now.Add(runnerRegistrationTTL),
			CreatedAt: now,
		}
		return tx.Create(registration).Error
	})
	if err == nil && reservedTask != nil {
		metricScope, scopeErr := fleet.MetricScope(database.DB(ctx))
		if scopeErr == nil {
			telemetry.RecordRunnerTaskQueueDuration(ctx, reservedTask, fleetID, metricScope)
		}
		messages.PublishPlanningRunnerStatus(database.DB(ctx), reservedTask)
	}
	if idempotencyKey != "" && models.IsRunnerCreationIdempotencyKeyConflict(err) {
		runner, err = loadIdempotentRunner(database.DB(ctx), fleet.ID, idempotencyKey, requestHash)
		if err == nil {
			registration, err = models.FindRunnerRegistration(database.DB(ctx), runner.ID)
		}
	}

	switch {
	case errors.Is(err, models.ErrRunnerFleetNotFound):
		return nil, nil, grpcerrors.NotFound(err, "fleet not found")
	case errors.Is(err, models.ErrRunnerTaskNotFound):
		return nil, nil, grpcerrors.NotFound(err, "task not found")
	case errors.Is(err, models.ErrCodexConnectionStale),
		errors.Is(err, models.ErrCodexConnectionNotEligible),
		errors.Is(err, models.ErrCodexConnectionNotReady),
		errors.Is(err, models.ErrSubscriptionOwnerRequired),
		errors.Is(err, models.ErrSubscriptionOwnerInvalid),
		errors.Is(err, models.ErrSubscriptionTaskAssociation),
		errors.Is(err, models.ErrSubscriptionModelUnavailable),
		errors.Is(err, models.ErrRunnerTaskNotReservable),
		errors.Is(err, models.ErrRunnerTaskFleetMismatch),
		errors.Is(err, models.ErrRunnerTaskBackendMismatch):
		return nil, nil, grpcerrors.FailedPrecondition(err, err.Error())
	case errors.Is(err, models.ErrRunnerIdempotencyConflict):
		return nil, nil, grpcerrors.AlreadyExists(err, err.Error())
	case errors.Is(err, models.ErrRunnerFleetDisabled), errors.Is(err, models.ErrOrganizationRunnersDisabled):
		return nil, nil, grpcerrors.FailedPrecondition(err, err.Error())
	case err != nil:
		return nil, nil, grpcerrors.Internal(err, "failed to create runner")
	default:
		return runner, registration, nil
	}
}

func loadIdempotentRunner(
	tx *gorm.DB,
	fleetID uuid.UUID,
	idempotencyKey, requestHash string,
) (*models.Runner, error) {
	runner, err := models.FindRunnerByCreationIdempotencyKey(tx, fleetID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if runner.CreationRequestHash == nil || *runner.CreationRequestHash != requestHash {
		return nil, models.ErrRunnerIdempotencyConflict
	}
	return runner, nil
}

func loadFleetCapacity(ctx context.Context, fleet *models.RunnerFleet) (*FleetCapacity, error) {
	db := database.DB(ctx)
	runnable, err := fleet.CountTasks(db, models.RunnerTaskStateQueued)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to count runnable tasks")
	}
	counts, err := fleet.CountRunnersByState(db)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to count runners")
	}

	generationSource := fmt.Sprintf(
		"%d:%d:%d:%d:%d:%d",
		fleet.UpdatedAt.UnixNano(),
		runnable,
		counts[models.RunnerStatePending],
		counts[models.RunnerStateIdle],
		counts[models.RunnerStateBusy],
		counts[models.RunnerStateTerminated],
	)
	generationBytes := sha256.Sum256([]byte(generationSource))

	return &FleetCapacity{
		RunnableTasks:     runnable,
		PendingRunners:    counts[models.RunnerStatePending],
		IdleRunners:       counts[models.RunnerStateIdle],
		BusyRunners:       counts[models.RunnerStateBusy],
		TerminatedRunners: counts[models.RunnerStateTerminated],
		Generation:        hex.EncodeToString(generationBytes[:]),
	}, nil
}

func findFleetRunner(
	ctx context.Context,
	fleet *models.RunnerFleet,
	rawID string,
) (*models.Runner, error) {
	id, err := parseID(rawID, "runner ID")
	if err != nil {
		return nil, err
	}
	runner, err := fleet.FindRunner(database.DB(ctx), id)
	if errors.Is(err, models.ErrRunnerNotFound) {
		return nil, grpcerrors.NotFound(err, "runner not found")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to find runner")
	}
	return runner, nil
}

func parseID(value, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, grpcerrors.InvalidArgument(err, "invalid "+field)
	}
	return id, nil
}

func parseOptionalID(value, field string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := parseID(value, field)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func optionalID(value *string, field string) (*uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	id, err := parseID(*value, field)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func createRunnerRequestHash(fleetID string, taskID *uuid.UUID, ephemeral bool) string {
	source := fmt.Sprintf("%s:%s:%t", fleetID, optionalIDString(taskID), ephemeral)
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

func optionalIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func listLimit(requested int32) int {
	if requested <= 0 {
		return defaultListLimit
	}
	return min(int(requested), maxListLimit)
}

func validateRunnerStates(states []string) error {
	valid := []string{
		models.RunnerStatePending,
		models.RunnerStateIdle,
		models.RunnerStateBusy,
		models.RunnerStateTerminated,
	}
	for _, state := range states {
		if !slices.Contains(valid, state) {
			return grpcerrors.InvalidArgument(nil, "invalid runner state")
		}
	}
	return nil
}
