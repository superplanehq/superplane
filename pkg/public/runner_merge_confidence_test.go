package public

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
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

	t.Setenv("JWT_SECRET", "test")
	ctx := core.ExecutionContext{ID: nodeExecution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(), OrganizationID: r.Organization.ID.String(), NodeID: nodeID, BaseURL: "http://app.test"}
	token := runnerTokenFrom(runneraction.AttachRunnerEnv(ctx, nil, 60, &runneraction.MergeConfidenceScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: nodeExecution.ID,
		EnabledChecks:   []string{"risk"},
	}))
	require.NotEmpty(t, token)
	parsed, err := runneraction.ParseRunnerToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, order.ID, parsed.WorkOrderID)
	owners := suggestRunnerOwners(t, server, token, `{}`)
	require.Equal(t, http.StatusOK, owners.Code, owners.Body.String())
	assert.NotEmpty(t, decodeSuggestOwners(t, owners).Members)

	unknownScope := *parsed
	unknownScope.CanvasRunID = uuid.New()
	unknownRun, err := runneraction.MintRunnerToken(signer, unknownScope, time.Hour)
	require.NoError(t, err)

	const reportPath = "/api/v1/runner/task/merge-checks"
	const reportBody = `{"check":"risk","score":4,"summary":"Higher risk because the pull request raises the limit."}`
	unauthorized := runnerRequest(t, server, http.MethodPost, "", reportPath, reportBody)
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	disabled := runnerRequest(t, server, http.MethodPost, token, reportPath, `{"check":"performance","score":5,"summary":"No performance practice applies."}`)
	assert.Equal(t, http.StatusBadRequest, disabled.Code)

	missingRun := runnerRequest(t, server, http.MethodPost, unknownRun, reportPath, reportBody)
	assert.Equal(t, http.StatusForbidden, missingRun.Code)

	report := runnerRequest(t, server, http.MethodPost, token, reportPath, reportBody)
	require.Equal(t, http.StatusOK, report.Code, report.Body.String())

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "risk-review", checks[0].Key)
	assert.Equal(t, "Blast radius", checks[0].Name)
	assert.Equal(t, 4.0, checks[0].Score)
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

	require.NoError(t, db.Model(nodeExecution).Update("state", models.CanvasNodeExecutionStateFinished).Error)
	ownerRec := assignRunnerOwner(t, server, token, `{"user":"`+r.User.String()+`","reason":"Owns pkg."}`)
	assert.Equal(t, http.StatusForbidden, ownerRec.Code)
	late := runnerRequest(t, server, http.MethodPost, token, reportPath, `{"check":"risk","score":1,"summary":"Lower risk because the pull request is introducing user interface changes only."}`)
	assert.Equal(t, http.StatusForbidden, late.Code)

	checks, err = order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, 4.0, checks[0].Score)
}
