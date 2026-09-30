package runners

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func TestCreateRunnerIsIdempotentAndReservesTask(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	task := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted secret"))

	ctx := authentication.SetAccountIDInMetadata(t.Context(), resource.Account.ID.String())
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	request := &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		TaskId:         stringPointer(task.ID.String()),
		IdempotencyKey: "request-1",
		Ephemeral:      true,
	}

	first, err := service.CreateRunner(ctx, request)
	require.NoError(t, err)
	second, err := service.CreateRunner(ctx, request)
	require.NoError(t, err)

	assert.Equal(t, first.Runner.Id, second.Runner.Id)
	assert.Equal(t, first.RegistrationToken, second.RegistrationToken)
	assert.True(t, first.Runner.Ephemeral)
	assert.Equal(t, "https://example.com/runner/v1", first.RunnerApiUrl)
	assert.Equal(t, "runner-"+first.Runner.Id, first.DisplayName)

	var runnerCount int64
	require.NoError(t, db.Model(&models.Runner{}).Count(&runnerCount).Error)
	assert.Equal(t, int64(1), runnerCount)

	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateReserved, reloadedTask.State)
	require.NotNil(t, reloadedTask.RunnerID)
	assert.Equal(t, first.Runner.Id, reloadedTask.RunnerID.String())
}

func TestCreateRunnerReplacesTerminatedIdempotentRunner(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	task := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted secret"))

	ctx := authentication.SetAccountIDInMetadata(t.Context(), resource.Account.ID.String())
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	request := &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		TaskId:         stringPointer(task.ID.String()),
		IdempotencyKey: "replacement-request",
		Ephemeral:      true,
	}

	first, err := service.CreateRunner(ctx, request)
	require.NoError(t, err)
	runner, err := models.FindRunner(db, uuid.MustParse(first.Runner.Id))
	require.NoError(t, err)
	require.NoError(t, runner.Terminate(db, models.RunnerTerminationRegistrationExpired))

	// Simulate a terminated row written before termination started releasing
	// creation keys.
	requestHash := createRunnerRequestHash(fleet.Slug, &task.ID, true)
	require.NoError(t, db.Model(runner).Updates(map[string]any{
		"creation_idempotency_key": request.IdempotencyKey,
		"creation_request_hash":    requestHash,
	}).Error)

	replacement, err := service.CreateRunner(ctx, request)
	require.NoError(t, err)
	assert.NotEqual(t, first.Runner.Id, replacement.Runner.Id)
	assert.Equal(t, models.RunnerStatePending, replacement.Runner.State)
}

func TestCreateGenericRunnerWithoutIdempotencyKeyCreatesNewRunnerEachTime(t *testing.T) {
	support.Setup(t)
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	request := &pb.CreateRunnerRequest{FleetId: fleet.Slug}

	first, err := service.CreateRunner(t.Context(), request)
	require.NoError(t, err)
	second, err := service.CreateRunner(t.Context(), request)
	require.NoError(t, err)

	assert.NotEqual(t, first.Runner.Id, second.Runner.Id)
	assert.False(t, first.Runner.Ephemeral)
	assert.False(t, second.Runner.Ephemeral)
}

func TestCreateGenericEphemeralRunner(t *testing.T) {
	support.Setup(t)
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")

	response, err := service.CreateRunner(t.Context(), &pb.CreateRunnerRequest{
		FleetId:   fleet.Slug,
		Ephemeral: true,
	})
	require.NoError(t, err)
	assert.True(t, response.Runner.Ephemeral)
}

func TestCreateFleetCreatesInstallationFleet(t *testing.T) {
	support.Setup(t)
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	spec := &pb.FleetSpec{
		OperatingSystem:            "linux",
		Architecture:               "amd64",
		CpuMillicores:              8000,
		MemoryMb:                   32768,
		DiskGb:                     30,
		Capabilities:               []string{"docker"},
		MaxExecutionTimeoutSeconds: 3600,
	}

	response, err := service.CreateFleet(t.Context(), &pb.CreateFleetRequest{
		FleetId:       " aws-large-amd64 ",
		Spec:          spec,
		RunnerVersion: "v0.0.1",
	})
	require.NoError(t, err)
	assert.Equal(t, "aws-large-amd64", response.Fleet.Id)
	assert.Equal(t, "v0.0.1", response.Fleet.RunnerVersion)
	assert.True(t, response.Fleet.Enabled)
	assert.Equal(t, spec, response.Fleet.Spec)

	reloaded, err := models.FindInstallationRunnerFleet(
		database.DB(t.Context()),
		"aws-large-amd64",
	)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerFleetScopeInstallation, reloaded.ScopeType)
	assert.Nil(t, reloaded.ScopeID)
	assert.Equal(t, int32(32768), reloaded.Spec.Data().MemoryMB)
}

