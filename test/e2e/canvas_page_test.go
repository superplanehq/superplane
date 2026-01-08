package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasPage(t *testing.T) {
	steps := &CanvasPageSteps{t: t}

	t.Run("adding a node to canvas displays custom node name", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.assertNodeIsAdded("Hello")
	})

	t.Run("duplicating a node on canvas", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.duplicateNodeOnCanvas("Hello")
		steps.assertUnsavedChangesNoteIsVisible()
		steps.saveCanvas()
		steps.canvas.RenameNode("Hello", "Hello previous")
		steps.assertNodeDuplicatedInDB("Hello previous", "Hello")
	})

	t.Run("collapsing and expanding a node on canvas", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.assertIsNodeExpanded("Hello")
		steps.toggleNodeViewOnCanvas("Hello")
		steps.assertIsNodeCollapsed("Hello")
		steps.toggleNodeViewOnCanvas("Hello")
		steps.assertIsNodeExpanded("Hello")
	})

	t.Run("deleting a node from a canvas", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExistsWithANoopNode()
		steps.deleteNodeFromCanvas("DeleteMe")
		steps.assertNodeDeletedInDB("DeleteMe")
	})

	t.Run("viewing queued items in the sidebar", func(t *testing.T) {
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(4)
		steps.openSidebarForNode("Wait")
		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 3)
		steps.assertQueuedItemsVisibleInSidebar()
	})

	t.Run("canceling queued items from the sidebar", func(t *testing.T) {
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(4)
		steps.openSidebarForNode("Wait")

		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 3)
		steps.cancelFirstQueueItemFromSidebar()
		steps.assertQueuedItemsCount("Wait", 2)
	})

	t.Run("canceling running execution from the sidebar", func(t *testing.T) {
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(1)
		steps.openSidebarForNode("Wait")

		steps.session.Sleep(1000)

		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 0)
		steps.cancelRunningExecutionFromSidebar()
		steps.assertExecutionWasCancelled("Wait")
	})

	t.Run("deleting a connection between nodes", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addTwoNodesAndConnect()
		steps.deleteConnectionBetweenNodes("First", "Second")
		steps.assertNodesAreNotConnectedInDB("First", "Second")
	})
}

type CanvasPageSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *CanvasPageSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *CanvasPageSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("E2E Canvas", s.t, s.session)
	s.canvas.Create()
}

func (s *CanvasPageSteps) addNoop(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
}

func (s *CanvasPageSteps) addTwoNodesAndConnect() {
	s.canvas.AddManualTrigger("First", models.Position{X: 500, Y: 200})
	s.canvas.AddNoop("Second", models.Position{X: 900, Y: 200})
	s.canvas.Connect("First", "Second")
}

func (s *CanvasPageSteps) deleteConnectionBetweenNodes(sourceName, targetName string) {
	s.canvas.DeleteConnection(sourceName, targetName)
}

func (s *CanvasPageSteps) assertUnsavedChangesNoteIsVisible() {
	s.session.AssertText("You have unsaved changes")
}

func (s *CanvasPageSteps) assertIsNodeCollapsed(nodeName string) {
	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Sleep(100)

	s.session.AssertVisible(q.TestID("node", nodeName, "header-dropdown"))
	s.session.AssertText("Detailed view")

	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Sleep(100)
}

func (s *CanvasPageSteps) assertIsNodeExpanded(nodeName string) {
	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Sleep(100)

	s.session.AssertVisible(q.TestID("node", nodeName, "header-dropdown"))
	s.session.AssertText("Compact view")

	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Sleep(100)
}

func (s *CanvasPageSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *CanvasPageSteps) assertNodeIsAdded(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *CanvasPageSteps) givenACanvasExistsWithANoopNode() {
	s.canvas = shared.NewCanvasSteps("E2E Canvas With Noop", s.t, s.session)

	s.canvas.Create()
	s.canvas.Visit()
	s.canvas.AddNoop("DeleteMe", models.Position{X: 500, Y: 200})
}

func (s *CanvasPageSteps) duplicateNodeOnCanvas(nodeName string) {
	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Click(q.TestID("node-action-duplicate"))
}

