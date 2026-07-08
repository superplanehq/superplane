package e2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasStagingHiddenOnLiveView(t *testing.T) {
	t.Run("staged nodes are not visible on live view until commit", func(t *testing.T) {
		steps := &canvasStagingLiveViewSteps{t: t}
		steps.start()
		steps.givenACanvasOnLiveView()
		steps.whenIEnterEditMode()
		steps.whenIStageNode("StagedOnlyNode")
		steps.thenNodeIsVisibleInEditor("StagedOnlyNode")
		steps.thenNodeIsOnlyStagedNotLive("StagedOnlyNode")
		steps.whenIExitEditMode()
		steps.thenNodeIsHiddenOnLiveView("StagedOnlyNode")
		steps.thenNodeIsOnlyStagedNotLive("StagedOnlyNode")
		steps.whenIEnterEditMode()
		steps.thenNodeIsVisibleInEditor("StagedOnlyNode")
	})
}

type canvasStagingLiveViewSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasStagingLiveViewSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasStagingLiveViewSteps) givenACanvasOnLiveView() {
	s.canvas = shared.NewCanvasSteps("E2E Staging Live View", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *canvasStagingLiveViewSteps) whenIEnterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *canvasStagingLiveViewSteps) whenIStageNode(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
	s.canvas.WaitForStagingOnCurrentDraft()
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasStagingLiveViewSteps) whenIExitEditMode() {
	s.canvas.ExitEditMode()
}

func (s *canvasStagingLiveViewSteps) thenNodeIsVisibleInEditor(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *canvasStagingLiveViewSteps) thenNodeIsHiddenOnLiveView(nodeName string) {
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
	s.session.AssertHidden(q.TestID("node", nodeName, "header"))
}

func (s *canvasStagingLiveViewSteps) thenNodeIsOnlyStagedNotLive(nodeName string) {
	s.canvas.AssertHasStaging(uuid.Nil)
	require.True(s.t, s.canvas.StagingContainsNodeForUser(s.canvas.UserIDForEmail(s.session.Account.Email), nodeName))
	s.canvas.AssertLiveCanvasLacksNode(nodeName)
	s.canvas.AssertLiveVersionLacksNode(nodeName)
}
