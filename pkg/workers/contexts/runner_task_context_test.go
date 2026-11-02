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
