package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestApprovalDeletionCancelsRun(t *testing.T) {
	t.Run("deleting an approval node while a run is waiting cancels the run", func(t *testing.T) {
		steps := &ApprovalSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerApprovalAndNoop()
		steps.runManualTrigger()
		steps.waitForApprovalExecutionToBeWaiting()
		steps.rememberWaitingApprovalExecution()
		steps.deleteApprovalNodeFromCanvas()
		steps.assertApprovalNodeDeletedFromDB()
		steps.assertWaitingApprovalExecutionCancelled()
		steps.assertWaitingRunCancelled()
	})
}

type ApprovalSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
	waiting waitingApproval
}

type waitingApproval struct {
	nodeID      string
	executionID uuid.UUID
	runID       uuid.UUID
}

func (s *ApprovalSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *ApprovalSteps) givenCanvasWithManualTriggerApprovalAndNoop() {
	user, err := models.FindActiveUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: "start", Name: "Start", Type: models.NodeTypeTrigger,
			Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
		},
		{
			NodeID: "approval", Name: "Approval", Type: models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "approval"}}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"items": []any{map[string]any{"type": "user", "user": user.ID.String()}},
			}),
		},
		{
			NodeID: "output", Name: "Output", Type: models.NodeTypeComponent,
			Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
		},
	}, []models.Edge{
		{SourceID: "start", TargetID: "approval", Channel: "default"},
		{SourceID: "approval", TargetID: "output", Channel: "approved"},
	})
	s.canvas = shared.NewCanvasSteps(canvas.Name, s.t, s.session)
	s.canvas.WorkflowID = canvas.ID
}

func (s *ApprovalSteps) runManualTrigger() {
	s.canvas.EmitManualTrigger("Start")
}

func (s *ApprovalSteps) waitForApprovalExecutionToBeWaiting() {
	s.canvas.WaitForExecution("Approval", models.CanvasNodeExecutionStateStarted, 90*time.Second)
}

func (s *ApprovalSteps) rememberWaitingApprovalExecution() {
	node := s.canvas.GetNodeFromDB("Approval")
	executions := s.canvas.GetExecutionsForNode("Approval")
	require.NotEmpty(s.t, executions, "expected a waiting approval execution")

	s.waiting = waitingApproval{
		nodeID:      node.NodeID,
		executionID: executions[0].ID,
		runID:       executions[0].RunID,
	}
	require.NotEqual(s.t, uuid.Nil, s.waiting.runID, "expected approval execution to belong to a run")
}

func (s *ApprovalSteps) deleteApprovalNodeFromCanvas() {
	version, err := models.FindLiveCanvasVersion(s.canvas.WorkflowID)
	require.NoError(s.t, err)
	version.Nodes = slices.DeleteFunc(version.Nodes, func(node models.Node) bool {
		return node.ID == s.waiting.nodeID
	})
	version.Edges = slices.DeleteFunc(version.Edges, func(edge models.Edge) bool {
		return edge.SourceID == s.waiting.nodeID || edge.TargetID == s.waiting.nodeID
	})
	canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err)
	content, err := yaml.VersionToCanvasYAML(canvas.Name, canvas.Description, version)
	require.NoError(s.t, err)

	// Use the real HTTP commit path so publication must notify the workers.
	path := "/api/v1/canvases/" + s.canvas.WorkflowID.String() + "/staging"
	s.sendCanvasRequest(http.MethodPut, path, map[string]any{
		"operations": []map[string]any{{"path": "canvas.yaml", "content": base64.StdEncoding.EncodeToString([]byte(content))}},
	})
	s.sendCanvasRequest(http.MethodPost, path+"/commit", map[string]any{"commitMessage": "Remove waiting approval"})
}

func (s *ApprovalSteps) sendCanvasRequest(method, path string, body any) {
	encoded, err := json.Marshal(body)
	require.NoError(s.t, err)
	request, err := http.NewRequestWithContext(s.t.Context(), method, s.session.BaseURL+path, bytes.NewReader(encoded))
	require.NoError(s.t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Organization-Id", s.session.OrgID.String())
	cookies, err := s.session.Page().Context().Cookies(s.session.BaseURL)
	require.NoError(s.t, err)
	for _, cookie := range cookies {
		request.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	}
	response, err := http.DefaultClient.Do(request)
	require.NoError(s.t, err)
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(s.t, err)
	require.Equal(s.t, http.StatusOK, response.StatusCode, "%s %s: %s", method, path, responseBody)
}

func (s *ApprovalSteps) assertApprovalNodeDeletedFromDB() {
	require.Eventually(s.t, func() bool {
		var count int64
		err := database.Conn().
			Model(&models.CanvasNode{}).
			Where("workflow_id = ?", s.canvas.WorkflowID).
			Where("node_id = ?", s.waiting.nodeID).
			Count(&count).
			Error
		return err == nil && count == 0
	}, 10*time.Second, 250*time.Millisecond, "approval node should be deleted from active canvas nodes")
}

func (s *ApprovalSteps) assertWaitingApprovalExecutionCancelled() {
	require.Eventually(s.t, func() bool {
		var execution models.CanvasNodeExecution
		err := database.Conn().
			Where("workflow_id = ?", s.canvas.WorkflowID).
			Where("id = ?", s.waiting.executionID).
			First(&execution).
			Error
		return err == nil &&
			execution.State == models.CanvasNodeExecutionStateFinished &&
			execution.Result == models.CanvasNodeExecutionResultCancelled
	}, 30*time.Second, 500*time.Millisecond, "waiting approval execution should be cancelled")
}

func (s *ApprovalSteps) assertWaitingRunCancelled() {
	require.Eventually(s.t, func() bool {
		run, err := models.FindCanvasRunInTransaction(database.Conn(), s.canvas.WorkflowID, s.waiting.runID)
		return err == nil &&
			run.State == models.CanvasRunStateFinished &&
			run.Result == models.CanvasRunResultCancelled
	}, 30*time.Second, 500*time.Millisecond, "waiting approval run should finish as cancelled")
}