func TestCreateFleetRejectsDuplicateID(t *testing.T) {
	support.Setup(t)
	createTestFleetWithSlug(t, "aws-large-amd64")
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")

	_, err := service.CreateFleet(t.Context(), &pb.CreateFleetRequest{
		FleetId:       "aws-large-amd64",
		Spec:          &pb.FleetSpec{OperatingSystem: "linux", Architecture: "amd64"},
		RunnerVersion: "v0.0.1",
	})

	assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func TestCreateFleetValidatesRequiredFields(t *testing.T) {
	support.Setup(t)
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	requests := map[string]*pb.CreateFleetRequest{
		"fleet ID": {
			Spec:          &pb.FleetSpec{},
			RunnerVersion: "v0.0.1",
		},
		"spec": {
			FleetId:       "aws-large-amd64",
			RunnerVersion: "v0.0.1",
		},
		"runner version": {
			FleetId: "aws-large-amd64",
			Spec:    &pb.FleetSpec{},
		},
	}

	for name, request := range requests {
		t.Run(name, func(t *testing.T) {
			_, err := service.CreateFleet(t.Context(), request)
			assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		})
	}
}

func TestUpdateFleetReplacesSpec(t *testing.T) {
	support.Setup(t)
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	spec := &pb.FleetSpec{
		OperatingSystem:            "linux",
		Architecture:               "arm64",
		CpuMillicores:              4000,
		MemoryMb:                   8192,
		DiskGb:                     50,
		Capabilities:               []string{"docker"},
		MaxExecutionTimeoutSeconds: 7200,
	}

	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	response, err := service.UpdateFleet(t.Context(), &pb.UpdateFleetRequest{
		FleetId: fleet.Slug,
		Spec:    spec,
	})
	require.NoError(t, err)
	assert.Equal(t, spec, response.Fleet.Spec)

	reloaded, err := models.FindInstallationRunnerFleet(database.DB(t.Context()), fleet.Slug)
	require.NoError(t, err)
	assert.Equal(t, "arm64", reloaded.Spec.Data().Architecture)
	assert.Equal(t, int32(8192), reloaded.Spec.Data().MemoryMB)
}

func TestCreateTaskSpecificRunnerRequiresIdempotencyKey(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	task := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted secret"))
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")

	_, err := service.CreateRunner(t.Context(), &pb.CreateRunnerRequest{
		FleetId:   fleet.Slug,
		TaskId:    stringPointer(task.ID.String()),
		Ephemeral: true,
	})

	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	var runnerCount int64
	require.NoError(t, db.Model(&models.Runner{}).Count(&runnerCount).Error)
	assert.Zero(t, runnerCount)
}

func TestCreateTaskSpecificRunnerMustBeEphemeral(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	task := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted secret"))
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")

	_, err := service.CreateRunner(t.Context(), &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		TaskId:         stringPointer(task.ID.String()),
		IdempotencyKey: "request-1",
	})

	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	var runnerCount int64
	require.NoError(t, db.Model(&models.Runner{}).Count(&runnerCount).Error)
	assert.Zero(t, runnerCount)
}

func TestCreateRunnerRejectsReusedIdempotencyKeyForDifferentRequest(t *testing.T) {
	resource := support.Setup(t)
	firstFleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	secondFleet := createTestFleetWithSlug(t, "other-linux-amd64")
	ctx := authentication.SetAccountIDInMetadata(t.Context(), resource.Account.ID.String())
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")

	_, err := service.CreateRunner(ctx, &pb.CreateRunnerRequest{
		FleetId:        firstFleet.Slug,
		IdempotencyKey: "request-1",
	})
	require.NoError(t, err)

	_, err = service.CreateRunner(ctx, &pb.CreateRunnerRequest{
		FleetId:        secondFleet.Slug,
		IdempotencyKey: "request-1",
	})
	assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func TestCreateRunnerRejectsReusedIdempotencyKeyForDifferentLifecycle(t *testing.T) {
	support.Setup(t)
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")

	_, err := service.CreateRunner(t.Context(), &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		IdempotencyKey: "request-1",
	})
	require.NoError(t, err)

	_, err = service.CreateRunner(t.Context(), &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		IdempotencyKey: "request-1",
		Ephemeral:      true,
	})
	assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func TestCreateTaskSpecificRunnerHandlesConcurrentIdempotentRequests(t *testing.T) {
	resource := support.Setup(t)
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	task := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted secret"))

	ctx := authentication.SetAccountIDInMetadata(t.Context(), resource.Account.ID.String())
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	request := &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		TaskId:         stringPointer(task.ID.String()),
		IdempotencyKey: "concurrent-request",
		Ephemeral:      true,
	}

	start := make(chan struct{})
	responses := make([]*pb.CreateRunnerResponse, 2)
	errs := make([]error, 2)
	var group sync.WaitGroup
	for i := range responses {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			responses[index], errs[index] = service.CreateRunner(ctx, request)
		}(i)
	}
	close(start)
	group.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, responses[0].Runner.Id, responses[1].Runner.Id)
	assert.Equal(t, responses[0].RegistrationToken, responses[1].RegistrationToken)
}

