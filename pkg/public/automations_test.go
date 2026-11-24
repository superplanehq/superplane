package public

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestAutomationAPIUsesFactoryPermissionsAndBothFlags(t *testing.T) {
	resources := support.Setup(t)
	server, token := automationAPITestServer(t, resources)
	orgID := resources.Organization.ID
	factory, err := models.CreateFactory(database.DB(t.Context()), orgID, "Automation API test", "", "AAP")
	require.NoError(t, err)

	const endpoint = "/api/v2/automations"
	list := func(factoryID uuid.UUID) int {
		return execRequest(server, requestParams{
			method: http.MethodGet, path: endpoint + "?factory_id=" + factoryID.String(),
			authCookie: token, headers: map[string]string{"X-Organization-Id": orgID.String()},
		}).Code
	}
	require.Equal(t, http.StatusUnauthorized, execRequest(server, requestParams{method: http.MethodGet, path: endpoint}).Code)

	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactoryCustomAutomations))
	require.Equal(t, http.StatusNotFound, list(factory.ID))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureAutomationsAsCode))
	require.Equal(t, http.StatusOK, list(factory.ID))
	require.Equal(t, http.StatusNotFound, list(uuid.New()))
	require.NoError(t, models.DisableExperimentalFeature(orgID, features.FeatureFactoryCustomAutomations))
	require.Equal(t, http.StatusNotFound, list(factory.ID))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactoryCustomAutomations))

	input, err := json.Marshal(automationWriteRequest{
		FactoryID: factory.ID.String(),
		Stage:     "backlog",
		YAML: `apiVersion: v2
kind: Automation
name: Inspect revisions
triggers:
  - use: pull_request_updated
steps:
  - name: Print marker
    bash: echo ready
`,
	})
	require.NoError(t, err)
	validate := func() (int, []byte) {
		response := execRequest(server, requestParams{
			method: http.MethodPost, path: endpoint + ":validate", body: input, contentType: "application/json",
			authCookie: token, headers: map[string]string{"X-Organization-Id": orgID.String()},
		})
		return response.Code, response.Body.Bytes()
	}
	code, body := validate()
	require.Equal(t, http.StatusBadRequest, code, string(body))
	var failure struct {
		Diagnostics []automationspec.Diagnostic `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(body, &failure))
	require.Len(t, failure.Diagnostics, 1)
	require.Equal(t, "triggers[0].use", failure.Diagnostics[0].Path)
	require.Equal(t, 5, failure.Diagnostics[0].Line)
	require.Positive(t, failure.Diagnostics[0].Column)

	account, err := models.CreateAccount("Automation operator", "automation-operator@example.com")
	require.NoError(t, err)
	operator, err := models.CreateUser(orgID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, resources.AuthService.AssignRole(operator.ID.String(), models.RoleOrgOperator, orgID.String(), models.DomainTypeOrganization))
	token, err = authentication.GenerateAccountToken(server.jwt, account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, list(factory.ID))
	code, _ = validate()
	require.Equal(t, http.StatusNotFound, code)
}

func TestAutomationAPIPublishesInTheSelectedStage(t *testing.T) {
	resources := support.Setup(t)
	server, token := automationAPITestServer(t, resources)

	db := database.DB(t.Context())
	orgID := resources.Organization.ID
	for _, feature := range []string{
		features.FeatureFactories, features.FeatureFactoryCustomAutomations,
		features.FeatureAutomationsAsCode,
	} {
		require.NoError(t, models.EnableExperimentalFeature(orgID, feature))
	}
	support.AssignOpenSystemTrial(t, orgID)
	provider, model := models.UsageProviderOpenRouter, "anthropic/claude-sonnet-4-6"
	const pinnedModel = "anthropic/claude-opus-5-5::revision"
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: provider, Enabled: true, APIKey: []byte("test-hosted-key"), ManagementKey: []byte("test-management-key"),
		AllowedModels: datatypes.JSONSlice[string]{model, pinnedModel},
	})
	require.NoError(t, err)
	previous, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.Background()).Model(previous).Updates(map[string]any{
			"default_hosted_provider": previous.DefaultHostedProvider,
			"default_hosted_model":    previous.DefaultHostedModel,
		}).Error)
	})
	_, err = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		DefaultHostedProvider: &provider, DefaultHostedModel: &model,
	})
	require.NoError(t, err)
	factory, err := models.CreateFactory(db, orgID, "Automation API test", "", "AAP")
	require.NoError(t, err)
	createConnection := func(app string) string {
		t.Helper()
		connection, err := models.CreateIntegration(uuid.New(), orgID, app, app+"-demo", map[string]any{})
		require.NoError(t, err)
		require.NoError(t, db.Model(connection).Update("state", models.IntegrationStateReady).Error)
		return connection.ID.String()
	}
	vcs, repository := createConnection("github"), "demo/example"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcs, AppRepository: &repository,
	}))

	const collection = "/api/v2/automations"
	call := func(method, path string, input *automationWriteRequest, wantStatus int) *httptest.ResponseRecorder {
		t.Helper()
		var body []byte
		if input != nil {
			encoded, err := json.Marshal(input)
			require.NoError(t, err)
			body = encoded
		}
		response := execRequest(server, requestParams{
			method: method, path: path, body: body, contentType: "application/json",
			authCookie: token, headers: map[string]string{"X-Organization-Id": orgID.String()},
		})
		require.Equal(t, wantStatus, response.Code, response.Body.String())
		return response
	}

	source := `# Keep this comment.
apiVersion: v2
kind: Automation
name: Print a marker
agent:
  harness: superplane
triggers:
  - use: manual
steps:
  - name: Marker
    bash: echo first
`
	input := &automationWriteRequest{FactoryID: factory.ID.String(), Stage: "backlog", YAML: source}
	for _, invalidAgent := range []string{"harness: codex", "model: unavailable", "model: byok::openrouter::" + pinnedModel} {
		invalid := *input
		invalid.YAML = strings.Replace(source, "harness: superplane", invalidAgent, 1)
		call(http.MethodPost, collection+":validate", &invalid, http.StatusBadRequest)
		call(http.MethodPost, collection, &invalid, http.StatusBadRequest)
	}
	source = strings.Replace(source, "harness: superplane", "model: "+model, 1)
	input.YAML = source
	call(http.MethodPost, collection+":validate", input, http.StatusOK)
	created := call(http.MethodPost, collection, input, http.StatusCreated)
	var first automations.Automation
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &first))
	require.Equal(t, source, first.YAML)
	require.Equal(t, factory.ID, first.FactoryID)
	require.Equal(t, "backlog", first.Stage)
	require.EqualValues(t, 1, first.Revision)

	path := collection + "/" + first.ID.String()
	require.JSONEq(t, created.Body.String(), call(http.MethodGet, path, nil, http.StatusOK).Body.String())

	source = strings.Replace(source, "echo first", "echo second", 1)
	hostedKey := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, provider, pinnedModel)
	for _, key := range []string{models.FormatHostedLLMModelKey(provider, pinnedModel), hostedKey} {
		legacy := *input
		legacy.YAML = strings.Replace(source, "model: "+model, "model: "+key, 1)
		call(http.MethodPost, collection+":validate", &legacy, http.StatusOK)
	}
	source = strings.Replace(source, "model: "+model, "model: "+pinnedModel, 1)
	updated := call(http.MethodPut, path, &automationWriteRequest{YAML: source}, http.StatusOK)
	var second automations.Automation
	require.NoError(t, json.Unmarshal(updated.Body.Bytes(), &second))
	require.Equal(t, first.Summary, second.Summary)
	require.Equal(t, source, second.YAML)
	require.EqualValues(t, 2, second.Revision)
	live, err := models.FindLiveCanvasVersionInTransaction(db, first.ID)
	require.NoError(t, err)
	require.Len(t, live.Edges, 1)
	require.Len(t, live.Nodes, 2)
	for _, node := range live.Nodes {
		if node.ID == live.Edges[0].SourceID {
			require.Equal(t, "onWorkOrder", node.ComponentName())
		}
		if node.ID == live.Edges[0].TargetID {
			require.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
			require.Equal(t, hostedKey, node.Configuration["model"])
			require.NotContains(t, node.Configuration, "credentials")
			steps, err := json.Marshal(node.Configuration["steps"])
			require.NoError(t, err)
			require.Contains(t, string(steps), "echo second")
		}
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, first.ID)
	require.NoError(t, err)
	require.NoError(t, canvas.SetColumnKey(db, models.CanvasColumnKeyVerify))
	second.Stage = models.CanvasColumnKeyVerify
	var read automations.Automation
	require.NoError(t, json.Unmarshal(call(http.MethodGet, path, nil, http.StatusOK).Body.Bytes(), &read))
	require.Equal(t, second, read)
	listed := call(http.MethodGet, collection+"?factory_id="+factory.ID.String(), nil, http.StatusOK)
	var list struct {
		Automations []automations.Summary `json:"automations"`
	}
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &list))
	require.Equal(t, []automations.Summary{second.Summary}, list.Automations)
	stageEntry := strings.Replace(source, "use: manual", "use: task_enters_stage", 1)
	call(http.MethodPut, path, &automationWriteRequest{YAML: stageEntry}, http.StatusBadRequest)
	line, err := factory.CreateLine(db, "Ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: first.ID, Entrypoint: live.Edges[0].SourceID,
	}})
	require.NoError(t, err)
	stages := line.StagesValue()
	call(http.MethodDelete, path, nil, http.StatusNoContent)
	call(http.MethodGet, path, nil, http.StatusNotFound)
	listed = call(http.MethodGet, collection+"?factory_id="+factory.ID.String(), nil, http.StatusOK)
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &list))
	require.Empty(t, list.Automations)
	line, err = factory.FindLine(db, line.ID)
	require.NoError(t, err)
	require.Empty(t, line.Steps)
	stages[1].StepIndex = nil
	require.Equal(t, stages, line.StagesValue())

	t.Run("validate a model from the authenticated member's subscription", func(t *testing.T) {
		connection, err := models.CreateCodexConnection(db, orgID, resources.User, time.Now())
		require.NoError(t, err)
		const modelID = "gpt-5.4"
		require.NoError(t, db.Model(connection).Updates(map[string]any{
			"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("test-bundle"),
			"available_models": datatypes.JSONSlice[models.CodexModel]{{Model: modelID}},
		}).Error)
		subscription := true
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
		request := &automationWriteRequest{FactoryID: factory.ID.String(), Stage: "backlog", YAML: `apiVersion: v2
kind: Automation
name: Review task
agent: {model: gpt-5.4}
triggers: [{use: manual}]
steps: [{name: Review, prompt: Review the changes.}]
`}
		body, err := json.Marshal(request)
		require.NoError(t, err)
		validated := execRequest(server, requestParams{
			method: http.MethodPost, path: collection + ":validate", body: body, contentType: "application/json",
			authCookie: token, headers: map[string]string{"X-Organization-Id": orgID.String()},
		})
		require.Equal(t, http.StatusOK, validated.Code, validated.Body.String())
		var result struct {
			Definition struct {
				Agent map[string]string `json:"agent"`
			} `json:"definition"`
		}
		require.NoError(t, json.Unmarshal(validated.Body.Bytes(), &result))
		require.Equal(t, map[string]string{"model": modelID}, result.Definition.Agent)
	})
}

func automationAPITestServer(t *testing.T, resources *support.ResourceRegistry) (*Server, string) {
	t.Helper()
	signer := jwt.NewSigner("test-automation-api")
	oidcProvider := support.NewOIDCProvider()
	server, err := NewServer(resources.Encryptor, resources.Registry, signer, oidcProvider,
		"", "http://localhost", "http://localhost", "test", "/app/templates", resources.AuthService, false)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, resources.AuthService, resources.Registry, resources.Encryptor, oidcProvider)
	server.RegisterWebRoutes("")
	token, err := authentication.GenerateAccountToken(signer, resources.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	return server, token
}
