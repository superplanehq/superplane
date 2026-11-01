package contexts

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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
	assert.Equal(t, fleet.ID, task.FleetID)
	assert.Nil(t, task.ReadyAt)
	var stored runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(task.PayloadCiphertext, &stored))
	require.NotNil(t, stored.LogUploadPolicy)
	assert.Equal(t, runnerapi.DefaultLogUploadPolicy(), *stored.LogUploadPolicy)
}

func TestRunnerTaskContextResolvesOrganizationFleetWithoutFallback(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureOrganizationRunners))
	fleet := models.RunnerFleet{ID: uuid.New(), Slug: "team-runner", ScopeType: models.RunnerFleetScopeOrganization, ScopeID: &resource.Organization.ID, Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion}
	require.NoError(t, fleet.Create(db))
	context := NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), resource.Organization.ID)
	taskID := uuid.New()
	payload, err := json.Marshal(runnerapi.TaskPayload{ID: taskID.String()})
	require.NoError(t, err)
	require.NoError(t, context.Create(taskID.String(), fleet.Slug, payload))
	task, err := models.FindRunnerTask(db, taskID)
	require.NoError(t, err)
	assert.Equal(t, fleet.ID, task.FleetID)
	for _, state := range []string{"missing", "disabled", "foreign", "feature-disabled"} {
		t.Run(state, func(t *testing.T) {
			require.NoError(t, fleet.SetEnabled(db, true))
			require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureOrganizationRunners))
			caller := context
			slug := fleet.Slug
			switch state {
			case "missing":
				slug = "missing-fleet"
			case "disabled":
				require.NoError(t, fleet.SetEnabled(db, false))
			case "foreign":
				caller = NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), uuid.New())
			case "feature-disabled":
				require.NoError(t, models.DisableExperimentalFeature(resource.Organization.ID, features.FeatureOrganizationRunners))
			}
			id := uuid.New()
			require.Error(t, caller.Create(id.String(), slug, payload))
			_, err := models.FindRunnerTask(db, id)
			require.Error(t, err)
		})
	}
}
