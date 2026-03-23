package e2e

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasGroups(t *testing.T) {
	t.Run("grouping two nodes creates a group container", func(t *testing.T) {
		steps := &CanvasGroupSteps{t: t}
		steps.start()
		steps.givenACanvasWithTwoNoopNodes("GroupAlpha", "GroupBeta")
		steps.selectNodesForGrouping("GroupAlpha", "GroupBeta")
		steps.clickGroupInSelectionToolbar()
		steps.assertGroupNodeVisible()
		steps.assertChildNodeNamesVisible("GroupAlpha", "GroupBeta")
	})

	t.Run("ungroup restores nodes to the canvas", func(t *testing.T) {
		steps := &CanvasGroupSteps{t: t}
		steps.start()
		steps.givenACanvasWithTwoNoopNodes("UngroupA", "UngroupB")
		steps.selectNodesForGrouping("UngroupA", "UngroupB")
		steps.clickGroupInSelectionToolbar()
		steps.assertGroupNodeVisible()
		steps.hoverGroupNodeAndUngroup()
		steps.assertGroupNodeNotVisible()
		steps.assertChildNodeNamesVisible("UngroupA", "UngroupB")
	})
}

type CanvasGroupSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *CanvasGroupSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *CanvasGroupSteps) givenACanvasWithTwoNoopNodes(first, second string) {
	s.canvas = shared.NewCanvasSteps("E2E Canvas Groups", s.t, s.session)
	s.canvas.Create()
	s.canvas.AddNoop(first, models.Position{X: 200, Y: 220})
	s.canvas.AddNoop(second, models.Position{X: 200, Y: 420})
	s.canvas.ClickOnEmptyCanvasArea()
	s.session.Sleep(500)
}

func (s *CanvasGroupSteps) selectNodesForGrouping(first, second string) {
	s.session.Click(q.TestID("node", first, "header"))
	s.session.Sleep(300)
	s.session.ClickWithControlOrMeta(q.TestID("node", second, "header"))
	s.session.Sleep(500)
	s.session.AssertVisible(q.TestID("multi-select-group"))
}

func (s *CanvasGroupSteps) clickGroupInSelectionToolbar() {
	s.session.Click(q.TestID("multi-select-group"))
	s.session.Sleep(800)
}

func (s *CanvasGroupSteps) assertGroupNodeVisible() {
	s.session.AssertVisible(q.TestID("canvas", "group", "node"))
}

func (s *CanvasGroupSteps) assertGroupNodeNotVisible() {
	s.session.AssertHidden(q.TestID("canvas", "group", "node"))
}

func (s *CanvasGroupSteps) assertChildNodeNamesVisible(names ...string) {
	for _, name := range names {
		s.session.AssertText(name)
	}
}

func (s *CanvasGroupSteps) hoverGroupNodeAndUngroup() {
	s.session.HoverOver(q.TestID("canvas", "group", "node"))
	s.session.Sleep(250)
	s.session.Click(q.Locator(`button[aria-label="Ungroup nodes"]`))
	s.session.Sleep(800)
}
