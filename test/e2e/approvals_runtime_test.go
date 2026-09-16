package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/approval"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"gorm.io/datatypes"
)

const (
	approvalRuntimeStartNodeID    = "start-trigger"
	approvalRuntimeApprovalNodeID = "approval"
	approvalRuntimeOutputNodeID   = "noop-output"
)

func TestApprovalRuntime(t *testing.T) {
	t.Run("running and approving on a canvas", func(t *testing.T) {
		steps := &approvalRuntimeSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerApprovalAndNoop()
		steps.runManualTrigger()
		steps.approveFirstPendingRequirement()
		steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	})

	t.Run("deleting an approval node while a run is waiting cancels the run", func(t *testing.T) {
		steps := &approvalRuntimeSteps{t: t}
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

	t.Run("preventing duplicate approvals across approver types", func(t *testing.T) {
		steps := &approvalRuntimeSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerAnyoneAndUserApprovalAndNoop()
		steps.runManualTrigger()
		steps.approveAnyoneRequirement()
		steps.waitForApprovalMetadata("Approval", 1, 1, approval.ItemTypeAnyone)
		steps.assertNoApproveButtons()
	})

	t.Run("running and approving a role requirement", func(t *testing.T) {
		steps := &approvalRuntimeSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerRoleApprovalAndNoop(models.RoleOrgAdmin)
		steps.runManualTrigger()
		steps.approveFirstPendingRequirement()
		steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	})

	t.Run("running and approving a group requirement", func(t *testing.T) {
		steps := &approvalRuntimeSteps{t: t}
		steps.start()
		groupName := createOrgApprovalGroup(t, steps.session.OrgID)
		addUserToOrgGroup(t, steps.session.OrgID, steps.session.Account.Email, groupName)
		steps.givenCanvasWithManualTriggerGroupApprovalAndNoop(groupName)
		steps.runManualTrigger()
		steps.approveFirstPendingRequirement()
		steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	})
}

type approvalRuntimeSteps struct {
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

func (s *approvalRuntimeSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *approvalRuntimeSteps) givenCanvasWithManualTriggerApprovalAndNoop() {
	s.createPublishedApprovalCanvas([]map[string]any{
		{
			"type": approval.ItemTypeUser,
			"user": s.currentUserID().String(),
		},
	})
}

func (s *approvalRuntimeSteps) givenCanvasWithManualTriggerRoleApprovalAndNoop(roleName string) {
	s.createPublishedApprovalCanvas([]map[string]any{
		{
			"type": approval.ItemTypeRole,
			"role": roleName,
		},
	})
}

func (s *approvalRuntimeSteps) givenCanvasWithManualTriggerGroupApprovalAndNoop(groupName string) {
	s.createPublishedApprovalCanvas([]map[string]any{
		{
			"type":  approval.ItemTypeGroup,
			"group": groupName,
		},
	})
}

func (s *approvalRuntimeSteps) givenCanvasWithManualTriggerAnyoneAndUserApprovalAndNoop() {
	s.createPublishedApprovalCanvas([]map[string]any{
		{"type": approval.ItemTypeAnyone},
		{
			"type": approval.ItemTypeUser,
			"user": s.currentUserID().String(),
		},
	})
}

func (s *approvalRuntimeSteps) createPublishedApprovalCanvas(items []map[string]any) {
	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
	s.canvas.CreatePublished(approvalRuntimeNodes(items), approvalRuntimeEdges())
}

func (s *approvalRuntimeSteps) currentUserID() uuid.UUID {
	user, err := models.FindActiveUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	return user.ID
}

func approvalRuntimeNodes(items []map[string]any) []models.CanvasNode {
	return []models.CanvasNode{
		{
			NodeID: approvalRuntimeStartNodeID,
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID: approvalRuntimeApprovalNodeID,
			Name:   "Approval",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "approval"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"items": items,
			}),
			Position: datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
		{
			NodeID: approvalRuntimeOutputNodeID,
			Name:   "Output",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 1600, Y: 200}),
		},
	}
}

func approvalRuntimeEdges() []models.Edge {
	return []models.Edge{
		{SourceID: approvalRuntimeStartNodeID, TargetID: approvalRuntimeApprovalNodeID, Channel: "default"},
		{SourceID: approvalRuntimeApprovalNodeID, TargetID: approvalRuntimeOutputNodeID, Channel: approval.ChannelApproved},
	}
}

