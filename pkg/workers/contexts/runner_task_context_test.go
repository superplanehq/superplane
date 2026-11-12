package contexts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRunnerTaskContextAddsDefaultLogUploadPolicy(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	db := database.DB(t.Context())

	fleet := models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          "runner-task-context-test",
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       true,
		Spec:          datatypes.NewJSONType(models.RunnerFleetSpec{}),
		RunnerVersion: models.DefaultRunnerVersion,
	}
	require.NoError(t, fleet.Create(db))

	taskID := uuid.New()
	payload, err := json.Marshal(runnerapi.TaskPayload{ID: taskID.String()})
	require.NoError(t, err)

	context := NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), resource.Organization.ID)
	require.NoError(t, context.Create(taskID.String(), fleet.Slug, payload))

	task, err := models.FindRunnerTask(db, taskID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStatePending, task.State)
	assert.Nil(t, task.ReadyAt)
	var stored runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(task.PayloadCiphertext, &stored))
	require.NotNil(t, stored.LogUploadPolicy)
	assert.Equal(t, runnerapi.DefaultLogUploadPolicy(), *stored.LogUploadPolicy)
}

func TestRunnerTaskContextRejectsPaidFleetWithoutCredit(t *testing.T) {
	t.Setenv(billing.BillingEnabledEnv, "yes")
	resource := support.Setup(t)
	defer resource.Close()
	db := database.DB(t.Context())
	support.AssignOpenSystemTrial(t, resource.Organization.ID)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ?", resource.Organization.ID).
		Update("expires_at", time.Now().Add(-time.Minute)).Error)

	for _, slug := range []string{models.RunnerFleetE1LargeAMD64, "local"} {
		fleet := models.RunnerFleet{
			ID: uuid.New(), Slug: slug, ScopeType: models.RunnerFleetScopeInstallation,
			Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion,
		}
		require.NoError(t, fleet.Create(db))
		taskID := uuid.New()
		payload, err := json.Marshal(runnerapi.TaskPayload{ID: taskID.String()})
		require.NoError(t, err)
		ctx := NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), resource.Organization.ID)
		err = ctx.Create(taskID.String(), fleet.Slug, payload)
		if slug == "local" {
			require.NoError(t, err)
			continue
		}
		require.ErrorIs(t, err, billing.ErrHostedCreditEmpty)
		require.ErrorIs(t, db.First(&models.RunnerTask{}, "id = ?", taskID).Error, gorm.ErrRecordNotFound)
	}
}

func TestRunnerTaskContextReadsImportedTerminalTasks(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	db := database.DB(t.Context())
	fleet := models.RunnerFleet{
		ID: uuid.New(), Slug: "imported-context", ScopeType: models.RunnerFleetScopeInstallation,
		Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion,
	}
	require.NoError(t, fleet.Create(db))
	otherOrganization := support.CreateOrganization(t, resource, resource.UserModel.ID)
	for _, state := range []string{models.RunnerTaskStateSucceeded, models.RunnerTaskStateFailed, models.RunnerTaskStateCanceled} {
		t.Run(state, func(t *testing.T) {
			now := time.Now()
			exitCode := int32(1)
			if state == models.RunnerTaskStateSucceeded {
				exitCode = 0
			}
			task := models.RunnerTask{
				ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID,
				Backend: models.RunnerTaskBackendLegacy, State: state, PayloadCiphertext: []byte{},
				Result: datatypes.JSON(`{"historical":true}`), ExitCode: &exitCode,
				FinishedAt: &now, QueuedAt: now.Add(-time.Minute), CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, db.Create(&task).Error)
			ctx := NewRunnerTaskContext(db, nil, resource.Organization.ID)
			found, err := ctx.Find(task.ID.String())
			require.NoError(t, err)
			assert.Equal(t, task.ID.String(), found.ID)
			assert.Equal(t, state, found.State)
			assert.JSONEq(t, `{"historical":true}`, string(found.Result))
			require.NotNil(t, found.ExitCode)
			assert.Equal(t, int(exitCode), *found.ExitCode)
			require.NoError(t, ctx.RequestCancel(task.ID.String()))
			persisted, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			assert.Equal(t, state, persisted.State)
			assert.Equal(t, models.RunnerTaskBackendLegacy, persisted.Backend)
			assert.Nil(t, persisted.CancelRequestedAt)
			foreign := NewRunnerTaskContext(db, nil, otherOrganization.ID)
			_, err = foreign.Find(task.ID.String())
			assert.ErrorIs(t, err, models.ErrRunnerTaskNotFound)
			assert.ErrorIs(t, foreign.RequestCancel(task.ID.String()), models.ErrRunnerTaskNotFound)
		})
	}
}

func TestRunnerTaskSnapshotKeepsPollAndCompletionConsistent(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := models.RunnerFleet{ID: uuid.New(), Slug: support.RandomName("snapshot"), ScopeType: models.RunnerFleetScopeInstallation,
		Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion}
	require.NoError(t, fleet.Create(db))
	now := time.Now()
	task := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStatePending, PayloadCiphertext: []byte{},
		QueuedAt: now, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(&task).Error)
	snapshot := NewRunnerTaskContext(db, nil, resource.Organization.ID).WithSnapshot(task.ID.String())
	// The agent can report its last check and finish after hook preparation.
	require.NoError(t, db.Model(&task).Update("state", models.RunnerTaskStateSucceeded).Error)
	polled, err := snapshot.Find(task.ID.String())
	require.NoError(t, err)
	assert.False(t, polled.IsTerminal(), "this hook must keep polling until its next snapshot")
	next, err := NewRunnerTaskContext(db, nil, resource.Organization.ID).WithSnapshot(task.ID.String()).Find(task.ID.String())
	require.NoError(t, err)
	assert.True(t, next.IsTerminal())
}
