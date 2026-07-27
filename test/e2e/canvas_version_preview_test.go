package e2e

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasVersionPreview(t *testing.T) {
	t.Run("see current version returns to editable live canvas after previewing history", func(t *testing.T) {
		steps := &canvasVersionPreviewSteps{t: t}
		steps.start()
		steps.givenACanvasWithVersionHistory()
		steps.whenIEnterEditMode()
		steps.whenIOpenVersionsSidebar()
		steps.whenIPreviewTheFirstVersion()
		steps.thenPreviousVersionPreviewBarIsVisible()
		steps.whenIClickSeeCurrentVersion()
		steps.thenEditSessionIsReadyOnLiveVersion()
	})
}

type canvasVersionPreviewSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasVersionPreviewSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasVersionPreviewSteps) givenACanvasWithVersionHistory() {
	s.canvas = shared.NewCanvasSteps("E2E Version Preview", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.session.AssertVisible(q.TestID("canvas-edit-button"))

	s.canvas.EnterEditMode()
	s.canvas.AddNoop("HistoryNode", models.Position{X: 500, Y: 200})
	s.canvas.WaitForStagingOnCurrentDraft()
	s.canvas.ClickOnEmptyCanvasArea()
	s.canvas.CommitStaging()
	s.canvas.AssertVersionCountAtLeast(2)
	s.canvas.ExitEditMode()
}

func (s *canvasVersionPreviewSteps) whenIEnterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *canvasVersionPreviewSteps) whenIOpenVersionsSidebar() {
	s.canvas.OpenVersionsSidebar()
}

func (s *canvasVersionPreviewSteps) whenIPreviewTheFirstVersion() {
	s.canvas.SelectVersionInHistorySidebar("v1")
}

func (s *canvasVersionPreviewSteps) thenPreviousVersionPreviewBarIsVisible() {
	s.canvas.AssertPreviewingPreviousVersionBarVisible()
}

func (s *canvasVersionPreviewSteps) whenIClickSeeCurrentVersion() {
	s.canvas.ClickSeeCurrentVersionFromPreviewBar()
}

func (s *canvasVersionPreviewSteps) thenEditSessionIsReadyOnLiveVersion() {
	s.canvas.AssertEditSessionReady()
	s.session.AssertText("HistoryNode")
}
