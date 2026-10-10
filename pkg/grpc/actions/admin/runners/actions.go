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
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	"github.com/superplanehq/superplane/pkg/public/runnerapi"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	runnerRegistrationTTL     = 10 * time.Minute
	defaultListLimit          = 200
	maxListLimit              = 1000
	maxIdempotencyKeyLength   = 255
	maxCapacityLongPoll       = 30 * time.Second
	capacityLongPollFrequency = time.Second
)

type Service struct {
	signer       *jwt.Signer
	runnerAPIURL string
}

func NewService(signer *jwt.Signer, baseURL string) *Service {
	return &Service{
		signer:       signer,
		runnerAPIURL: strings.TrimRight(baseURL, "/") + "/runner/v1",
	}
}

func (s *Service) ListFleets(ctx context.Context, _ *pb.ListFleetsRequest) (*pb.ListFleetsResponse, error) {
	fleets, err := models.ListInstallationRunnerFleets(database.DB(ctx))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list fleets")
	}

	out := make([]*pb.Fleet, 0, len(fleets))
	for i := range fleets {
		out = append(out, serializeFleet(&fleets[i]))
	}
	return &pb.ListFleetsResponse{Fleets: out}, nil
}

func (s *Service) CreateFleet(
	ctx context.Context,
	req *pb.CreateFleetRequest,
) (*pb.CreateFleetResponse, error) {
	fleetID, err := parseFleetID(req.GetFleetId())
	if err != nil {
		return nil, err
	}
	runnerVersion := strings.TrimSpace(req.GetRunnerVersion())
	if runnerVersion == "" {
		return nil, grpcerrors.InvalidArgument(nil, "runner version is required")
	}
	if req.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "fleet spec is required")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = req.GetEnabled()
	}
	now := time.Now()
	fleet := &models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          fleetID,
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       enabled,
		Spec:          datatypes.NewJSONType(fleetSpecFromProto(req.Spec)),
		RunnerVersion: runnerVersion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = fleet.Create(database.DB(ctx))
	if errors.Is(err, models.ErrRunnerFleetIDConflict) {
		return nil, grpcerrors.AlreadyExists(err, "fleet ID already exists")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create fleet")
	}
	return &pb.CreateFleetResponse{Fleet: serializeFleet(fleet)}, nil
}

func (s *Service) DescribeFleet(ctx context.Context, req *pb.DescribeFleetRequest) (*pb.DescribeFleetResponse, error) {
	fleet, err := findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeFleetResponse{Fleet: serializeFleet(fleet)}, nil
}

func (s *Service) UpdateFleet(ctx context.Context, req *pb.UpdateFleetRequest) (*pb.UpdateFleetResponse, error) {
	if req.RunnerVersion == nil && req.Enabled == nil && req.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "at least one fleet field is required")
	}
	if req.RunnerVersion != nil && strings.TrimSpace(req.GetRunnerVersion()) == "" {
		return nil, grpcerrors.InvalidArgument(nil, "runner version is required")
	}

	fleetID, err := parseFleetID(req.GetFleetId())
	if err != nil {
		return nil, err
	}

	var fleet *models.RunnerFleet
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		fleet, err = models.FindInstallationRunnerFleet(tx, fleetID)
		if err != nil {
			return err
		}
		if req.RunnerVersion != nil {
			if err := fleet.PinRunnerVersion(tx, req.GetRunnerVersion()); err != nil {
				return err
			}
		}
		if req.Enabled != nil {
			if err := fleet.SetEnabled(tx, req.GetEnabled()); err != nil {
				return err
			}
		}
		if req.Spec != nil {
			fleet.Spec = datatypes.NewJSONType(fleetSpecFromProto(req.Spec))
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
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update fleet")
	}
	return &pb.UpdateFleetResponse{Fleet: serializeFleet(fleet)}, nil
}

