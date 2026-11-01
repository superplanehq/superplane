package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/components/runner/openrouter"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// The timeout applies only to the execution transaction. MCP refresh keeps its
// own transaction and advisory lock, with the normal database timeout.
func shortenExecutionIdleTimeout(t *testing.T) {
	t.Helper()
	db := database.DB(t.Context())
	const callback = "test:execution_idle_timeout"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		_, locksRows := tx.Statement.Clauses["FOR"]
		if tx.Error == nil && tx.Statement.Table == "workflow_node_executions" &&
			locksRows {
			tx.AddError(tx.Session(&gorm.Session{NewDB: true}).Exec("SET LOCAL idle_in_transaction_session_timeout = '500ms'").Error)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
}

func TestNodeExecutorOpenRouterSlowPreparation(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			fixture := setupOpenRouterExecution(t, func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				select {
				case <-time.After(1500 * time.Millisecond):
				case <-req.Context().Done():
					return
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2","expires_in":7200,"token_type":"Bearer"}`))
			})
			shortenExecutionIdleTimeout(t)

			err := newTestNodeExecutor(t, fixture.resources).LockAndProcessNodeExecution(fixture.execution.ID)
			require.NoError(t, err)
			require.EqualValues(t, 1, calls.Load(), "preparation must not repeat inside the final transaction")
			payload := fixture.requireTaskLifecycle(t)
			var mcpConfig string
			for _, file := range payload.Files {
				if file.Path == runner.WorkspaceMCPConfigPath {
					mcpConfig = file.Content
				}
			}
			if status == http.StatusOK {
				require.Contains(t, mcpConfig, "Bearer fresh-access")
			} else {
				require.Empty(t, mcpConfig, "an optional MCP failure must skip the resource")
			}
		})
	}
}

func TestNodeExecutorOpenRouterConcurrentPreparation(t *testing.T) {
	entered, release, handler := blockedOAuthRefresh(t)
	fixture := setupOpenRouterExecution(t, handler)
	results := make(chan error, 2)
	executor := newTestNodeExecutor(t, fixture.resources)
	go func() { results <- executor.LockAndProcessNodeExecution(fixture.execution.ID) }()
	awaitPreparation(t, entered)
	fixture.requireNoTaskLifecycle(t)
	saved, err := models.FindNodeExecutionInTransaction(database.DB(t.Context()), fixture.execution.WorkflowID, fixture.execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionStatePending, saved.State)
	awaitPreparation(t, fixture.preparing)
	go func() { results <- executor.LockAndProcessNodeExecution(fixture.execution.ID) }()
	awaitPreparation(t, fixture.preparing)
	release()
	succeeded, locked := countConcurrentExecutionResults(t, []error{awaitExecutionResult(t, results), awaitExecutionResult(t, results)})
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, locked)
	fixture.requireTaskLifecycle(t)
}

