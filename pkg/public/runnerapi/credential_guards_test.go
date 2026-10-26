package runnerapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRunnerPreparationRejectsResourceRevocationAtAcceptanceAndDispatch(t *testing.T) {
	for _, phase := range []string{"before acceptance", "after acceptance"} {
		t.Run(phase, func(t *testing.T) {
			fixture := createRunnerProviderFixture(t)
			db := database.DB(t.Context())
			integration := models.NewIntegration(uuid.New(), fixture.task.OrganizationID, "example", "saved integration", nil)
			integration.State = models.IntegrationStateReady
			require.NoError(t, db.Create(integration).Error)
			payload, err := json.Marshal(runnerpayload.TaskPayload{ID: fixture.task.ID.String(), Environment: []runnerpayload.EnvironmentVariable{{Name: "TOKEN", Value: "prepared-token"}}, CredentialRefreshes: []runnerpayload.CredentialRefresh{{Version: 1, Kind: "integration", OrganizationID: fixture.task.OrganizationID.String(), ResourceID: integration.ID.String(), Recipient: integration.AppName, Scope: map[string]string{"key": "TOKEN", "identity": workercontexts.IntegrationCredentialIdentity(integration)}, Fields: []runnerpayload.CredentialField{{Path: []string{"environment", "0", "value"}, Original: "prepared-token"}}}}})
			require.NoError(t, err)
			guard := func(tx *gorm.DB) error { return fixture.server.validateRefreshResources(tx, fixture.task, payload) }
			if phase == "after acceptance" {
				require.NoError(t, fixture.preparation.Accept(db, fixture.runnerID, fixture.connectionID, payload, time.Now().Add(time.Hour), time.Now(), guard))
			}
			require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateError).Error)
			if phase == "before acceptance" {
				err = fixture.preparation.Accept(db, fixture.runnerID, fixture.connectionID, payload, time.Now().Add(time.Hour), time.Now(), guard)
			} else {
				err = fixture.task.BeginDispatch(db, fixture.runnerID, fixture.connectionID, fixture.preparation, time.Now(), guard)
			}
			require.ErrorIs(t, err, errInvalidCredentialDescriptor)
			require.NoError(t, fixture.task.FailPreparation(db, "Runner credentials are no longer authorized.", fixture.preparation, time.Now()))
			failed, err := models.FindRunnerTask(db, fixture.task.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStateFailed, failed.State)
			assert.Nil(t, failed.DispatchStartedAt)
			assert.Nil(t, failed.StartedAt)
		})
	}
}

func TestRunnerPreparationDoesNotFollowReplacedIntegrationName(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	integration := models.NewIntegration(uuid.New(), fixture.task.OrganizationID, "example", "unchanged name", nil)
	integration.State = models.IntegrationStateReady
	require.NoError(t, db.Create(integration).Error)
	descriptor := runnerpayload.CredentialRefresh{Version: 1, Kind: "integration", OrganizationID: fixture.task.OrganizationID.String(), ResourceID: integration.ID.String(), Recipient: integration.AppName, Scope: map[string]string{"key": "TOKEN", "identity": workercontexts.IntegrationCredentialIdentity(integration)}, Fields: []runnerpayload.CredentialField{{Path: []string{"environment", "0", "value"}, Original: "old"}}}
	savedName := integration.InstallationName
	require.NoError(t, integration.SoftDeleteInTransaction(db))
	replacement := models.NewIntegration(uuid.New(), fixture.task.OrganizationID, "example", savedName, nil)
	replacement.State = models.IntegrationStateReady
	require.NoError(t, db.Create(replacement).Error)
	payload, err := json.Marshal(runnerpayload.TaskPayload{CredentialRefreshes: []runnerpayload.CredentialRefresh{descriptor}})
	require.NoError(t, err)
	require.ErrorIs(t, fixture.server.validateRefreshResources(db, fixture.task, payload), errInvalidCredentialDescriptor)
}

