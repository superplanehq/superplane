package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestApprovals(t *testing.T) {
	t.Run("running and approving on a canvas", func(t *testing.T) {
		steps := &ApprovalSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerApprovalAndNoop()
		steps.runManualTrigger()
		steps.approveFirstPendingRequirement()
		steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	})

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

func (s *ApprovalSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *ApprovalSteps) givenCanvasWithManualTriggerApprovalAndNoop() {
	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.canvas.EnterEditMode()

	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddApproval("Approval", models.Position{X: 1000, Y: 200})
	s.canvas.AddNoop("Output", models.Position{X: 1600, Y: 200})

	s.canvas.Connect("Start", "Approval")
	s.canvas.Connect("Approval", "Output")

	s.saveCanvas()
	s.canvas.CommitAndPublish()
}

func (s *ApprovalSteps) runManualTrigger() {
	s.canvas.EmitManualTrigger("Start")
	if s.waitForApprovalExecution(90 * time.Second) {
		return
	}

	s.t.Fatalf("timed out waiting for execution of node Approval after emitting manual trigger")
}

func (s *ApprovalSteps) waitForApprovalExecutionToBeWaiting() {
	s.canvas.WaitForExecution("Approval", models.CanvasNodeExecutionStateStarted, 90*time.Second)
}

func (s *ApprovalSteps) waitForApprovalExecution(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		executions := s.canvas.GetExecutionsForNode("Approval")
		if len(executions) > 0 {
			return true
		}
		s.session.Sleep(500)
	}

	return false
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
	s.canvas.EnterEditMode()
	s.deleteNodeFromCanvas("Approval")
	s.canvas.CommitAndPublish()
}

func (s *ApprovalSteps) deleteNodeFromCanvas(nodeName string) {
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	nodeHeader := q.TestID("node", nodeName, "header")
	deleteButton := q.Locator(
		`.react-flow__node:has([data-testid="node-` + safe + `-header"]) [data-testid="node-action-delete"]`,
	)

	s.session.HoverOver(nodeHeader)
	s.session.Sleep(100)
	s.session.Click(deleteButton)
	s.session.Sleep(300)
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

func (s *ApprovalSteps) approveFirstPendingRequirement() {
	s.canvas.StartEditingNode("Approval")
	s.session.Click(q.Locator(`button:has-text("Approve")`))
	commentInput := s.session.Page().Locator(`[placeholder="Enter comment"]`).First()
	if count, err := commentInput.Count(); err == nil && count > 0 {
		if err := commentInput.Fill("Do it"); err != nil {
			s.t.Fatalf("filling approval comment: %v", err)
		}
	}
	s.session.Click(q.Locator(`button:has-text("Confirm Approval")`))
}

func (s *ApprovalSteps) assertApprovalExecutionFinishedAndOutputNodeProcessed() {
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 60*time.Second)

	approvaExecs := s.canvas.GetExecutionsForNode("Approval")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, approvaExecs, 1, "expected one execution for approval node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")
}
