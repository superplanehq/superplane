package public

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerPlanningSessionAskAndDraft(t *testing.T) {
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

	ask := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/ask", bytes.NewReader([]byte(
		`{"questions":[{"id":"area","prompt":"Which area?","options":["Payments"]}]}`,
	)))
	ask.Header.Set("Authorization", "Bearer "+token)
	askRec := httptest.NewRecorder()
	server.Router.ServeHTTP(askRec, ask)
	require.Equal(t, http.StatusOK, askRec.Code, askRec.Body.String())

	draft := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/drafts", bytes.NewReader([]byte(
		`{"title":"Retry refunds","description":"Stop double charges."}`,
	)))
	draft.Header.Set("Authorization", "Bearer "+token)
	draftRec := httptest.NewRecorder()
	server.Router.ServeHTTP(draftRec, draft)
	require.Equal(t, http.StatusOK, draftRec.Code, draftRec.Body.String())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Retry refunds", reloaded.PendingDraft.Data().Title)
}

func TestRunnerPlanningSessionRejectsWorkOrderToken(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	token, err := runneraction.MintWorkOrderSurveyToken(signer, runneraction.WorkOrderSurveyScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      uuid.New(),
		WorkOrderID:    uuid.New(),
		CanvasRunID:    uuid.New(),
	}, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
