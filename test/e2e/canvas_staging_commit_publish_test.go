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

const stagingCommitMessage = "E2E commit"

func TestCanvasStagingCommitPromoteLive(t *testing.T) {
	t.Run("edit stage commit promotes changes to live end-to-end", func(t *testing.T) {
		steps := &canvasStagingPromoteSteps{t: t}
		steps.start()
		steps.givenACanvasOnLiveView()
		initialLiveVersionID := steps.canvas.LiveVersionID()
		steps.whenIEnterEditMode()
		steps.whenIStageNode("PromotedNode")
		steps.thenNodeIsVisibleInEditor("PromotedNode")
		steps.thenNodeIsOnlyStagedNotLive("PromotedNode")
		steps.whenICommitStaging()
		steps.thenStagingIsCleared()
		steps.thenCommitPromotedNodeToLive("PromotedNode", initialLiveVersionID)
		steps.whenIExitEditMode()
		steps.thenNodeIsVisibleOnLiveView("PromotedNode")
		steps.thenVersionHistoryShowsCommit()
	})
}

type canvasStagingPromoteSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasStagingPromoteSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasStagingPromoteSteps) givenACanvasOnLiveView() {
	s.canvas = shared.NewCanvasSteps("E2E Staging Promote Live", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *canvasStagingPromoteSteps) whenIEnterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *canvasStagingPromoteSteps) whenIStageNode(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
	s.canvas.WaitForStagingOnCurrentDraft()
	s.session.AssertVisible(q.TestID("canvas-commit-staging-button"))
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasStagingPromoteSteps) whenICommitStaging() {
	s.canvas.CommitStaging()
}

func (s *canvasStagingPromoteSteps) whenIExitEditMode() {
	s.canvas.ExitEditMode()
}

func (s *canvasStagingPromoteSteps) thenNodeIsVisibleInEditor(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *canvasStagingPromoteSteps) thenNodeIsOnlyStagedNotLive(nodeName string) {
	s.canvas.AssertHasStaging(uuid.Nil)
	require.True(s.t, s.canvas.StagingContainsNodeForUser(s.canvas.UserIDForEmail(s.session.Account.Email), nodeName))
	s.canvas.AssertLiveCanvasLacksNode(nodeName)
	s.canvas.AssertLiveVersionLacksNode(nodeName)
}

func (s *canvasStagingPromoteSteps) thenStagingIsCleared() {
	s.canvas.AssertNoStaging(uuid.Nil)
}

func (s *canvasStagingPromoteSteps) thenCommitPromotedNodeToLive(nodeName string, previousLiveVersionID uuid.UUID) {
	require.NotEqual(s.t, previousLiveVersionID, s.canvas.LiveVersionID())
	s.canvas.AssertLiveVersionCommitMessage(stagingCommitMessage)
	s.canvas.AssertLiveVersionHasNode(nodeName)
	s.canvas.AssertLiveCanvasHasNode(nodeName)
	s.canvas.AssertVersionCountAtLeast(2)
}

func (s *canvasStagingPromoteSteps) thenNodeIsVisibleOnLiveView(nodeName string) {
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
	s.session.AssertText(nodeName)
}

func (s *canvasStagingPromoteSteps) thenVersionHistoryShowsCommit() {
	s.canvas.EnterEditMode()
	s.canvas.AssertVersionHistoryContains(stagingCommitMessage)
}
