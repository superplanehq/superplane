package e2e

import (
	"testing"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasSidebarClose(t *testing.T) {
	t.Run("building blocks sidebar is not shown after exiting edit mode on versioned canvas", func(t *testing.T) {
		steps := &sidebarCloseSteps{t: t}
		steps.start()
		steps.givenCanvas("E2E Sidebar Close")
		steps.enterEditMode()
		steps.openBuildingBlocksSidebar()
		steps.assertSidebarVisible()
		steps.exitEditMode()
		steps.assertSidebarHidden()
	})
}

type sidebarCloseSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *sidebarCloseSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *sidebarCloseSteps) givenCanvas(name string) {
	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *sidebarCloseSteps) enterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *sidebarCloseSteps) openBuildingBlocksSidebar() {
	s.canvas.OpenBuildingBlocksSidebar()
}

func (s *sidebarCloseSteps) assertSidebarVisible() {
	s.session.AssertVisible(q.TestID("building-blocks-sidebar"))
}

func (s *sidebarCloseSteps) assertSidebarHidden() {
	s.session.AssertHidden(q.TestID("building-blocks-sidebar"))
}

func (s *sidebarCloseSteps) exitEditMode() {
	s.canvas.ExitEditMode()
}
