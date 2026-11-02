package public

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/components/runner/claude"
	"github.com/superplanehq/superplane/pkg/components/runner/codex"
	"github.com/superplanehq/superplane/pkg/components/runner/openrouter"
	"github.com/superplanehq/superplane/pkg/components/runner/superplane"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/gorm"
)

type mergeRunnerStartup struct {
	name  string
	token string
	files []runner.TaskFile
}

func TestRunnerGlobalPlanUpdate(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	server, session, factoryModel, planningToken := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	token := mustPlanningGlobalRunnerToken(t, r, session, planningToken)

	assert.ElementsMatch(t, globalRunnerToolsWith("propose_plan", "propose_update", "inspect_attachment"), listRunnerTools(t, server, token))

	rec := runnerRequest(t, server, http.MethodPut, token, "/api/v1/runner/task/plan", `{
		"scores": {
			"clarity": {"score": 3, "summary": "Outcome, scope, and done are defined."},
			"complexity": {"score": 2, "summary": "One agent can finish this in one run."},
			"verifiability": {"score": 3, "summary": "Existing tests cover the change."}
		},
		"spec": "# Retry refunds\n\nStop double charges.\n"
	}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"status":"shown"}`, rec.Body.String())

	require.NotNil(t, session.DraftWorkOrderID)
	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	assert.Len(t, checks, 3)
	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.NotEmpty(t, artifacts)
	assert.Contains(t, string(artifacts[0].Data), "Stop double charges.")

	ctx, _, _ := mustTaskLinkedAgentRun(t, r)
	taskToken := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	rec = runnerRequest(t, server, http.MethodPut, taskToken, "/api/v1/runner/task/plan", `{"spec":"# Plan"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPut, planningToken, "/api/v1/runner/task/plan", `{"spec":"# Plan"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
}

func TestRunnerGlobalMergeCheck(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	signer := jwt.NewSigner("test")
	server := newRunnerArtifactTestServer(t, r, signer)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	const nodeID = "assess-risk"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: nodeID, Name: "Assess risk", Type: models.NodeTypeComponent}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	execution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
	ctx := core.ExecutionContext{ID: execution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(), OrganizationID: r.Organization.ID.String(), NodeID: nodeID, BaseURL: "http://app.test"}
	var token string
	var startups []mergeRunnerStartup
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		// NodeExecutor starts the execution in its transaction before Execute.
		// Runner setup uses a separate connection, which still sees pending.
		require.NoError(t, execution.StartInTransaction(tx))
		visible, err := models.FindNodeExecutionInTransaction(db, canvas.ID, execution.ID)
		require.NoError(t, err)
		require.Equal(t, models.CanvasNodeExecutionStatePending, visible.State)
		for _, provider := range []struct {
			name      string
			component interface {
				Execute(core.ExecutionContext) error
			}
			model string
		}{
			{"Claude Code", &claude.RunClaudeCode{}, "sonnet"},
			{"Codex", &codex.RunCodex{}, "gpt-5"},
			{"OpenRouter", &openrouter.RunOpenRouter{}, "anthropic/claude-sonnet-4-6"},
			{"SuperPlane", &superplane.RunSuperPlane{}, "anthropic::claude-sonnet-4-6"},
		} {
			t.Run(provider.name, func(t *testing.T) {
				tasks := &testcontexts.RunnerTaskContext{}
				providerCtx := ctx
				providerCtx.RunnerTasks = tasks
				providerCtx.Configuration = map[string]any{
					"machineType": runner.MachineTypeE1LargeAMD64,
					"model":       provider.model,
					"credentials": map[string]any{"source": "secret", "secret": map[string]any{"secret": "provider", "key": "api_key"}},
					"environment": []map[string]any{{"name": runner.EnvSuperplaneMergeConfidenceOrderID, "value": order.ID.String(), "valueSource": "literal"}},
					"steps":       []map[string]any{{"name": "API latency", "type": "prompt", "prompt": "Merge check: api-latency.\nAssess latency."}},
				}
				if provider.name == "SuperPlane" {
					delete(providerCtx.Configuration.(map[string]any), "credentials")
				}
				providerCtx.Secrets = &testcontexts.SecretsContext{Values: map[string][]byte{"provider/api_key": []byte("test-key")}}
				providerCtx.HTTP = &testcontexts.HTTPContext{}
				providerCtx.Webhook = &testcontexts.NodeWebhookContext{}
				providerCtx.ExecutionState = &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
				providerCtx.Requests = &testcontexts.RequestContext{}
				providerCtx.HostedLLM = &testcontexts.HostedLLMContext{Access: core.HostedLLMAccess{APIKey: "test-key", AllowedModels: []string{"claude-sonnet-4-6"}}}
				require.NoError(t, provider.component.Execute(providerCtx))
				var task struct {
					Environment []runner.TaskEnvironmentVariable `json:"environment"`
					Files       []runner.TaskFile                `json:"files"`
				}
				require.NoError(t, json.Unmarshal(tasks.Payload, &task))
				token = runnerTokenFrom(task.Environment)
				require.NotEmpty(t, token, "runner token must be issued before the start transaction commits")
				scope, err := runner.ParseRunnerToken(signer, token)
				require.NoError(t, err)
				assert.Equal(t, []string{"api-latency"}, scope.MergeChecks)
				assert.Equal(t, map[string]string{"api-latency": "API latency"}, scope.MergeCheckLabels)
				startups = append(startups, mergeRunnerStartup{provider.name, token, task.Files})
				assert.ErrorContains(t, runner.ValidateRunnerScope(db, scope), "inactive")
				rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/merge-checks", `{"check":"api-latency","score":2,"summary":"Before commit."}`)
				assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
			})
		}
		return nil
	}))
	require.NotEmpty(t, token)
	active, err := models.FindNodeExecutionInTransaction(db, canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateStarted, active.State)

	assert.ElementsMatch(t, globalRunnerToolsWith("report_merge_check"), listRunnerTools(t, server, token))

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/merge-checks", `{"check":"risk","score":4,"summary":"Risk is not enabled."}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	for _, startup := range startups {
		t.Run(startup.name+" reporting", func(t *testing.T) {
			reportMergeCheckOverStdio(t, server, startup.token, startup.files)
		})
	}

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "API latency", checks[0].Name)
	assert.Equal(t, float64(2), checks[0].Score)
	assert.Equal(t, "The change adds one query per request.", checks[0].Summary)
	require.NotNil(t, checks[0].RunID)
	assert.Equal(t, run.ID, *checks[0].RunID)
	scope, err := runner.ParseRunnerToken(signer, token)
	require.NoError(t, err)
	mergeScope, ok := scope.MergeConfidenceScope()
	require.True(t, ok)
	mismatchedScope := *scope
	mismatchedScope.CanvasRunID = uuid.New()
	mismatchedRun, err := runner.MintRunnerToken(signer, mismatchedScope, time.Minute)
	require.NoError(t, err)
	rec = runnerRequest(t, server, http.MethodPost, mismatchedRun, "/api/v1/runner/task/merge-checks", `{"check":"api-latency","score":2,"summary":"Wrong run."}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	taskCtx, _, _ := mustTaskLinkedAgentRun(t, r)
	taskToken := runnerTokenFrom(runner.AttachRunnerEnv(taskCtx, nil, 60, nil))
	rec = runnerRequest(t, server, http.MethodPost, taskToken, "/api/v1/runner/task/merge-checks", `{"check":"api-latency","score":2,"summary":"Wrong flow."}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	for _, state := range []string{models.CanvasNodeExecutionStateFinished, models.CanvasNodeExecutionStateCancelling} {
		require.NoError(t, db.Model(execution).Update("state", state).Error)
		assert.False(t, runner.HasRunnerToken(runner.AttachRunnerEnv(ctx, nil, 60, mergeScope)))
		rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/merge-checks", `{"check":"api-latency","score":2,"summary":"Late report."}`)
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func reportMergeCheckOverStdio(t *testing.T, server *Server, token string, files []runner.TaskFile) {
	t.Helper()
	api := httptest.NewServer(server.Router)
	defer api.Close()
	taskDir := t.TempDir()
	for _, file := range files {
		if !strings.HasPrefix(file.Path, "superplane_mcp/") {
			continue
		}
		path := filepath.Join(taskDir, file.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, os.WriteFile(path, []byte(file.Content), 0600))
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", filepath.Join(taskDir, "superplane_mcp/server.js"))
	command.Env = append(os.Environ(), "SUPERPLANE_BASE_URL="+api.URL, "SUPERPLANE_RUNNER_TOKEN="+token, "SUPERPLANE_TASK_DIR="+taskDir)
	stdin, err := command.StdinPipe()
	require.NoError(t, err)
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	require.NoError(t, command.Start())
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	scanner := bufio.NewScanner(stdout)
	request := func(id int, method string, params any) map[string]any {
		require.NoError(t, json.NewEncoder(stdin).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}))
		require.True(t, scanner.Scan(), stderr.String())
		var response map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &response))
		require.Nil(t, response["error"])
		return response["result"].(map[string]any)
	}
	request(1, "initialize", map[string]any{})
	list := request(2, "tools/list", map[string]any{})
	names := []string{}
	for _, tool := range list["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	require.Contains(t, names, "report_merge_check")
	result := request(3, "tools/call", map[string]any{"name": "report_merge_check", "arguments": map[string]any{"check": "api-latency", "score": 2, "summary": "The change adds one query per request."}})
	require.NotEqual(t, true, result["isError"], result)
	assert.Equal(t, "reported", result["structuredContent"].(map[string]any)["status"])
}

