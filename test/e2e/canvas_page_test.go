package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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
		steps.visitCanvasPage()
		steps.addNoop("Hello")
		steps.saveCanvas()
		steps.assertNodeIsAdded("Hello")
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

	// t.Run("canceling queued items from the sidebar for a wait node", func(t *testing.T) {
	// 	steps.start()
	// 	steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems()
	// 	steps.openSidebarForNode("Wait")
	// 	steps.assertSidebarShowsQueueCount(3)
	// 	steps.cancelFirstQueueItemFromSidebar()
	// 	steps.assertSidebarShowsQueueCount(2)
	// })
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

func (s *CanvasPageSteps) visitCanvasPage() {
	s.canvas.Visit()
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

func (s *CanvasPageSteps) givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems() {
	s.canvas = shared.NewCanvasSteps("E2E Canvas With Queue", s.t, s.session)

	s.canvas.Create()
	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddWait("Wait", models.Position{X: 1000, Y: 200})
	s.canvas.Connect("Start", "Wait")
	s.canvas.Save()

	dropdown := q.TestID("node-manual-start-header-dropdown")
	runButton := q.Locator("button:has-text('Run')")

	for i := 0; i < 3; i++ {
		s.session.Click(dropdown)
		s.session.Click(runButton)
		s.session.Sleep(500)
	}
}

func (s *CanvasPageSteps) openSidebarForNode(nodeID string) {
	safe := strings.ToLower(nodeID)
	safe = strings.ReplaceAll(safe, " ", "-")
	s.session.Click(q.TestID("node-" + safe + "-header-dropdown"))
	s.session.Click(q.Locator("button:has-text('View details')"))
	s.session.Sleep(500)
}

func (s *CanvasPageSteps) assertSidebarShowsQueueCount(expected int) {
	if expected == 0 {
		s.session.AssertText("Queue is empty")
		return
	}

	s.session.AssertText("Next in queue")
}

func (s *CanvasPageSteps) cancelFirstQueueItemFromSidebar() {
	s.session.Click(q.Locator("h2:has-text('Next in queue') ~ div button[aria-label='Open actions']"))
	s.session.Click(q.Locator("button:has-text('Cancel')"))
	s.session.Sleep(500)
}