func (s *approvalRuntimeSteps) runManualTrigger() {
	s.canvas.EmitManualTrigger("Start")
	if s.waitForApprovalExecution(90 * time.Second) {
		return
	}

	s.t.Fatalf("timed out waiting for execution of node Approval after emitting manual trigger")
}

func (s *approvalRuntimeSteps) waitForApprovalExecutionToBeWaiting() {
	s.canvas.WaitForExecution("Approval", models.CanvasNodeExecutionStateStarted, 90*time.Second)
}

func (s *approvalRuntimeSteps) waitForApprovalExecution(timeout time.Duration) bool {
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

func (s *approvalRuntimeSteps) rememberWaitingApprovalExecution() {
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

func (s *approvalRuntimeSteps) deleteApprovalNodeFromCanvas() {
	s.canvas.EnterEditMode()
	s.deleteNodeFromCanvas("Approval")
	s.canvas.CommitAndPublish()
}

func (s *approvalRuntimeSteps) deleteNodeFromCanvas(nodeName string) {
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

func (s *approvalRuntimeSteps) assertApprovalNodeDeletedFromDB() {
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

func (s *approvalRuntimeSteps) assertWaitingApprovalExecutionCancelled() {
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

func (s *approvalRuntimeSteps) assertWaitingRunCancelled() {
	require.Eventually(s.t, func() bool {
		run, err := models.FindCanvasRunInTransaction(database.Conn(), s.canvas.WorkflowID, s.waiting.runID)
		return err == nil &&
			run.State == models.CanvasRunStateFinished &&
			run.Result == models.CanvasRunResultCancelled
	}, 30*time.Second, 500*time.Millisecond, "waiting approval run should finish as cancelled")
}

func (s *approvalRuntimeSteps) approveFirstPendingRequirement() {
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

func (s *approvalRuntimeSteps) approveAnyoneRequirement() {
	s.canvas.StartEditingNode("Approval")
	s.session.AssertVisible(q.Locator(`button:has-text("Approve")`))

	item := s.session.Page().Locator(`[data-slot="item"]:has([data-slot="item-title"]:has-text("Any user"))`)
	approveButton := item.Locator(`button:has-text("Approve")`).First()
	count, err := approveButton.Count()
	if err != nil {
		s.t.Fatalf("counting approve buttons for any user: %v", err)
	}
	if count == 0 {
		approveButton = s.session.Page().Locator(`button:has-text("Approve")`).First()
	}
	if err := approveButton.Click(); err != nil {
		s.t.Fatalf("clicking approve button: %v", err)
	}
	commentInput := s.session.Page().Locator(`[placeholder="Enter comment"]`).First()
	if count, err := commentInput.Count(); err == nil && count > 0 {
		if err := commentInput.Fill("Do it"); err != nil {
			s.t.Fatalf("filling approval comment: %v", err)
		}
	}
	s.session.Click(q.Locator(`button:has-text("Confirm Approval")`))
}

func (s *approvalRuntimeSteps) waitForApprovalMetadata(nodeName string, approvedCount int, pendingCount int, approvedType string) {
	found := false
	start := time.Now()

	for time.Since(start) < 5*time.Second {
		executions := s.canvas.GetExecutionsForNode(nodeName)
		if len(executions) == 0 {
			s.session.Sleep(500)
			continue
		}

		metadata := executions[0].Metadata.Data()
		rawRecords, ok := metadata["records"].([]any)
		require.True(s.t, ok, "expected approval records metadata")

		approved := 0
		pending := 0
		approvedTypeMatch := false
		for _, rawRecord := range rawRecords {
			record, ok := rawRecord.(map[string]any)
			require.True(s.t, ok, "expected approval record metadata")
			state, _ := record["state"].(string)
			recordType, _ := record["type"].(string)
			switch state {
			case "approved":
				approved++
				if recordType == approvedType {
					approvedTypeMatch = true
				}
			case "pending":
				pending++
			}
		}

		if approved == approvedCount && pending == pendingCount && approvedTypeMatch {
			found = true
			break
		}

		s.session.Sleep(500)
	}

	require.True(s.t, found, "timed out waiting for approval metadata to update")
}

func (s *approvalRuntimeSteps) assertNoApproveButtons() {
	s.session.AssertHidden(q.Locator(`button:has-text("Approve")`))
}

func (s *approvalRuntimeSteps) assertApprovalExecutionFinishedAndOutputNodeProcessed() {
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 60*time.Second)

	approvalExecs := s.canvas.GetExecutionsForNode("Approval")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, approvalExecs, 1, "expected one execution for approval node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")
}