func TestRunnerMCPRefreshRejectsChangedTokenRecipient(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, fixture.task.OrganizationID, "workspace", "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "saved-server", true, models.FactoryAgentResourceConfig{URL: "https://mcp.example/tools", Auth: models.FactoryAgentResourceAuthOAuth, ToolsDefaultApplied: true})
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Microsecond)
	metadata := models.FactoryAgentResourceOAuthMetadata{ClientID: "same-client", Resource: "https://mcp.example/tools", TokenEndpoint: "https://authorization.example/token"}
	require.NoError(t, db.Model(resource).Updates(map[string]any{"oauth_status": models.FactoryAgentResourceOAuthConnected, "oauth_connected_at": now, "oauth_metadata": datatypes.NewJSONType(metadata)}).Error)
	descriptor := runnerpayload.CredentialRefresh{Version: 1, Kind: "mcp_oauth", OrganizationID: fixture.task.OrganizationID.String(), ResourceID: resource.ID.String(), Recipient: resource.Config.Data().URL, Scope: map[string]string{"factory_id": factory.ID.String(), "auth": models.FactoryAgentResourceAuthOAuth, "client_id": metadata.ClientID, "resource": metadata.Resource, "connected_at": now.Format(time.RFC3339Nano), "token_endpoint": metadata.TokenEndpoint}, Fields: []runnerpayload.CredentialField{{Path: []string{"files", "0", "content"}, Original: "previous-token"}}}
	metadata.TokenEndpoint = "https://replacement.example/token"
	require.NoError(t, db.Model(resource).Update("oauth_metadata", datatypes.NewJSONType(metadata)).Error)
	called := false
	httpClient := &preparationHTTP{ctx: t.Context(), http: runnerCredentialHTTPFunc(func(*http.Request) (*http.Response, error) { called = true; return runnerCredentialResponse(`{}`), nil })}
	_, err = fixture.server.refreshCredential(t.Context(), fixture.task, fixture.preparation, descriptor, time.Hour, httpClient)
	require.ErrorIs(t, err, errInvalidCredentialDescriptor)
	assert.False(t, called, "replacement recipients must never receive saved refresh credentials")
}