func (s *CanvasPageSteps) toggleNodeViewOnCanvas(nodeName string) {
	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Click(q.TestID("node-action-toggle-view"))
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) deleteNodeFromCanvas(nodeName string) {
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	dropdown := q.TestID("node-" + safe + "-header-dropdown")
	deleteButton := q.Locator("button:has-text('Delete')")

	s.session.Click(dropdown)
	s.session.Click(deleteButton)
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) assertNodeDeletedInDB(nodeName string) {
	deadline := time.Now().Add(2 * time.Second)

	for {
		wf, err := models.FindWorkflow(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)

		nodes, err := models.FindWorkflowNodes(wf.ID)
		require.NoError(s.t, err)

		exists := false
		for _, n := range nodes {
			if n.Name == nodeName {
				exists = true
				break
			}
		}

		if !exists {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected node %q to be deleted, but it still exists in DB", nodeName)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (s *CanvasPageSteps) assertNodeDuplicatedInDB(originalName, duplicateName string) {
	originalNode := s.canvas.GetNodeFromDB(originalName)
	duplicateNode := s.canvas.GetNodeFromDB(duplicateName)

	require.NotNil(s.t, originalNode, "original node %q not found in DB", originalName)
	require.NotNil(s.t, duplicateNode, "duplicate node %q not found in DB", duplicateName)

	originalPos := originalNode.Position.Data()
	duplicatePos := duplicateNode.Position.Data()

	require.Equal(s.t, originalPos.X+50, duplicatePos.X, "duplicate node X position should be offset by 50")
	require.Equal(s.t, originalPos.Y+50, duplicatePos.Y, "duplicate node Y position should be offset by 50")
}

func (s *CanvasPageSteps) givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(itemsAmount int) {
	s.canvas = shared.NewCanvasSteps("E2E Canvas With Queue", s.t, s.session)

	s.canvas.Create()
	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddWait("Wait", models.Position{X: 1000, Y: 200}, 10, "Seconds")
	s.session.TakeScreenshot()
	s.canvas.Connect("Start", "Wait")
	s.canvas.Save()

	dropdown := q.TestID("node", "start", "header-dropdown")
	runButton := q.Locator("button:has-text('Run')")
	emitEvent := q.Locator("button:has-text('Emit Event')")

	for i := 0; i < itemsAmount; i++ {
		s.session.Click(dropdown)
		s.session.Click(runButton)
		s.session.Click(emitEvent)
		s.session.Sleep(100)
	}

	// wait for the first item to start processing
	s.session.Sleep(500)
}

func (s *CanvasPageSteps) openSidebarForNode(node string) {
	s.session.Click(q.TestID("node", node, "header"))
	s.session.TakeScreenshot()
}

func (s *CanvasPageSteps) assertQueuedItemsCount(nodeName string, expected int) {
	canvas, err := models.FindWorkflow(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindWorkflowNodes(canvas.ID)
	require.NoError(s.t, err)

	var waitNode *models.WorkflowNode
	for _, n := range nodes {
		if n.Name == nodeName {
			waitNode = &n
			break
		}
	}
	require.NotNil(s.t, waitNode, nodeName+" node not found")

	queueItems, err := models.ListNodeQueueItems(waitNode.WorkflowID, waitNode.NodeID, 100, nil)
	require.NoError(s.t, err)

	require.Equal(s.t, expected, len(queueItems))
}

func (s *CanvasPageSteps) assertRunningItemsCount(nodeName string, expected int) {
	canvas, err := models.FindWorkflow(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindWorkflowNodes(canvas.ID)
	require.NoError(s.t, err)

	var waitNode *models.WorkflowNode
	for _, n := range nodes {
		if n.Name == nodeName {
			waitNode = &n
			break
		}
	}
	require.NotNil(s.t, waitNode, nodeName+" node not found")

	var executions []models.WorkflowNodeExecution
	query := database.Conn().
		Where("workflow_id = ?", waitNode.WorkflowID).
		Where("node_id = ?", waitNode.NodeID).
		Order("created_at DESC")

	err = query.Find(&executions).Error
	require.NoError(s.t, err)

	require.Equal(s.t, expected, len(executions))
}

func (s *CanvasPageSteps) assertQueuedItemsVisibleInSidebar() {
	s.session.AssertText("Queued")
}

func (s *CanvasPageSteps) cancelFirstQueueItemFromSidebar() {
	eventItem := q.Locator("h2:has-text('Queued') ~ div")
	s.session.HoverOver(eventItem)
	s.session.Sleep(300) // Wait for hover to register and actions button to appear
	s.session.Click(q.Locator("h2:has-text('Queued') ~ div button[aria-label='Open actions']"))
	s.session.TakeScreenshot()
	s.session.Sleep(300)
	s.session.Click(q.TestID("cancel-queue-item"))
	s.session.TakeScreenshot()
	s.session.Sleep(500) // wait for the cancellation to be processed
}

func (s *CanvasPageSteps) cancelRunningExecutionFromSidebar() {
	eventItem := q.Locator("h2:has-text('Latest') ~ div")
	s.session.HoverOver(eventItem)
	s.session.Sleep(300) // Wait for hover to register and actions button to appear
	s.session.Click(q.Locator("h2:has-text('Latest') ~ div button[aria-label='Open actions']"))
	s.session.TakeScreenshot()
	s.session.Sleep(300)
	s.session.Click(q.TestID("cancel-queue-item"))
	s.session.TakeScreenshot()
	s.session.Sleep(500) // wait for the cancellation to be processed
}

func (s *CanvasPageSteps) assertExecutionWasCancelled(nodeName string) {
	executions := s.canvas.GetExecutionsForNode(nodeName)
	require.Greater(s.t, len(executions), 0, "expected at least one execution")

	execution := executions[0]
	require.Equal(s.t, models.WorkflowNodeExecutionResultCancelled, execution.Result, "expected execution to be cancelled")
}

func (s *CanvasPageSteps) assertNodesAreNotConnectedInDB(sourceName, targetName string) {
	deadline := time.Now().Add(2 * time.Second)

	for {
		workflows := s.canvas.GetWorkflowFromDB()
		sourceNode := s.canvas.GetNodeFromDB(sourceName)
		targetNode := s.canvas.GetNodeFromDB(targetName)

		connected := false
		for _, conn := range workflows.Edges {
			if conn.SourceID == sourceNode.NodeID && conn.TargetID == targetNode.NodeID {
				connected = true
				break
			}
		}

		if !connected {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected nodes %q and %q to not be connected, but connection exists in DB", sourceName, targetName)
		}

		time.Sleep(200 * time.Millisecond)
	}
}
