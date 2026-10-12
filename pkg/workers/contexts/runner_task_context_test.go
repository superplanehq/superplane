package contexts

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
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
	var stored runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(task.PayloadCiphertext, &stored))
	require.NotNil(t, stored.LogUploadPolicy)
	assert.Equal(t, runnerapi.DefaultLogUploadPolicy(), *stored.LogUploadPolicy)
}