func (s *Service) GetFleetCapacity(
	ctx context.Context,
	req *pb.GetFleetCapacityRequest,
) (*pb.GetFleetCapacityResponse, error) {
	fleet, err := findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}

	wait := time.Duration(req.GetWaitSeconds()) * time.Second
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
		if req.Generation == nil || req.GetGeneration() != capacity.Generation || wait == 0 {
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

func (s *Service) ListFleetTasks(
	ctx context.Context,
	req *pb.ListFleetTasksRequest,
) (*pb.ListFleetTasksResponse, error) {
	fleet, err := findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	if err := validateTaskStates(req.GetStates()); err != nil {
		return nil, err
	}
	afterID, err := parseOptionalID(req.GetAfterId(), "after ID")
	if err != nil {
		return nil, err
	}

	page, err := fleet.ListTasks(database.DB(ctx), models.ListPage{
		States:  req.GetStates(),
		Limit:   listLimit(req.GetLimit()),
		AfterID: afterID,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list fleet tasks")
	}

	out := make([]*pb.Task, 0, len(page.Tasks))
	for i := range page.Tasks {
		out = append(out, serializeTask(&page.Tasks[i], fleet.Slug))
	}
	return &pb.ListFleetTasksResponse{
		Tasks:       out,
		TotalCount:  page.TotalCount,
		HasNextPage: page.HasNextPage,
	}, nil
}

func (s *Service) CreateRunner(ctx context.Context, req *pb.CreateRunnerRequest) (*pb.CreateRunnerResponse, error) {
	fleetID, err := parseFleetID(req.GetFleetId())
	if err != nil {
		return nil, err
	}

	taskID, err := optionalID(req.TaskId, "task ID")
	if err != nil {
		return nil, err
	}
	if taskID != nil && !req.GetEphemeral() {
		return nil, grpcerrors.InvalidArgument(nil, "task-specific runners must be ephemeral")
	}

	idempotencyKey := strings.TrimSpace(req.GetIdempotencyKey())
	// Generic capacity can tolerate creating another pending runner after a
	// lost response. A task-specific request cannot: its reservation must be
	// recoverable by retrying the same logical create request.
	if taskID != nil && idempotencyKey == "" {
		return nil, grpcerrors.InvalidArgument(nil, "idempotency key is required for a task-specific runner")
	}
	if len(idempotencyKey) > maxIdempotencyKeyLength {
		return nil, grpcerrors.InvalidArgument(nil, "idempotency key is too long")
	}
	requestHash := ""
	if idempotencyKey != "" {
		requestHash = createRunnerRequestHash(fleetID, taskID, req.GetEphemeral())
	}

	runner, registration, err := s.createRunner(
		ctx,
		fleetID,
		taskID,
		req.GetEphemeral(),
		idempotencyKey,
		requestHash,
	)
	if err != nil {
		return nil, err
	}

	token, err := runnerapi.MintRegistrationToken(s.signer, runner, registration, fleetID, taskID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create runner registration token")
	}

	return &pb.CreateRunnerResponse{
		Runner:                serializeRunner(runner, fleetID),
		RegistrationToken:     token,
		RegistrationExpiresAt: timestamppb.New(registration.ExpiresAt),
		RunnerApiUrl:          s.runnerAPIURL,
		DisplayName:           "runner-" + runner.ID.String(),
	}, nil
}

func (s *Service) ListRunners(ctx context.Context, req *pb.ListRunnersRequest) (*pb.ListRunnersResponse, error) {
	fleet, err := findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	if err := validateRunnerStates(req.GetStates()); err != nil {
		return nil, err
	}
	afterID, err := parseOptionalID(req.GetAfterId(), "after ID")
	if err != nil {
		return nil, err
	}

	page, err := fleet.ListRunners(database.DB(ctx), models.ListPage{
		States:  req.GetStates(),
		Limit:   listLimit(req.GetLimit()),
		AfterID: afterID,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list runners")
	}

	out := make([]*pb.Runner, 0, len(page.Runners))
	for i := range page.Runners {
		out = append(out, serializeRunner(&page.Runners[i], fleet.Slug))
	}
	return &pb.ListRunnersResponse{
		Runners:     out,
		TotalCount:  page.TotalCount,
		HasNextPage: page.HasNextPage,
	}, nil
}

func (s *Service) DescribeRunner(
	ctx context.Context,
	req *pb.DescribeRunnerRequest,
) (*pb.DescribeRunnerResponse, error) {
	fleet, err := findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	runner, err := findFleetRunner(ctx, fleet, req.GetRunnerId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeRunnerResponse{Runner: serializeRunner(runner, fleet.Slug)}, nil
}

func (s *Service) DeleteRunner(ctx context.Context, req *pb.DeleteRunnerRequest) (*pb.DeleteRunnerResponse, error) {
	fleet, err := findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	runnerID, err := parseID(req.GetRunnerId(), "runner ID")
	if err != nil {
		return nil, err
	}

	var runner *models.Runner
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		runner, err = fleet.FindRunner(tx, runnerID)
		if err != nil {
			return err
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
		return runner.Terminate(tx, models.RunnerTerminationRequested)
	})
	if errors.Is(err, models.ErrRunnerNotFound) {
		return nil, grpcerrors.NotFound(err, "runner not found")
	}
	if errors.Is(err, models.ErrRunnerBusy) ||
		errors.Is(err, models.ErrRunnerTaskNotCompletable) {
		return nil, grpcerrors.Conflict(err, "busy runner cannot be terminated")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to terminate runner")
	}
	_ = runnercontrol.Publish(runnercontrol.Notification{RunnerID: runner.ID.String()})
	return &pb.DeleteRunnerResponse{Runner: serializeRunner(runner, fleet.Slug)}, nil
}

func (s *Service) createRunner(
	ctx context.Context,
	fleetID string,
	taskID *uuid.UUID,
	ephemeral bool,
	idempotencyKey, requestHash string,
) (*models.Runner, *models.RunnerRegistration, error) {
	var runner *models.Runner
	var registration *models.RunnerRegistration
	var reservedTask *models.RunnerTask

	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if idempotencyKey != "" {
			if releaseErr := models.ReleaseTerminatedRunnerCreationIdempotencyKey(
				tx,
				idempotencyKey,
			); releaseErr != nil {
				return releaseErr
			}
			existingRunner, findErr := loadIdempotentRunner(tx, idempotencyKey, requestHash)
			if findErr == nil {
				runner = existingRunner
				registration, findErr = models.FindRunnerRegistration(tx, runner.ID)
				return findErr
			}
			if !errors.Is(findErr, models.ErrRunnerNotFound) {
				return findErr
			}
		}

		fleet, findErr := models.FindInstallationRunnerFleet(tx, fleetID)
		if findErr != nil {
			return findErr
		}
		if !fleet.Enabled {
			return errFleetDisabled
		}

		now := time.Now()
		runner = &models.Runner{
			ID:            uuid.New(),
			FleetID:       fleet.ID,
			State:         models.RunnerStatePending,
			RunnerVersion: fleet.RunnerVersion,
			Ephemeral:     ephemeral,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if idempotencyKey != "" {
			// Store the key on the runner before reserving a task. The unique
			// index fences concurrent retries, while the hash rejects reuse of
			// the same key for different fleet or task inputs.
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
		telemetry.RecordRunnerTaskQueueDuration(ctx, reservedTask)
	}
	if idempotencyKey != "" && models.IsRunnerCreationIdempotencyKeyConflict(err) {
		runner, err = loadIdempotentRunner(database.DB(ctx), idempotencyKey, requestHash)
		if err == nil {
			registration, err = models.FindRunnerRegistration(database.DB(ctx), runner.ID)
		}
	}

	switch {
	case errors.Is(err, models.ErrRunnerFleetNotFound):
		return nil, nil, grpcerrors.NotFound(err, "fleet not found")
	case errors.Is(err, models.ErrRunnerTaskNotFound):
		return nil, nil, grpcerrors.NotFound(err, "task not found")
	case errors.Is(err, models.ErrRunnerTaskNotReservable),
		errors.Is(err, models.ErrRunnerTaskFleetMismatch),
		errors.Is(err, models.ErrRunnerTaskBackendMismatch):
		return nil, nil, grpcerrors.FailedPrecondition(err, err.Error())
	case errors.Is(err, models.ErrRunnerIdempotencyConflict):
		return nil, nil, grpcerrors.AlreadyExists(err, err.Error())
	case errors.Is(err, errFleetDisabled):
		return nil, nil, grpcerrors.FailedPrecondition(err, err.Error())
	case err != nil:
		return nil, nil, grpcerrors.Internal(err, "failed to create runner")
	default:
		return runner, registration, nil
	}
}

func loadIdempotentRunner(
	tx *gorm.DB,
	idempotencyKey, requestHash string,
) (*models.Runner, error) {
	runner, err := models.FindInstallationRunnerByCreationIdempotencyKey(tx, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if runner.CreationRequestHash == nil || *runner.CreationRequestHash != requestHash {
		return nil, models.ErrRunnerIdempotencyConflict
	}
	return runner, nil
}

var errFleetDisabled = errors.New("fleet is disabled")

func loadFleetCapacity(ctx context.Context, fleet *models.RunnerFleet) (*pb.GetFleetCapacityResponse, error) {
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

	return &pb.GetFleetCapacityResponse{
		RunnableTasks:     runnable,
		PendingRunners:    counts[models.RunnerStatePending],
		IdleRunners:       counts[models.RunnerStateIdle],
		BusyRunners:       counts[models.RunnerStateBusy],
		TerminatedRunners: counts[models.RunnerStateTerminated],
		Generation:        hex.EncodeToString(generationBytes[:]),
	}, nil
}

func findFleet(ctx context.Context, rawID string) (*models.RunnerFleet, error) {
	id, err := parseFleetID(rawID)
	if err != nil {
		return nil, err
	}
	fleet, err := models.FindInstallationRunnerFleet(database.DB(ctx), id)
	if errors.Is(err, models.ErrRunnerFleetNotFound) {
		return nil, grpcerrors.NotFound(err, "fleet not found")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to find fleet")
	}
	return fleet, nil
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

func parseFleetID(value string) (string, error) {
	id := strings.TrimSpace(value)
	if id == "" {
		return "", grpcerrors.InvalidArgument(nil, "fleet ID is required")
	}
	return id, nil
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

func validateTaskStates(states []string) error {
	valid := []string{
		models.RunnerTaskStateQueued,
		models.RunnerTaskStateReserved,
		models.RunnerTaskStateRunning,
		models.RunnerTaskStateSucceeded,
		models.RunnerTaskStateFailed,
		models.RunnerTaskStateCanceled,
		models.RunnerTaskStateLost,
	}
	for _, state := range states {
		if !slices.Contains(valid, state) {
			return grpcerrors.InvalidArgument(nil, "invalid task state")
		}
	}
	return nil
}
