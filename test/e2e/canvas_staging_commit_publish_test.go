package e2e

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasStagingCommitPublish(t *testing.T) {
	t.Run("edit stage commit and publish promotes changes to live", func(t *testing.T) {
		steps := &canvasStagingSteps{t: t}
		steps.start()
		steps.givenACanvasInEditMode()
		steps.whenIAddNode("CommitPublishNode")
		steps.thenStagingExistsForCurrentDraft()
		steps.thenNodeIsNotCommittedInCurrentDraft("CommitPublishNode")
		steps.whenICommitStaging()
		steps.thenStagingIsClearedForCurrentDraft()
		steps.thenNodeIsCommittedInCurrentDraft("CommitPublishNode")
		steps.whenIPublish()
		steps.thenNodeIsLive("CommitPublishNode")
	})

	t.Run("staging is isolated per draft branch", func(t *testing.T) {
		steps := &canvasStagingSteps{t: t}
		steps.start()
		steps.givenACanvas()
		steps.whenICreateFirstDraftWithStagedNode("AlphaNode")
		steps.whenIExitEditMode()
		steps.whenICreateSecondDraftWithStagedNode("BetaNode")
		steps.whenIExitEditMode()
		steps.thenBothDraftsHaveStaging()
		steps.whenIOpenDraftBranch("Draft #1")
		steps.thenNodeIsVisible("AlphaNode")
		steps.thenNodeIsHidden("BetaNode")
		steps.thenStagingExistsForDraft("Draft #1")
		steps.whenIOpenDraftBranch("Draft #2")
		steps.thenNodeIsVisible("BetaNode")
		steps.thenNodeIsHidden("AlphaNode")
		steps.thenStagingExistsForDraft("Draft #2")
		steps.whenICommitStagingOnDraft("Draft #1")
		steps.thenDraftIsCommittedWithoutStaging("Draft #1", "AlphaNode")
		steps.thenDraftStillHasStaging("Draft #2")
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

func (s *canvasStagingSteps) givenACanvas() {
	s.canvas = shared.NewCanvasSteps("E2E Staging Commit Publish", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *canvasStagingSteps) givenACanvasInEditMode() {
	s.givenACanvas()
	s.canvas.EnterEditMode()
}

func (s *canvasStagingSteps) whenIAddNode(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
	s.session.AssertText(name)
	s.canvas.WaitForStagingOnCurrentDraft()
	s.session.AssertVisible(q.TestID("canvas-commit-staging-button"))
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasStagingSteps) whenICreateFirstDraftWithStagedNode(name string) {
	s.canvas.EnterEditMode()
	s.whenIAddNode(name)
}

func (s *canvasStagingSteps) whenICreateSecondDraftWithStagedNode(name string) {
	s.canvas.CreateNewDraftFromEditMenu()
	s.whenIAddNode(name)
}

func (s *canvasStagingSteps) whenIExitEditMode() {
	s.canvas.ExitEditMode()
}

func (s *canvasStagingSteps) whenICommitStaging() {
	s.canvas.ClickOnEmptyCanvasArea()
	s.canvas.CommitStaging()
}

func (s *canvasStagingSteps) whenICommitStagingOnDraft(displayName string) {
	s.canvas.OpenDraftBranchInSidebar(displayName)
	s.canvas.CommitStaging()
}

func (s *canvasStagingSteps) whenIPublish() {
	s.canvas.Publish()
}

func (s *canvasStagingSteps) whenIOpenDraftBranch(displayName string) {
	s.canvas.OpenDraftBranchInSidebar(displayName)
}

func (s *canvasStagingSteps) thenStagingExistsForCurrentDraft() {
	draft := s.canvas.FindCurrentDraft()
	s.canvas.AssertHasStaging(draft.ID)
}

func (s *canvasStagingSteps) thenStagingExistsForDraft(displayName string) {
	draft := s.canvas.FindDraftByDisplayName(displayName)
	s.canvas.AssertHasStaging(draft.ID)
}

func (s *canvasStagingSteps) thenStagingIsClearedForCurrentDraft() {
	draft := s.canvas.FindCurrentDraft()
	s.canvas.AssertNoStaging(draft.ID)
}

func (s *canvasStagingSteps) thenNodeIsNotCommittedInCurrentDraft(nodeName string) {
	draft := s.canvas.FindCurrentDraft()
	s.canvas.AssertDraftCommittedLacksNode(draft.ID, nodeName)
}

func (s *canvasStagingSteps) thenNodeIsCommittedInCurrentDraft(nodeName string) {
	draft := s.canvas.FindCurrentDraft()
	s.canvas.AssertDraftCommittedHasNode(draft.ID, nodeName)
}

func (s *canvasStagingSteps) thenNodeIsLive(nodeName string) {
	s.canvas.AssertLiveCanvasHasNode(nodeName)
}

func (s *canvasStagingSteps) thenBothDraftsHaveStaging() {
	s.canvas.AssertDraftCount(2)
	draftOne := s.canvas.FindDraftByDisplayName("Draft #1")
	draftTwo := s.canvas.FindDraftByDisplayName("Draft #2")
	s.canvas.AssertHasStaging(draftOne.ID)
	s.canvas.AssertHasStaging(draftTwo.ID)
}

func (s *canvasStagingSteps) thenNodeIsVisible(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *canvasStagingSteps) thenNodeIsHidden(nodeName string) {
	s.session.AssertHidden(q.TestID("node", nodeName, "header"))
}

func (s *canvasStagingSteps) thenDraftIsCommittedWithoutStaging(displayName, nodeName string) {
	draft := s.canvas.FindDraftByDisplayName(displayName)
	s.canvas.AssertNoStaging(draft.ID)
	s.canvas.AssertDraftCommittedHasNode(draft.ID, nodeName)
}

func (s *canvasStagingSteps) thenDraftStillHasStaging(displayName string) {
	draft := s.canvas.FindDraftByDisplayName(displayName)
	s.canvas.AssertHasStaging(draft.ID)
}