func TestRunnerMCPRefreshChangesOnlyEachSavedHeader(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, fixture.task.OrganizationID, "workspace", "", "")
	require.NoError(t, err)
	secret := &models.Secret{ID: uuid.New(), DomainType: models.DomainTypeOrganization, DomainID: fixture.task.OrganizationID, Name: "mcp-auth", Data: []byte(`{"TOKEN":"old-secret"}`)}
	require.NoError(t, db.Create(secret).Error)
	producer := workercontexts.NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), fixture.task.OrganizationID)
	var servers []map[string]any
	for _, name := range []string{"first", "second"} {
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, name, true, models.FactoryAgentResourceConfig{URL: "https://" + name + ".old-secret.example/mcp", Auth: models.FactoryAgentResourceAuthHeaders, ToolsDefaultApplied: true, Headers: []models.FactoryAgentResourceHeader{{Name: "Authorization", SecretName: secret.Name, SecretKey: "TOKEN"}}})
		require.NoError(t, err)
		producer.RecordCredentialRefresh(runnerpayload.CredentialRefresh{Version: 1, Kind: "mcp_header", OrganizationID: fixture.task.OrganizationID.String(), ResourceID: resource.ID.String(), Recipient: resource.Config.Data().URL, Scope: map[string]string{"name": name, "factory_id": factory.ID.String(), "secret_id": secret.ID.String(), "key": "TOKEN", "header": "Authorization", "auth": models.FactoryAgentResourceAuthHeaders}, Current: "old-secret"})
		servers = append(servers, map[string]any{"name": name, "url": resource.Config.Data().URL, "headers": map[string]string{"Authorization": "old-secret"}})
	}
	file, err := json.Marshal(map[string]any{"servers": servers})
	require.NoError(t, err)
	jobID := uuid.New()
	saved := runnerpayload.TaskPayload{ID: jobID.String(), Script: "echo old-secret is saved prompt text", Files: []runnerpayload.TaskFile{{Path: "workspace_mcp.json", Content: string(file)}}}
	payload, err := json.Marshal(saved)
	require.NoError(t, err)
	fleet, err := models.FindRunnerFleet(db, fixture.task.FleetID)
	require.NoError(t, err)
	require.NoError(t, producer.Create(jobID.String(), fleet.Slug, payload))
	job, err := models.FindRunnerTask(db, jobID)
	require.NoError(t, err)
	require.NoError(t, db.Model(secret).Update("data", []byte(`{"TOKEN":"new-secret"}`)).Error)
	prepared, err := fixture.server.refreshTaskPayload(t.Context(), job, fixture.preparation, &preparationHTTP{ctx: t.Context()})
	require.NoError(t, err)
	var actual runnerpayload.TaskPayload
	require.NoError(t, json.Unmarshal(prepared, &actual))
	assert.Equal(t, saved.Script, actual.Script)
	var config struct {
		Servers []struct {
			Name    string            `json:"name"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(actual.Files[0].Content), &config))
	require.Len(t, config.Servers, 2)
	for _, server := range config.Servers {
		assert.Equal(t, "https://"+server.Name+".old-secret.example/mcp", server.URL)
		assert.Equal(t, "new-secret", server.Headers["Authorization"])
	}
}

func TestRunnerMCPAcceptedPayloadExpiresBeforeRunLifetimeBecomesTooShort(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, fixture.task.OrganizationID, "workspace", "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "saved-server", true, models.FactoryAgentResourceConfig{URL: "https://mcp.example/tools", Auth: models.FactoryAgentResourceAuthOAuth, ToolsDefaultApplied: true})
	require.NoError(t, err)
	connectedAt := time.Now().UTC().Truncate(time.Microsecond)
	expiresAt := connectedAt.Add(time.Hour).Truncate(time.Second)
	metadata := models.FactoryAgentResourceOAuthMetadata{ClientID: "same-client", Resource: "https://mcp.example/tools", TokenEndpoint: "https://authorization.example/token", AccessTokenExpiresAt: expiresAt.Format(time.RFC3339)}
	require.NoError(t, db.Model(resource).Updates(map[string]any{"oauth_status": models.FactoryAgentResourceOAuthConnected, "oauth_connected_at": connectedAt, "oauth_metadata": datatypes.NewJSONType(metadata)}).Error)
	require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretAccessToken, []byte("cached-access")))
	timeout := 3600
	descriptor := runnerpayload.CredentialRefresh{Version: 1, Kind: "mcp_oauth", OrganizationID: fixture.task.OrganizationID.String(), ResourceID: resource.ID.String(), Recipient: resource.Config.Data().URL, Scope: map[string]string{"factory_id": factory.ID.String(), "auth": models.FactoryAgentResourceAuthOAuth, "client_id": metadata.ClientID, "resource": metadata.Resource, "connected_at": connectedAt.Format(time.RFC3339Nano), "token_endpoint": metadata.TokenEndpoint}, Fields: []runnerpayload.CredentialField{{Path: []string{"environment", "0", "value"}, Original: "placeholder"}}}
	saved, err := json.Marshal(runnerpayload.TaskPayload{ID: fixture.task.ID.String(), ExecutionTimeoutSeconds: &timeout, Environment: []runnerpayload.EnvironmentVariable{{Name: "MCP_TOKEN", Value: "placeholder"}}, CredentialRefreshes: []runnerpayload.CredentialRefresh{descriptor}})
	require.NoError(t, err)
	fixture.task.PayloadCiphertext = saved
	prepared, err := fixture.server.refreshTaskPayload(t.Context(), fixture.task, fixture.preparation, &preparationHTTP{ctx: t.Context(), http: runnerCredentialHTTPFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("a fresh one-hour token should be reusable for this one-hour job")
		return nil, nil
	})})
	require.NoError(t, err)
	validUntil := preparedPayloadValidUntil(prepared)
	require.True(t, validUntil.After(time.Now()))
	require.True(t, validUntil.Before(time.Now().Add(2*time.Minute)))
	require.NoError(t, fixture.preparation.Accept(db, fixture.runnerID, fixture.connectionID, prepared, validUntil, time.Now()))
	// The token still exists and has nearly an hour left, but the accepted
	// payload must be refreshed once its delivery tolerance is exhausted.
	err = fixture.task.BeginDispatch(db, fixture.runnerID, fixture.connectionID, fixture.preparation, time.Now().Add(2*time.Minute))
	require.ErrorIs(t, err, models.ErrRunnerPreparationUnavailable)
}
