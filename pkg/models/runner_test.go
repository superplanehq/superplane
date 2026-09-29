package models_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestInstallationRunnerFleetLifecycle(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())

	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	list, err := models.ListInstallationRunnerFleets(db)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, fleet.ID, list[0].ID)

	require.NoError(t, fleet.PinRunnerVersion(db, "0.2.0"))

	reloaded, err := models.FindInstallationRunnerFleet(db, fleet.Slug)
	require.NoError(t, err)
	assert.Equal(t, "0.2.0", reloaded.RunnerVersion)
}

func TestOrganizationRunnerFleetIsNotListedAsInstallationFleet(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())

	fleet := newTestRunnerFleet()
	fleet.ScopeType = models.RunnerFleetScopeOrganization
	fleet.ScopeID = &resource.Organization.ID
	require.NoError(t, fleet.Create(db))

	list, err := models.ListInstallationRunnerFleets(db)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestRunnerFleetIDCanBeReusedByDifferentOrganizations(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	firstOrganization, err := models.CreateOrganization("First Fleet Organization", "")
	require.NoError(t, err)
	secondOrganization, err := models.CreateOrganization("Second Fleet Organization", "")
	require.NoError(t, err)
	db := database.DB(t.Context())

	firstFleet := newOrganizationTestRunnerFleet(firstOrganization.ID, "s1-hello")
	require.NoError(t, firstFleet.Create(db))
	secondFleet := newOrganizationTestRunnerFleet(secondOrganization.ID, "s1-hello")
	require.NoError(t, secondFleet.Create(db))
}

func TestOrganizationRunnerFleetIDCannotConflictWithInstallationFleet(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("Fleet Conflict Organization", "")
	require.NoError(t, err)
	db := database.DB(t.Context())

	installationFleet := newTestRunnerFleet()
	installationFleet.Slug = "s1-hello"
	require.NoError(t, installationFleet.Create(db))

	organizationFleet := newOrganizationTestRunnerFleet(organization.ID, "s1-hello")
	assert.ErrorIs(t, organizationFleet.Create(db), models.ErrRunnerFleetIDConflict)
}

func TestInstallationRunnerFleetIDCannotConflictWithOrganizationFleet(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("Reverse Fleet Conflict Organization", "")
	require.NoError(t, err)
	db := database.DB(t.Context())

	organizationFleet := newOrganizationTestRunnerFleet(organization.ID, "s1-hello")
	require.NoError(t, organizationFleet.Create(db))

	installationFleet := newTestRunnerFleet()
	installationFleet.Slug = "s1-hello"
	assert.ErrorIs(t, installationFleet.Create(db), models.ErrRunnerFleetIDConflict)
}

func TestConcurrentRunnerFleetCreatesEnforceInstallationReservation(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("Concurrent Fleet Organization", "")
	require.NoError(t, err)

	fleets := []*models.RunnerFleet{
		newTestRunnerFleet(),
		newOrganizationTestRunnerFleet(organization.ID, "linux-amd64"),
	}
	errs := make([]error, len(fleets))
	start := make(chan struct{})
	var group sync.WaitGroup
	for i := range fleets {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			errs[index] = fleets[index].Create(database.DB(t.Context()))
		}(i)
	}
	close(start)
	group.Wait()

	successes := 0
	conflicts := 0
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, models.ErrRunnerFleetIDConflict):
			conflicts++
		default:
			require.NoError(t, err)
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, conflicts)
}

func TestRunnerTaskReservationAndRunnerTermination(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())

	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	idempotencyKey := "fleet-manager/linux-amd64/task/task-1"
	requestHash := "request-hash"
	runner := &models.Runner{
		ID:                     uuid.New(),
		FleetID:                fleet.ID,
		State:                  models.RunnerStatePending,
		RunnerVersion:          "0.1.0",
		CreationIdempotencyKey: &idempotencyKey,
		CreationRequestHash:    &requestHash,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	require.NoError(t, db.Create(runner).Error)

	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))

	assert.Equal(t, models.RunnerTaskStateReserved, task.State)
	require.NotNil(t, task.RunnerID)
	assert.Equal(t, runner.ID, *task.RunnerID)

	require.NoError(t, db.Model(runner).Update("state", models.RunnerStateIdle).Error)
	runner.State = models.RunnerStateIdle
	require.NoError(t, runner.Terminate(db, models.RunnerTerminationRequested))
	assert.Equal(t, models.RunnerStateTerminated, runner.State)
	require.NotNil(t, runner.TerminatedAt)
	assert.Nil(t, runner.CreationIdempotencyKey)
	assert.Nil(t, runner.CreationRequestHash)

	persisted, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Nil(t, persisted.CreationIdempotencyKey)
	assert.Nil(t, persisted.CreationRequestHash)

	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, reloadedTask.State)
	assert.Nil(t, reloadedTask.RunnerID)
}

func TestBusyRunnerCannotBeTerminated(t *testing.T) {
	runner := &models.Runner{ID: uuid.New(), State: models.RunnerStateBusy}
	err := runner.Terminate(database.DB(t.Context()), models.RunnerTerminationRequested)
	assert.ErrorIs(t, err, models.ErrRunnerBusy)
}

func newTestRunnerFleet() *models.RunnerFleet {
	return &models.RunnerFleet{
		ID:        uuid.New(),
		ScopeType: models.RunnerFleetScopeInstallation,
		Slug:      "linux-amd64",
		Enabled:   true,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{
			Architecture: "amd64",
			Capabilities: []string{"docker"},
		}),
		RunnerVersion: "0.1.0",
	}
}

func newOrganizationTestRunnerFleet(organizationID uuid.UUID, slug string) *models.RunnerFleet {
	fleet := newTestRunnerFleet()
	fleet.ID = uuid.New()
	fleet.ScopeType = models.RunnerFleetScopeOrganization
	fleet.ScopeID = &organizationID
	fleet.Slug = slug
	return fleet
}