func TestNodeExecutorOpenRouterDiscardsStalePreparation(t *testing.T) {
	for _, change := range []struct {
		name     string
		apply    func(*testing.T, openRouterExecutionFixture)
		expected error
	}{
		{name: "cancellation", apply: func(t *testing.T, f openRouterExecutionFixture) {
			require.NoError(t, f.execution.RequestCancellation(database.DB(t.Context()), &f.resources.User))
		}, expected: ErrRecordLocked},
		{name: "finished execution", apply: func(t *testing.T, f openRouterExecutionFixture) {
			_, err := f.execution.PassInTransaction(database.DB(t.Context()), nil)
			require.NoError(t, err)
		}, expected: ErrRecordLocked},
		{name: "deleted canvas", apply: func(t *testing.T, f openRouterExecutionFixture) {
			require.NoError(t, database.DB(t.Context()).Model(&models.Canvas{}).
				Where("id = ?", f.execution.WorkflowID).Update("deleted_at", time.Now()).Error)
		}, expected: ErrRecordLocked},
		{name: "configuration change", apply: func(t *testing.T, f openRouterExecutionFixture) {
			configuration := f.execution.Configuration.Data()
			configuration["model"] = "openai/gpt-5"
			require.NoError(t, database.DB(t.Context()).Model(f.execution).
				Update("configuration", datatypes.NewJSONType(configuration)).Error)
		}, expected: errExecutionPreparationChanged},
		{name: "component change", apply: func(t *testing.T, f openRouterExecutionFixture) {
			node, err := models.FindCanvasNode(database.DB(t.Context()), f.execution.WorkflowID, f.execution.NodeID)
			require.NoError(t, err)
			require.NoError(t, database.DB(t.Context()).Model(node).
				Update("ref", datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})).Error)
		}, expected: errExecutionPreparationChanged},
	} {
		t.Run(change.name, func(t *testing.T) {
			entered, release, handler := blockedOAuthRefresh(t)
			fixture := setupOpenRouterExecution(t, handler)
			results := make(chan error, 1)
			executor := newTestNodeExecutor(t, fixture.resources)
			go func() { results <- executor.LockAndProcessNodeExecution(fixture.execution.ID) }()
			awaitPreparation(t, entered)
			change.apply(t, fixture)
			release()
			require.ErrorIs(t, awaitExecutionResult(t, results), change.expected)
			fixture.requireNoTaskLifecycle(t)
		})
	}
}

func TestNodeExecutorOpenRouterRechecksBYOKAfterPreparation(t *testing.T) {
	entered, release, handler := blockedOAuthRefresh(t)
	fixture := setupOpenRouterExecution(t, handler)
	results := make(chan error, 1)
	executor := newTestNodeExecutor(t, fixture.resources)
	go func() { results <- executor.LockAndProcessNodeExecution(fixture.execution.ID) }()
	awaitPreparation(t, entered)
	require.NoError(t, models.DisableExperimentalFeature(fixture.resources.Organization.ID, features.FeatureOrganizationBYOK))
	release()
	require.NoError(t, awaitExecutionResult(t, results))
	execution, err := models.FindNodeExecutionInTransaction(database.DB(t.Context()), fixture.execution.WorkflowID, fixture.execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionResultFailed, execution.Result)
	require.Contains(t, execution.ResultMessage, "does not have access")
	fixture.requireNoTaskLifecycle(t)
}

func TestNodeExecutorOpenRouterRollsBackFailedDispatch(t *testing.T) {
	fixture := setupOpenRouterExecution(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2","expires_in":7200}`))
	})
	db := database.DB(t.Context())
	failure := errors.New("poll request persistence failed")
	const callback = "test:fail_openrouter_poll"
	var pollAttempted bool
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "workflow_node_requests" {
			pollAttempted = true
			tx.AddError(failure)
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Create().Remove(callback) })
	executor := newTestNodeExecutor(t, fixture.resources)
	require.ErrorIs(t, executor.LockAndProcessNodeExecution(fixture.execution.ID), failure)
	require.True(t, pollAttempted, "fail after task, KV, and metadata persistence")
	fixture.requireNoTaskLifecycle(t)
	execution, err := models.FindNodeExecutionInTransaction(db, fixture.execution.WorkflowID, fixture.execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionStatePending, execution.State)
	require.NoError(t, db.Callback().Create().Remove(callback))
	require.NoError(t, executor.LockAndProcessNodeExecution(fixture.execution.ID))
	fixture.requireTaskLifecycle(t)
}