func TestRunnerGlobalArtifactUpload(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	ctx, _, order := mustTaskLinkedAgentRun(t, r)

	env := runner.AttachArtifactUploadEnv(ctx, nil, 60, true)
	require.True(t, runner.HasVisualEvidence(env))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, env, 60, nil))
	require.NotEmpty(t, token)
	assert.ElementsMatch(t, globalRunnerToolsWith(
		"inspect_screenshot", "upload_artifact", "report_visual_evidence_unavailable"), listRunnerTools(t, server, token))

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)
	rec := postRunnerArtifactFile(t, server, token, png)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var uploaded runnerArtifactResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &uploaded))
	assert.Equal(t, "Checkout", uploaded.Title)
	artifacts, err := order.ListArtifacts(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.FactoryWorkOrderArtifactTypeFile, artifacts[0].Type)

	links := runnerTokenFrom(runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplaneArtifactToken, Value: "artifact-token"},
		{Name: runner.EnvSuperplanePublishLinks, Value: "yes"},
	}, 60, nil))
	rec = postRunnerArtifactFile(t, server, links, png)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	shared := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	rec = postRunnerArtifactFile(t, server, shared, png)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	withoutArtifacts := runnerTokenWithScopes(t, shared, runner.ScopeArtifactsRead)
	rec = postRunnerArtifactFile(t, server, withoutArtifacts, png)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestRunnerGlobalArtifactLink(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	signer := jwt.NewSigner("test")
	server := newRunnerArtifactTestServer(t, r, signer)
	ctx, _, order := mustTaskLinkedAgentRun(t, r)
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplaneArtifactToken, Value: "artifact-token"},
		{Name: runner.EnvSuperplanePublishLinks, Value: "yes"},
	}, 60, nil))
	require.NotEmpty(t, token)
	assert.ElementsMatch(t, globalRunnerToolsWith("publish_link"), listRunnerTools(t, server, token))

	body := `{"url":"https://staging.example.com/pr-42","title":"Staging","key":"staging"}`
	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/links", body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	artifacts, err := order.ListArtifacts(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.FactoryWorkOrderArtifactTypeLink, artifacts[0].Type)
	require.NotNil(t, artifacts[0].Key)
	assert.Equal(t, order.ID.String()+":staging", *artifacts[0].Key)

	markdown := `{"body":"# Note","title":"Note","key":"note"}`
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/markdown", markdown)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	withoutLinks := runnerTokenWithScopes(t, token, runner.ScopeArtifactsRead)
	rec = runnerRequest(t, server, http.MethodPost, withoutLinks, "/api/v1/runner/task/links", body)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPost, withoutLinks, "/api/v1/runner/task/markdown", markdown)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func mustPlanningGlobalRunnerToken(t *testing.T, r *support.ResourceRegistry, session *models.FactoryPlanningSession, planningToken string) string {
	t.Helper()
	db := database.DB(t.Context())
	require.NotNil(t, session.CanvasID)
	run, err := models.FindUnscopedCanvasRun(db, *session.CanvasRunID)
	require.NoError(t, err)
	event := support.EmitCanvasEventForNode(t, *session.CanvasID, "start", "default", nil)
	execution := createExecutionForCanvasRun(t, run, event.ID, "start")
	ctx := core.ExecutionContext{
		ID: execution.ID, RunID: run.ID, WorkflowID: session.CanvasID.String(),
		OrganizationID: r.Organization.ID.String(), NodeID: "start", BaseURL: "http://app.test",
	}
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplanePlanningID, Value: session.ID.String()},
		{Name: runner.EnvSuperplaneRunToken, Value: planningToken},
	}, 60, nil))
	require.NotEmpty(t, token)
	return token
}

func postRunnerArtifactFile(t *testing.T, server *Server, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/task/artifacts", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "image/png")
	request.Header.Set("Content-Disposition", `attachment; filename="checkout.png"`)
	request.Header.Set("X-SuperPlane-Artifact-Title", "Checkout")
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}
