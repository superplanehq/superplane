package public

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerListsTasksByState(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, factory, draft := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	require.NotEmpty(t, token)

	db := database.DB(t.Context())
	openOrder, err := factory.CreateWorkOrder(db, "Open task", "Stop the retry loop.", nil, nil, nil)
	require.NoError(t, err)
	_, err = openOrder.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/tasks?state=open", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	listed := decodeRunnerTaskList(t, rec.Body.Bytes())
	require.Len(t, listed, 1)
	assert.Equal(t, openOrder.ID.String(), listed[0].ID)
	assert.Equal(t, "open", listed[0].State)
	assert.Equal(t, factory.WorkOrderKey(openOrder.Number), listed[0].Key)
	assert.Contains(t, listed[0].URL, "/task/")

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/tasks?state=draft", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	listed = decodeRunnerTaskList(t, rec.Body.Bytes())
	require.Len(t, listed, 1)
	assert.Equal(t, draft.ID.String(), listed[0].ID)

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/tasks?state=later", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/tasks?state=open&state=open&state=draft&state=closed", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/tasks?limit=100000", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "Stop the retry loop.")
}

func TestRunnerCreatesADraftTaskWithoutAnOwner(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, factory, _ := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/tasks", `{"title":" Ship the fix ","description":"Stop the retry loop."}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	created := decodeRunnerTask(t, rec.Body.Bytes())
	assert.Equal(t, "Ship the fix", created.Title)
	assert.Equal(t, "draft", created.State)
	assert.NotEmpty(t, created.Key)
	assert.Contains(t, created.URL, "/task/")

	db := database.DB(t.Context())
	order, err := factory.FindWorkOrder(db, uuid.MustParse(created.ID))
	require.NoError(t, err)
	assert.Nil(t, order.CreatedByID)
	assignees, err := order.ListAssignees(db)
	require.NoError(t, err)
	assert.Empty(t, assignees)

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/tasks", `{"title":" "}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/tasks", `{"title":"`+strings.Repeat("a", 257)+`"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/tasks", `{"title":"ok","description":"`+strings.Repeat("b", 128*1024)+`"}`)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
}

func TestRunnerClosesThisTaskOnce(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	db := database.DB(t.Context())
	_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/close", `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	closed := decodeRunnerTask(t, rec.Body.Bytes())
	assert.Equal(t, order.ID.String(), closed.ID)
	assert.Equal(t, "closed", closed.State)
	assert.Equal(t, "completed", closed.Result)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
	assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
	run, err := models.FindUnscopedCanvasRun(db, ctx.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, run.State)

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/close", `{}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}

func TestRunnerRemovesOneArtifactByKeyOrTitle(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, _, order := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	db := database.DB(t.Context())
	mustMarkdownArtifact(t, order, "plan", "PLAN.md")
	mustMarkdownArtifact(t, order, "", "NOTES.md")

	rec := runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/artifacts", `{"key":"plan"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	remaining, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, "NOTES.md", artifactTitle(remaining[0].Data))

	rec = runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/artifacts", `{"title":"NOTES.md"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	remaining, err = order.ListArtifacts(db)
	require.NoError(t, err)
	assert.Empty(t, remaining)

	rec = runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/artifacts", `{"title":"NOTES.md"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/artifacts", `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestPlanningRunnerEditsTheSessionDraft(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	server, session, factory, planningToken := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NotNil(t, session.CanvasID)
	require.NotNil(t, session.DraftWorkOrderID)
	run, err := models.FindUnscopedCanvasRun(db, *session.CanvasRunID)
	require.NoError(t, err)
	event := support.EmitCanvasEventForNode(t, *session.CanvasID, "start", "default", nil)
	execution := createExecutionForCanvasRun(t, run, event.ID, "start")
	ctx := core.ExecutionContext{
		ID: execution.ID, RunID: run.ID, WorkflowID: session.CanvasID.String(),
		OrganizationID: r.Organization.ID.String(), NodeID: "start", BaseURL: "http://app.test",
	}
	env := runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplanePlanningID, Value: session.ID.String()},
		{Name: runner.EnvSuperplaneRunToken, Value: planningToken},
	}, 60, nil)
	token := runnerTokenFrom(env)
	require.NotEmpty(t, token)

	draft, err := factory.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	other, err := factory.CreateWorkOrder(db, "Other task", "", nil, nil, nil)
	require.NoError(t, err)
	mustMarkdownArtifact(t, draft, "", "DRAFT.md")
	mustMarkdownArtifact(t, other, "", "OTHER.md")

	rec := runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/artifacts", `{"title":"DRAFT.md"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	draftArtifacts, err := draft.ListArtifacts(db)
	require.NoError(t, err)
	assert.Empty(t, draftArtifacts)
	otherArtifacts, err := other.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, otherArtifacts, 1)

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/close", `{"result":"rejected"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	closed := decodeRunnerTask(t, rec.Body.Bytes())
	assert.Equal(t, draft.ID.String(), closed.ID)
	assert.Equal(t, "rejected", closed.Result)
	reloaded, err := factory.FindWorkOrder(db, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
	assert.Equal(t, models.FactoryWorkOrderResultManualRejected, reloaded.Result)
	untouched, err := factory.FindWorkOrder(db, other.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, untouched.State)
	reloadedSession, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, reloadedSession.State)
	planningRun, err := models.FindUnscopedCanvasRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, planningRun.State)
}

func mustMarkdownArtifact(t *testing.T, order *models.FactoryWorkOrder, key, title string) {
	t.Helper()
	_, err := order.CreateArtifact(database.DB(t.Context()), models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeMarkdown,
		Key:  key,
		Data: map[string]any{"title": title, "body": title},
	})
	require.NoError(t, err)
}

type runnerTaskJSON struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Result string `json:"result"`
	URL    string `json:"url"`
}

func decodeRunnerTask(t *testing.T, raw []byte) runnerTaskJSON {
	t.Helper()
	var task runnerTaskJSON
	require.NoError(t, json.Unmarshal(raw, &task))
	return task
}

func decodeRunnerTaskList(t *testing.T, raw []byte) []runnerTaskJSON {
	t.Helper()
	var body struct {
		Tasks []runnerTaskJSON `json:"tasks"`
	}
	require.NoError(t, json.Unmarshal(raw, &body))
	return body.Tasks
}
