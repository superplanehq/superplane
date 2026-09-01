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
	assert.Equal(t, "Retry refunds", reloaded.PendingDraft.Data().Title)
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
