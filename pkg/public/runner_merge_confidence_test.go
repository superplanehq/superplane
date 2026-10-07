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

func TestRunnerMergeConfidenceCheck(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)

	const nodeID = "assess-risk"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: nodeID,
		Name:   "Assess risk",
		Type:   models.NodeTypeComponent,
	}}, nil)
	require.NoError(t, db.Model(canvas).Updates(map[string]any{
		"factory_id": factoryModel.ID,
		"name":       "Merge confidence",
	}).Error)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	nodeExecution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
	require.NoError(t, db.Model(nodeExecution).Update("state", models.CanvasNodeExecutionStateStarted).Error)

	token, err := runneraction.MintMergeConfidenceToken(signer, runneraction.MergeConfidenceScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: nodeExecution.ID,
		EnabledChecks:   []string{"risk"},
		CheckMaxScores:  map[string]int{"risk": 3},
	}, time.Hour)
	require.NoError(t, err)
	unknownRun, err := runneraction.MintMergeConfidenceToken(signer, runneraction.MergeConfidenceScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasRunID:     uuid.New(),
		NodeExecutionID: nodeExecution.ID,
		EnabledChecks:   []string{"risk"},
	}, time.Hour)
	require.NoError(t, err)

	unauthorized := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"risk","score":4,"summary":"Higher risk because the pull request raises the limit."}`,
	)))
	unauthorizedRec := httptest.NewRecorder()
	server.Router.ServeHTTP(unauthorizedRec, unauthorized)
	assert.Equal(t, http.StatusUnauthorized, unauthorizedRec.Code)

	disabled := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"performance","score":3,"summary":"No performance practice applies."}`,
	)))
	disabled.Header.Set("Authorization", "Bearer "+token)
	disabledRec := httptest.NewRecorder()
	server.Router.ServeHTTP(disabledRec, disabled)
	assert.Equal(t, http.StatusBadRequest, disabledRec.Code)

	rejected := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"risk","score":4,"summary":"Higher risk because the pull request raises the limit."}`,
	)))
	rejected.Header.Set("Authorization", "Bearer "+token)
	rejectedRec := httptest.NewRecorder()
	server.Router.ServeHTTP(rejectedRec, rejected)
	assert.Equal(t, http.StatusBadRequest, rejectedRec.Code)
	assert.Contains(t, rejectedRec.Body.String(), "1 through 3")

	missingRun := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"risk","score":4,"summary":"Higher risk because the pull request raises the limit."}`,
	)))
	missingRun.Header.Set("Authorization", "Bearer "+unknownRun)
	missingRunRec := httptest.NewRecorder()
	server.Router.ServeHTTP(missingRunRec, missingRun)
	assert.Equal(t, http.StatusNotFound, missingRunRec.Code)

	report := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"risk","score":3,"summary":"Higher risk because the pull request raises the limit."}`,
	)))
	report.Header.Set("Authorization", "Bearer "+token)
	reportRec := httptest.NewRecorder()
	server.Router.ServeHTTP(reportRec, report)
	require.Equal(t, http.StatusOK, reportRec.Code, reportRec.Body.String())

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "risk-review", checks[0].Key)
	assert.Equal(t, "Blast radius", checks[0].Name)
	assert.Equal(t, 3.0, checks[0].Score)
	assert.Equal(t, models.FactoryWorkOrderCheckLevelCritical, checks[0].Level)
	assert.Equal(t, "Higher risk because the pull request raises the limit.", checks[0].Summary)
	require.NotNil(t, checks[0].RunID)
	assert.Equal(t, run.ID, *checks[0].RunID)
	automation, err := checks[0].AutomationRef()
	require.NoError(t, err)
	require.NotNil(t, automation)
	assert.Equal(t, canvas.ID, automation.AppID)
	assert.Equal(t, "Merge confidence", automation.AppName)
	assert.Equal(t, nodeID, automation.NodeID)
	assert.Equal(t, "Assess risk", automation.NodeName)

	legacyToken, err := runneraction.MintMergeConfidenceToken(signer, runneraction.MergeConfidenceScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: nodeExecution.ID,
		EnabledChecks:   []string{"performance"},
	}, time.Hour)
	require.NoError(t, err)
	legacy := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"performance","score":5,"summary":"No performance practice applies."}`,
	)))
	legacy.Header.Set("Authorization", "Bearer "+legacyToken)
	legacyRec := httptest.NewRecorder()
	server.Router.ServeHTTP(legacyRec, legacy)
	require.Equal(t, http.StatusOK, legacyRec.Code, legacyRec.Body.String())

	require.NoError(t, db.Model(nodeExecution).Update("state", models.CanvasNodeExecutionStateFinished).Error)
	late := httptest.NewRequest(http.MethodPost, "/api/v1/runner/merge-confidence/checks", bytes.NewReader([]byte(
		`{"check":"risk","score":1,"summary":"Lower risk because the pull request is introducing user interface changes only."}`,
	)))
	late.Header.Set("Authorization", "Bearer "+token)
	lateRec := httptest.NewRecorder()
	server.Router.ServeHTTP(lateRec, late)
	assert.Equal(t, http.StatusNotFound, lateRec.Code)

	checks, err = order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 2)
	var performance *models.FactoryWorkOrderCheck
	for i := range checks {
		if checks[i].Key == "performance-review" {
			performance = &checks[i]
		}
	}
	require.NotNil(t, performance)
	assert.Equal(t, 5.0, performance.Score)
	assert.Equal(t, 5.0, performance.MaxScore)
	assert.Equal(t, models.FactoryWorkOrderCheckLevelPositive, performance.Level)
}
