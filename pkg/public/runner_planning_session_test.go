package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestRunnerPlanningSessionDraft(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	session, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)

	token, err := runneraction.MintPlanningSessionToken(signer, runneraction.PlanningSessionScope{
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		SessionID:      session.ID,
		CanvasRunID:    *session.CanvasRunID,
	}, time.Hour)
	require.NoError(t, err)

	draft := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/drafts", bytes.NewReader([]byte(
		`{"title":"Retry refunds","description":"Stop double charges."}`,
	)))
	draft.Header.Set("Authorization", "Bearer "+token)
	draftRec := httptest.NewRecorder()
	server.Router.ServeHTTP(draftRec, draft)
	require.Equal(t, http.StatusOK, draftRec.Code, draftRec.Body.String())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Retry refunds", reloaded.Draft().Title)
}

func TestRunnerPlanningSessionSurvey(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	session, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)

	token, err := runneraction.MintPlanningSessionToken(signer, runneraction.PlanningSessionScope{
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		SessionID:      session.ID,
		CanvasRunID:    *session.CanvasRunID,
	}, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/surveys", bytes.NewReader([]byte(
		`{"questions":[{"prompt":"What is the priority?","options":["High","Low"]}]}`,
	)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	require.Len(t, reloaded.CurrentSurvey().Questions, 1)
	assert.Equal(t, "What is the priority?", reloaded.CurrentSurvey().Questions[0].Prompt)
	assert.Equal(t, []string{"High", "Low"}, reloaded.CurrentSurvey().Questions[0].Options)
}

func TestRunnerPlanningSessionRejectsOtherToken(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":       "other",
		"org_id":        r.Organization.ID.String(),
		"factory_id":    uuid.New().String(),
		"session_id":    uuid.New().String(),
		"canvas_run_id": uuid.New().String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConsumeResolvedWaitTreatsDoubleConsumeAsMiss(t *testing.T) {
	r := support.Setup(t)
	_, session, factoryModel, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	requireResolvedCreatedWait(t, db, session, factoryModel)

	winner, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	loser, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	require.Equal(t, models.PlanningWaitResolved, winner.WaitState)
	require.Equal(t, models.PlanningWaitResolved, loser.WaitState)

	result, consumed, err := consumeResolvedWait(winner, db)
	require.NoError(t, err)
	assert.True(t, consumed)
	assert.Equal(t, models.PlanningWaitKindCreated, result.Kind)

	_, consumed, err = consumeResolvedWait(loser, db)
	require.NoError(t, err)
	assert.False(t, consumed)
}

func TestRunnerPlanningWaitDoubleConsumeReturnsPending(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	requireResolvedCreatedWait(t, db, session, factoryModel)

	const waiters = 2
	recs := make([]*httptest.ResponseRecorder, waiters)
	var wg sync.WaitGroup
	wg.Add(waiters)
	for i := range recs {
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=1", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.Router.ServeHTTP(rec, req)
			recs[i] = rec
		}(i)
	}
	wg.Wait()

	createdCount := 0
	for _, rec := range recs {
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		status, _ := body["status"].(string)
		require.Contains(t, []string{models.PlanningWaitKindCreated, "pending"}, status)
		if status == models.PlanningWaitKindCreated {
			createdCount++
		}
	}
	assert.Equal(t, 1, createdCount)
}

func mustPlanningRunnerSession(t *testing.T, r *support.ResourceRegistry) (*Server, *models.FactoryPlanningSession, *models.Factory, string) {
	t.Helper()
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	session, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	return server, session, factoryModel, mustPlanningRunnerToken(t, signer, session)
}

func mustPlanningRunnerToken(t *testing.T, signer *jwt.Signer, session *models.FactoryPlanningSession) string {
	t.Helper()
	require.NotNil(t, session.CanvasRunID)
	token, err := runneraction.MintPlanningSessionToken(signer, runneraction.PlanningSessionScope{
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		SessionID:      session.ID,
		CanvasRunID:    *session.CanvasRunID,
	}, time.Hour)
	require.NoError(t, err)
	return token
}

func requireResolvedCreatedWait(t *testing.T, db *gorm.DB, session *models.FactoryPlanningSession, factoryModel *models.Factory) {
	t.Helper()
	require.NoError(t, session.ProposeDraft(db, models.PlanningSessionDraft{
		Title:       "Retry refunds",
		Description: "Stop double charges.",
	}))
	require.NoError(t, session.BeginWait(db))
	_, err := session.CreateDraftWorkOrder(db, factoryModel, session.CreatedByUserID)
	require.NoError(t, err)
	require.Equal(t, models.PlanningWaitResolved, session.WaitState)
}
