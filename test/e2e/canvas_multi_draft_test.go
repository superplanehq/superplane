package e2e

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasMultiDraft(t *testing.T) {
	t.Run("creating and switching between multiple draft branches", func(t *testing.T) {
		steps := &canvasMultiDraftSteps{t: t}
		steps.start()
		steps.givenAPublishedCanvas()
		steps.whenICreateFirstDraftWithNode("AlphaNode")
		steps.whenIExitEditMode()
		steps.thenDraftCountIs(1)
		steps.whenICreateSecondDraftWithNode("BetaNode")
		steps.whenIExitEditMode()
		steps.thenDraftCountIs(2)
		steps.thenDraftBranchesAreVisibleInSidebar("Draft #1", "Draft #2")
		steps.whenIOpenDraftBranch("Draft #1")
		steps.thenNodeIsVisible("AlphaNode")
		steps.thenNodeIsHidden("BetaNode")
		steps.whenIOpenDraftBranch("Draft #2")
		steps.thenNodeIsVisible("BetaNode")
		steps.thenNodeIsHidden("AlphaNode")
	})
}

type canvasMultiDraftSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasMultiDraftSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasMultiDraftSteps) givenAPublishedCanvas() {
	s.canvas = shared.NewCanvasSteps("E2E Multi Draft", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *canvasMultiDraftSteps) whenICreateFirstDraftWithNode(name string) {
	s.canvas.EnterEditMode()
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
	s.canvas.Save()
}

func (s *canvasMultiDraftSteps) whenICreateSecondDraftWithNode(name string) {
	s.canvas.CreateNewDraftFromEditMenu()
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 400})
	s.canvas.Save()
}

func (s *canvasMultiDraftSteps) whenIExitEditMode() {
	s.canvas.ExitEditMode()
}

func (s *canvasMultiDraftSteps) thenDraftCountIs(expected int) {
	s.canvas.AssertDraftCount(expected)
}

func (s *canvasMultiDraftSteps) thenDraftBranchesAreVisibleInSidebar(displayNames ...string) {
	s.canvas.AssertDraftBranchesInSidebar(displayNames...)
}

func (s *canvasMultiDraftSteps) whenIOpenDraftBranch(displayName string) {
	s.canvas.OpenDraftBranchInSidebar(displayName)
}

func (s *canvasMultiDraftSteps) thenNodeIsVisible(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *canvasMultiDraftSteps) thenNodeIsHidden(nodeName string) {
	s.session.AssertHidden(q.TestID("node", nodeName, "header"))
}