func TestListRunnersOnlyReturnsRunnersForFleet(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	installationFleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	organizationFleet := createTestFleet(t, models.RunnerFleetScopeOrganization, &resource.Organization.ID)

	now := time.Now()
	installationRunner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       installationFleet.ID,
		State:         models.RunnerStatePending,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(installationRunner).Error)
	organizationRunner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       organizationFleet.ID,
		State:         models.RunnerStatePending,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(organizationRunner).Error)

	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	response, err := service.ListRunners(t.Context(), &pb.ListRunnersRequest{
		FleetId: installationFleet.Slug,
	})
	require.NoError(t, err)
	require.Len(t, response.Runners, 1)
	assert.Equal(t, installationRunner.ID.String(), response.Runners[0].Id)
}

func TestDeleteRunnerRejectsBusyRunnerWithConflict(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateBusy,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)

	ctx := authentication.SetAccountIDInMetadata(t.Context(), resource.Account.ID.String())
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	_, err := service.DeleteRunner(ctx, &pb.DeleteRunnerRequest{
		FleetId:  fleet.Slug,
		RunnerId: runner.ID.String(),
	})

	assert.Equal(t, codes.Aborted, grpcerrors.Code(err))
}

func TestDeletePendingTaskRunnerReleasesReservation(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	task := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted secret"))

	ctx := authentication.SetAccountIDInMetadata(t.Context(), resource.Account.ID.String())
	service := NewService(jwt.NewSigner("runner-registration-secret"), "https://example.com")
	created, err := service.CreateRunner(ctx, &pb.CreateRunnerRequest{
		FleetId:        fleet.Slug,
		TaskId:         stringPointer(task.ID.String()),
		IdempotencyKey: "task-runner-to-delete",
		Ephemeral:      true,
	})
	require.NoError(t, err)

	_, err = service.DeleteRunner(ctx, &pb.DeleteRunnerRequest{
		FleetId:  fleet.Slug,
		RunnerId: created.Runner.Id,
	})
	require.NoError(t, err)

	reloaded, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, reloaded.State)
	assert.Nil(t, reloaded.RunnerID)
	assert.Nil(t, reloaded.ReservedAt)

	var uploads int64
	require.NoError(t, db.Model(&models.TaskLogUpload{}).
		Where("task_id = ?", task.ID).
		Count(&uploads).
		Error)
	assert.Zero(t, uploads)
}

func createTestFleet(t *testing.T, scope string, scopeID *uuid.UUID) *models.RunnerFleet {
	t.Helper()
	slug := "linux-amd64"
	if scope == models.RunnerFleetScopeOrganization {
		slug = "organization-linux-amd64"
	}
	return createTestFleetRecord(t, &models.RunnerFleet{
		ID:        uuid.New(),
		ScopeType: scope,
		ScopeID:   scopeID,
		Slug:      slug,
		Enabled:   true,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{
			Architecture: "amd64",
			Capabilities: []string{"docker"},
		}),
		RunnerVersion: "0.1.0",
	})
}

func createTestFleetWithSlug(t *testing.T, slug string) *models.RunnerFleet {
	t.Helper()
	return createTestFleetRecord(t, &models.RunnerFleet{
		ID:        uuid.New(),
		ScopeType: models.RunnerFleetScopeInstallation,
		Slug:      slug,
		Enabled:   true,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{
			Architecture: "amd64",
		}),
		RunnerVersion: "0.1.0",
	})
}

func createTestFleetRecord(t *testing.T, fleet *models.RunnerFleet) *models.RunnerFleet {
	t.Helper()
	require.NoError(t, fleet.Create(database.DB(t.Context())))
	return fleet
}

func createTestTask(
	t *testing.T,
	organizationID, fleetID uuid.UUID,
	payloadCiphertext []byte,
) *models.RunnerTask {
	t.Helper()
	now := time.Now()
	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    organizationID,
		FleetID:           fleetID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: payloadCiphertext,
		QueuedAt:          now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, database.DB(t.Context()).Create(task).Error)
	return task
}

func stringPointer(value string) *string {
	return &value
}
