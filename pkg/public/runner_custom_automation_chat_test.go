package public

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerCustomAutomationChatTransportDoesNotApplyPlanning(t *testing.T) {
	r := support.Setup(t)
	server, session, factory, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	line := mustAutoStartLine(t, r, factory, true)
	mustEnableAutoStart(t, db, factory, line.ID)
	order, err := factory.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	// Existing planning results must not make a custom conversation start a task.
	require.NoError(t, session.ProposeUpdate(db, models.PlanningSessionUpdate{
		Scores: &models.PlanningReviewScores{
			Clarity:       models.PlanningScoreValue{Score: 3, Summary: "The task is clear."},
			Complexity:    models.PlanningScoreValue{Score: 3, Summary: "The change is small."},
			Verifiability: models.PlanningScoreValue{Score: 3, Summary: "The outcome is testable."},
		},
		Spec: "# Existing plan\n\nKeep the existing behavior.\n",
	}))
	ready, err := models.WorkOrderReadyForAutoStart(db, factory, order, session)
	require.NoError(t, err)
	require.True(t, ready, "the fixture must otherwise qualify for planning auto-start")
	require.NoError(t, db.Model(session).Update("kind", models.PlanningSessionKindCustomAutomation).Error)
	session.Kind = models.PlanningSessionKindCustomAutomation

	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/api/v1/runner/planning-sessions/"+path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		return rec
	}
	activityID := uuid.New()
	activity := request(http.MethodPut, "activities/"+activityID.String(), fmt.Sprintf(`{
		"schema_version":2,"activity_id":%q,"provider":"codex","turn":1,
		"sequence":1,"status":"passed","started_at":%d,
		"items":[{"type":"tool","id":"read-1","kind":"bash","input":"git status","status":"passed"}]
	}`, activityID.String(), time.Now().Add(-time.Second).UnixMilli()))
	require.Equal(t, http.StatusOK, activity.Code, activity.Body.String())
	message := request(http.MethodPost, "agent-messages", fmt.Sprintf(`{"text":"The check is complete.","activity_id":%q}`, activityID.String()))
	require.Equal(t, http.StatusOK, message.Code, message.Body.String())
	recorded, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, recorded)
	require.NotNil(t, recorded[len(recorded)-1].ActivityID)
	require.Equal(t, activityID, *recorded[len(recorded)-1].ActivityID)

	waiting := request(http.MethodGet, "wait?hold_seconds=1", "")
	require.Equal(t, http.StatusOK, waiting.Code, waiting.Body.String())
	unchanged, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	require.Equal(t, models.FactoryWorkOrderStateDraft, unchanged.State)
	require.NoError(t, session.SendUserMessage(db, "Explain that check.", uuid.Nil))
	reply := request(http.MethodGet, "wait?hold_seconds=1", "")
	require.Equal(t, http.StatusOK, reply.Code, reply.Body.String())
	var delivered map[string]any
	require.NoError(t, json.Unmarshal(reply.Body.Bytes(), &delivered))
	require.Equal(t, models.PlanningWaitKindMessage, delivered["status"])
	require.Equal(t, "Explain that check.", delivered["text"])
	require.Contains(t, delivered["continuation"], "The check is complete.")
	require.Contains(t, delivered["continuation"], "Explain that check.")
	require.NotContains(t, delivered["continuation"], "Existing plan")

	for _, mutation := range []struct{ path, body string }{
		{"updates", `{"scores":{"clarity":{"score":1,"summary":"Unclear"},"complexity":{"score":1,"summary":"Large"},"verifiability":{"score":1,"summary":"Unknown"}},"spec":"# Replacement"}`},
		{"specs", `{"body":"# Replacement"}`},
	} {
		rejected := request(http.MethodPost, mutation.path, mutation.body)
		require.Equal(t, http.StatusBadRequest, rejected.Code, rejected.Body.String())
	}
	checks, err := unchanged.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 3)
	for _, check := range checks {
		require.Equal(t, float64(3), check.Score)
	}
	artifacts, err := unchanged.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Contains(t, string(artifacts[0].Data), "Existing plan")
}

func TestRunnerCustomAutomationSurvey(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	db := database.DB(t.Context())
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	ordinaryToken := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60))
	canvasID := uuid.MustParse(ctx.WorkflowID)
	session := &models.FactoryPlanningSession{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID,
		Kind: models.PlanningSessionKindCustomAutomation, State: models.PlanningSessionStateRunning,
		CanvasID: &canvasID, CanvasRunID: &ctx.RunID, OriginRunID: &ctx.RunID,
		DraftWorkOrderID: &order.ID, HeartbeatAt: time.Now(),
	}
	require.NoError(t, db.Create(session).Error)
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60))
	mcp := startGlobalMCP(t, server, token, runner.GlobalMCPFiles())
	require.Contains(t, mcp.toolNames(), "survey")
	require.NotContains(t, mcp.toolNames(), "propose_plan")
	result := mcp.call("survey", map[string]any{"questions": []map[string]any{{"prompt": "Which approach?", "options": []string{"Small patch", "Refactor"}, "recommended_option": "Small patch"}}})
	require.Equal(t, "shown", result["status"])
	stored, err := models.FindPlanningSession(db, r.Organization.ID, factory.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, stored.CurrentSurvey().Questions, 1)
	require.Equal(t, "Small patch", stored.CurrentSurvey().Questions[0].RecommendedOption)
	require.NoError(t, stored.SendUserMessage(db, "Which approach?\nSmall patch", uuid.Nil))
	require.Empty(t, stored.CurrentSurvey().Questions)
	require.NoError(t, stored.BeginWait(db))
	wait, err := stored.ConsumeWait(db)
	require.NoError(t, err)
	require.Contains(t, wait.Text, "Small patch")
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Empty(t, checks)
	denied := runnerRequest(t, server, http.MethodPost, ordinaryToken, "/api/v1/runner/task/surveys", `{"questions":[{"prompt":"Question","options":["Answer"]}]}`)
	require.Equal(t, http.StatusForbidden, denied.Code)
}