func TestNodeExecutorOpenRouterPersistsAnalysisMarkerAtDispatch(t *testing.T) {
	entered, release, handler := blockedOAuthRefresh(t)
	fixture := setupOpenRouterExecution(t, handler)
	db := database.DB(t.Context())
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, fixture.execution.WorkflowID)
	require.NoError(t, err)
	now := time.Now()
	session := &models.FactoryPlanningSession{
		ID: uuid.New(), OrganizationID: fixture.resources.Organization.ID, FactoryID: *canvas.FactoryID,
		CreatedByUserID: &fixture.resources.User, Repository: "acme/payments",
		Kind: models.PlanningSessionKindWorkOrderAnalysis, State: models.PlanningSessionStateRunning,
		CanvasID: &canvas.ID, CanvasRunID: &fixture.execution.RunID,
		HeartbeatAt: now, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, db.Create(session).Error)
	results := make(chan error, 1)
	executor := newTestNodeExecutor(t, fixture.resources)
	go func() { results <- executor.LockAndProcessNodeExecution(fixture.execution.ID) }()
	awaitPreparation(t, entered)
	_, err = models.FindLatestNodeExecutionKVValueInTransaction(db, fixture.execution.ID, "analysis_session")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound, "preparation must not write execution state")
	release()
	require.NoError(t, awaitExecutionResult(t, results))
	fixture.requireTaskLifecycle(t)
	marker, err := models.FindLatestNodeExecutionKVValueInTransaction(db, fixture.execution.ID, "analysis_session")
	require.NoError(t, err)
	require.Equal(t, "true", marker)
}

func blockedOAuthRefresh(t *testing.T) (<-chan struct{}, func(), http.HandlerFunc) {
	t.Helper()
	entered := make(chan struct{}, 1)
	resume := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	t.Cleanup(release)
	return entered, release, func(w http.ResponseWriter, req *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-resume:
		case <-req.Context().Done():
			return
		case <-time.After(5 * time.Second):
			http.Error(w, "preparation did not resume", http.StatusGatewayTimeout)
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2","expires_in":7200}`))
	}
}

func awaitPreparation(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("OpenRouter did not reach OAuth preparation")
	}
}

func awaitExecutionResult(t *testing.T, results <-chan error) error {
	t.Helper()
	select {
	case err := <-results:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("execution worker did not finish")
		return nil
	}
}

type openRouterExecutionFixture struct {
	resources *support.ResourceRegistry
	execution *models.CanvasNodeExecution
	preparing <-chan struct{}
}

func setupOpenRouterExecution(t *testing.T, handler http.HandlerFunc) openRouterExecutionFixture {
	t.Helper()
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	t.Setenv("NO_ENCRYPTION", "yes")
	crypto.SetProcessEncryptor(r.Encryptor)
	t.Cleanup(func() { crypto.SetProcessEncryptor(nil) })
	db := database.DB(t.Context())
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	preparing := make(chan struct{}, 2)
	r.Registry.Actions[openrouter.ComponentName] = registry.NewPanicableAction(&openRouterHTTPAction{
		RunOpenRouter: &openrouter.RunOpenRouter{},
		http:          rewriteOAuthHTTP{server: server},
		preparing:     preparing,
	})
	settings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.Background()).Model(settings).Updates(map[string]any{
			"default_hosted_provider": settings.DefaultHostedProvider,
			"default_hosted_model":    settings.DefaultHostedModel,
		}).Error)
	})
	require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
		"default_hosted_provider": models.UsageProviderAnthropic, "default_hosted_model": "hosted-model",
	}).Error)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "sentry", true, models.FactoryAgentResourceConfig{
		Transport: "http", URL: "https://mcp.example.com/mcp",
		Auth: models.FactoryAgentResourceAuthOAuth, ToolsDefaultApplied: true,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		TokenEndpoint: "https://mcp.example.com/token", ClientID: "client-1",
		AccessTokenExpiresAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
	}))
	require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
	for name, value := range map[string]string{
		models.FactoryAgentResourceSecretAccessToken:  "expired-access",
		models.FactoryAgentResourceSecretRefreshToken: "refresh-1",
	} {
		encrypted, err := mcp.EncryptResourceSecret(t.Context(), r.Encryptor, resource.ID, value)
		require.NoError(t, err)
		require.NoError(t, resource.UpsertSecret(db, name, encrypted))
	}
	fleet := models.RunnerFleet{
		ID: uuid.New(), Slug: "openrouter-test", ScopeType: models.RunnerFleetScopeInstallation,
		Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion,
	}
	require.NoError(t, fleet.Create(db))
	secret, err := support.CreateSecret(t, r, map[string]string{"key": "openrouter-key"})
	require.NoError(t, err)
	config := map[string]any{
		"machineType": fleet.Slug, "model": "anthropic/claude-sonnet-4-6",
		"credentials": map[string]any{"source": "secret", "secret": map[string]any{"secret": secret.Name, "key": "key"}},
		"steps":       []any{map[string]any{"name": "Prompt", "type": "prompt", "prompt": "Fix the tests"}},
	}
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "agent", Type: models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerOpenRouter"}}),
		Configuration: datatypes.NewJSONType(config),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "agent", "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "agent", event.ID, event.ID, config)
	return openRouterExecutionFixture{resources: r, execution: execution, preparing: preparing}
}

