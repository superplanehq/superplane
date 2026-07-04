package e2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasStagingCommit(t *testing.T) {
	t.Run("edit stage commit promotes changes to live", func(t *testing.T) {
		steps := &canvasStagingSteps{t: t}
		steps.start()
		steps.givenACanvasInEditMode()
		steps.whenIAddNode("CommitNode")
		steps.thenStagingExists()
		steps.whenICommitStaging()
		steps.thenStagingIsCleared()
		steps.thenNodeIsLive("CommitNode")
	})
}

type canvasStagingSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasStagingSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasStagingSteps) givenACanvasInEditMode() {
	s.canvas = shared.NewCanvasSteps("E2E Staging Commit", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.canvas.EnterEditMode()
}

func (s *canvasStagingSteps) whenIAddNode(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
	s.session.AssertText(name)
	s.canvas.WaitForStagingOnCurrentDraft()
	s.session.AssertVisible(q.TestID("canvas-commit-staging-button"))
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasStagingSteps) whenICommitStaging() {
	s.canvas.ClickOnEmptyCanvasArea()
	s.canvas.CommitStaging()
}

func (s *canvasStagingSteps) thenStagingExists() {
	s.canvas.AssertHasStaging(uuid.Nil)
}

func (s *canvasStagingSteps) thenStagingIsCleared() {
	s.canvas.AssertNoStaging(uuid.Nil)
}

func (s *canvasStagingSteps) thenNodeIsLive(nodeName string) {
	s.canvas.AssertLiveCanvasHasNode(nodeName)
}
