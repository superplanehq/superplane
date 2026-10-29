package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

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

func TestRunnerToolsFollowTheTokenScopes(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, _, _ := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60))
	require.NotEmpty(t, token)

	assert.Equal(t, []string{"suggest_owners", "assign_owner"}, listRunnerTools(t, server, token))

	scope, err := runner.ParseRunnerToken(jwt.NewSigner("test"), token)
	require.NoError(t, err)
	scope.Scopes = []runner.RunnerToolScope{runner.ScopeOwnerRead}
	readOnly, err := runner.MintRunnerToken(jwt.NewSigner("test"), *scope, time.Minute)
	require.NoError(t, err)

	assert.Equal(t, []string{"suggest_owners"}, listRunnerTools(t, server, readOnly))
	rec := suggestRunnerOwners(t, server, readOnly, `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = assignRunnerOwner(t, server, readOnly, `{"user":"`+r.User.String()+`","reason":"Owns pkg."}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodGet, "not-a-token", "/api/v1/runner/tools", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// Runs that started before the global runner MCP server still hold the
// legacy owner token. They must finish on the legacy routes.
func TestLegacyOwnerTokenKeepsTheLegacyRoutes(t *testing.T) {
	r := support.Setup(t)
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	canvasID := uuid.MustParse(ctx.WorkflowID)
	token, err := runner.MintTaskOwnerToken(jwt.NewSigner("test"), runner.TaskOwnerScope{
		OrganizationID: r.Organization.ID, FactoryID: factory.ID, WorkOrderID: order.ID,
		CanvasID: canvasID, CanvasRunID: ctx.RunID, NodeExecutionID: ctx.ID, NodeID: ctx.NodeID,
	}, time.Minute)
	require.NoError(t, err)

	rec := postPlanningRunner(t, server, token, "/api/v1/runner/tasks/suggest-owners", `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = suggestRunnerOwners(t, server, token, `{}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPlanningRunnerTokenOwnsTheDraftTask(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	server, session, _, planningToken := mustPlanningRunnerSession(t, r)
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
	}, 60)
	token := runnerTokenFrom(env)
	require.NotEmpty(t, token)
	scope, err := runner.ParseRunnerToken(jwt.NewSigner("test"), token)
	require.NoError(t, err)
	assert.Equal(t, runner.RunnerFlowPlanning, scope.Flow)
	assert.Equal(t, session.ID, scope.PlanningSessionID)

	owner := mustTaskOwnerMember(t, r, "planning-dev")
	rec := suggestRunnerOwners(t, server, token, `{"paths":["pkg/payments"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assertAssignedOwner(t, assignRunnerOwner(t, server, token, `{"user":"planning-dev","reason":"Owns pkg/payments."}`), owner.ID)
	factory, err := models.FindFactory(db, session.OrganizationID, session.FactoryID)
	require.NoError(t, err)
	draft, err := factory.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	assert.Contains(t, draft.AssigneeIDs(), owner.ID)

	require.NoError(t, db.Model(session).Update("state", models.PlanningSessionStateEnded).Error)
	rec = assignRunnerOwner(t, server, token, `{"user":"planning-dev","reason":"Owns pkg/payments."}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.False(t, runner.HasRunnerToken(runner.AttachRunnerEnv(ctx, env, 60)))
}

func listRunnerTools(t *testing.T, server *Server, token string) []string {
	t.Helper()
	rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/tools", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Tools []string `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Tools
}
