package contexts

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github"
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

func TestRunnerTaskContextDefersTemporaryIntegrationCredentialsUntilDelivery(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	resource.Registry.Integrations["github"] = &github.GitHub{}
	integration := models.NewIntegration(uuid.New(), resource.Organization.ID, "github", "Queue GitHub", nil)
	integration.State = models.IntegrationStateReady
	// The integration requires an installation token. Missing provider credentials
	// must fail durable preparation, rather than prevent creation of a queued job.
	integration.Metadata = datatypes.NewJSONType(map[string]any{"installationId": "501", "githubApp": map[string]any{"id": 99}})
	require.NoError(t, db.Create(integration).Error)
	encryptor := crypto.NewNoOpEncryptor()
	_, err := NewSecretsContext(db, resource.Registry, resource.Organization.ID, encryptor).GetIntegrationSecrets(integration.InstallationName)
	require.Error(t, err, "ordinary integration resolution still validates credentials")

	secrets := NewSecretsContext(db, resource.Registry, resource.Organization.ID, encryptor)
	runner := NewRunnerTaskContext(db, encryptor, resource.Organization.ID).WithSecrets(secrets)
	exported, err := secrets.GetIntegrationSecrets(integration.InstallationName)
	require.NoError(t, err)
	require.NotEmpty(t, exported.Values["GITHUB_TOKEN"])
	assert.Contains(t, exported.Setup, "gh auth setup-git")
	fleet := models.DefaultInstallationRunnerFleets("dev")[0]
	require.NoError(t, fleet.Create(db))
	taskID := uuid.New()
	payload, err := json.Marshal(runnerapi.TaskPayload{ID: taskID.String(), Environment: []runnerapi.EnvironmentVariable{
		{Name: "GITHUB_TOKEN", Value: string(exported.Values["GITHUB_TOKEN"])},
	}})
	require.NoError(t, err)
	require.NoError(t, runner.Create(taskID.String(), fleet.Slug, payload))
	task, err := models.FindRunnerTask(db, taskID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, task.State)
	var saved runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(task.PayloadCiphertext, &saved))
	require.Len(t, saved.CredentialRefreshes, 1)
	descriptor := saved.CredentialRefreshes[0]
	assert.Equal(t, integration.ID.String(), descriptor.ResourceID)
	assert.Equal(t, resource.Organization.ID.String(), descriptor.OrganizationID)
	assert.Equal(t, "github", descriptor.Recipient)
	assert.Equal(t, "integration", descriptor.Kind)
	assert.Equal(t, "GITHUB_TOKEN", descriptor.Scope["key"])
	require.Len(t, descriptor.Fields, 1)
	assert.Equal(t, []string{"environment", "0", "value"}, descriptor.Fields[0].Path)
	assert.Equal(t, saved.Environment[0].Value, descriptor.Fields[0].Original)
}