func (f openRouterExecutionFixture) requireTaskLifecycle(t *testing.T) runnerapi.TaskPayload {
	t.Helper()
	db := database.DB(t.Context())
	var tasks []models.RunnerTask
	require.NoError(t, db.Find(&tasks).Error)
	require.Len(t, tasks, 1)
	execution, err := models.FindNodeExecutionInTransaction(db, f.execution.WorkflowID, f.execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionStateStarted, execution.State)
	taskID, err := models.FindLatestNodeExecutionKVValueInTransaction(db, execution.ID, "task_id")
	require.NoError(t, err)
	require.Equal(t, tasks[0].ID.String(), taskID)
	metadata := execution.Metadata.Data()
	require.Equal(t, taskID, metadata[runner.ExecutionMetadataBrokerTaskID])
	require.Equal(t, "integrated", metadata[runner.ExecutionMetadataTaskBackend])
	var requests []models.CanvasNodeRequest
	require.NoError(t, db.Where("execution_id = ?", execution.ID).Find(&requests).Error)
	require.Len(t, requests, 1)
	require.Equal(t, runner.HookPoll, requests[0].Spec.Data().InvokeAction.ActionName)
	require.Equal(t, taskID, requests[0].Spec.Data().InvokeAction.Parameters["task_id"])
	var payload runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(tasks[0].PayloadCiphertext, &payload))
	require.Equal(t, taskID, payload.ID)
	return payload
}

func (f openRouterExecutionFixture) requireNoTaskLifecycle(t *testing.T) {
	t.Helper()
	db := database.DB(t.Context())
	var tasks int64
	require.NoError(t, db.Model(&models.RunnerTask{}).Count(&tasks).Error)
	require.Zero(t, tasks)
	var requests int64
	require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("execution_id = ?", f.execution.ID).Count(&requests).Error)
	require.Zero(t, requests)
	_, err := models.FindLatestNodeExecutionKVValueInTransaction(db, f.execution.ID, "task_id")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	execution, err := models.FindNodeExecutionInTransaction(db, f.execution.WorkflowID, f.execution.ID)
	require.NoError(t, err)
	require.Empty(t, execution.Metadata.Data()[runner.ExecutionMetadataBrokerTaskID])
}

// Keep the real OpenRouter action and OAuth protocol; only route HTTP to the
// controlled local endpoint because MCP token URLs require public HTTPS.
type openRouterHTTPAction struct {
	*openrouter.RunOpenRouter
	http      core.HTTPContext
	preparing chan<- struct{}
}

func (a *openRouterHTTPAction) Execute(ctx core.ExecutionContext) error {
	ctx.HTTP = a.http
	return a.RunOpenRouter.Execute(ctx)
}

func (a *openRouterHTTPAction) Prepare(ctx core.ExecutionContext) (core.PreparedExecution, error) {
	select {
	case a.preparing <- struct{}{}:
	default:
	}
	ctx.HTTP = a.http
	return a.RunOpenRouter.Prepare(ctx)
}

type rewriteOAuthHTTP struct{ server *httptest.Server }

func (h rewriteOAuthHTTP) Do(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(h.server.URL)
	if err != nil {
		return nil, err
	}
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = target.Scheme
	cloned.URL.Host = target.Host
	client := h.server.Client()
	client.Timeout = 5 * time.Second
	return client.Do(cloned)
}
