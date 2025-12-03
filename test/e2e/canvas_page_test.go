package e2e

import (
	"strings"
	"testing"

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
		steps.saveCanvas()
		steps.assertNodeIsAdded("Hello")
	})

	t.Run("duplicating a node on canvas", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.saveCanvas()
		steps.duplicateNodeOnCanvas("Hello")
		steps.assertUnsavedChangesNoteIsVisible()
		steps.canvas.RenameNode("Hello", "Hello previous")
		steps.saveCanvas()
		steps.assertNodeDuplicatedInDB("Hello previous", "Hello")
	})

	t.Run("run is disabled when you have unsaved changes", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.assertUnsavedChangesNoteIsVisible()
		steps.assertCantRunNode("Hello")
		steps.assertExplainationIsShownWhenHoverOverRun()
	})

	t.Run("deleting a node from a canvas", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExistsWithANoopNode()
		steps.deleteNodeFromCanvas("DeleteMe")
		steps.assertUnsavedChangesNoteIsVisible()
		steps.saveCanvas()
		steps.assertNodeDeletedInDB("DeleteMe")
	})

	t.Run("viewing queued items in the sidebar", func(t *testing.T) {
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems()
		steps.openSidebarForNode("Wait")
		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 3)
		steps.assertQueuedItemsVisibleInSidebar()
	})

	t.Run("canceling queued items from the sidebar", func(t *testing.T) {
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems()
		steps.openSidebarForNode("Wait")

		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 3)
		steps.cancelFirstQueueItemFromSidebar()
		steps.assertQueuedItemsCount("Wait", 2)
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

func (s *CanvasPageSteps) assertUnsavedChangesNoteIsVisible() {
	s.session.AssertText("You have unsaved changes")
}

func (s *CanvasPageSteps) assertCantRunNode(nodeName string) {
	dropdown := q.TestID("node", nodeName, "header-dropdown")
	runOption := q.Locator("button:has-text('Run')")

	s.session.Click(dropdown)
	s.session.AssertVisible(runOption)
	s.session.AssertDisabled(runOption)
}

func (s *CanvasPageSteps) assertExplainationIsShownWhenHoverOverRun() {
	runOption := q.Locator("button:has-text('Run')")

	s.session.HoverOver(runOption)
	s.session.AssertText("Save canvas changes before running")
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
	s.canvas.Save()
}

func (s *CanvasPageSteps) duplicateNodeOnCanvas(nodeName string) {
	s.session.Click(q.TestID("node", nodeName, "header-dropdown"))
	s.session.Click(q.TestID("node-action-duplicate"))
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
	wf, err := models.FindWorkflow(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindWorkflowNodes(wf.ID)
	require.NoError(s.t, err)

	for _, n := range nodes {
		if n.Name == nodeName {
			s.t.Fatalf("expected node %q to be deleted, but it still exists in DB", nodeName)
		}
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

func (s *CanvasPageSteps) givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems() {
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

	for i := 0; i < 4; i++ {
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
	s.session.AssertText("Next in queue")
}

func (s *CanvasPageSteps) cancelFirstQueueItemFromSidebar() {
	s.session.Click(q.Locator("h2:has-text('Next in queue') ~ div button[aria-label='Open actions']"))
	s.session.TakeScreenshot()
	s.session.Sleep(300)
	s.session.Click(q.TestID("cancel-queue-item"))
	s.session.TakeScreenshot()
	s.session.Sleep(500) // wait for the cancellation to be processed
}
