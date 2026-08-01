package e2e

// -----------------------------------------------------------------------------
// Regression spec for issue #6417 ("Deleted components can still connect and
// be connected to").
//
// On a draft canvas, a node deleted from the draft but still present on the
// live version renders as a "removed" ghost via the draft visual diff. React
// Flow completes connections by handle proximity, so dropping a connection
// drag near the ghost's handle used to create an edge to the deleted node.
// The staged canvas.yaml then referenced a missing node and committing failed
// with the generic "invalid canvas yaml" error.
//
// This spec asserts the fixed behavior: the drop is silently rejected, no
// edge to the deleted node is staged, and the draft commits successfully.
// -----------------------------------------------------------------------------

import (
	"fmt"
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestGhostNodeConnection(t *testing.T) {
	t.Run("a connection dropped on a deleted ghost node is rejected and the draft still commits", func(t *testing.T) {
		steps := &ghostNodeConnectionSteps{t: t}
		steps.start()
		steps.givenACommittedCanvasWithTwoConnectedNodes()
		steps.whenIDeleteTheSecondNodeInTheDraft()
		steps.thenTheDeletedNodeRendersAsAGhost("ghost-node-fixed-1-ghost-visible")
		steps.whenIDragAConnectionFromTheLiveNodeToTheGhostNode()
		steps.thenNoEdgeToTheDeletedNodeIsStaged()
		steps.thenTheDraftCommitsSuccessfully("ghost-node-fixed-4-commit-succeeded")
	})

	t.Run("a connection dragged out of a deleted ghost node output handle cannot stage a ghost edge", func(t *testing.T) {
		steps := &ghostNodeConnectionSteps{t: t}
		steps.start()
		steps.givenACommittedChainWithAMiddleNode()
		steps.whenIDeleteTheSecondNodeInTheDraft()
		steps.thenTheDeletedNodeRendersAsAGhost("ghost-node-fixed-5-output-ghost-visible")
		steps.whenIDragFromTheGhostOutputHandleIntoEmptySpaceAndPickAComponent()
		steps.thenNoEdgeTouchingTheDeletedNodeIsStaged()
		steps.thenTheDraftCommitsSuccessfully("ghost-node-fixed-5-commit-succeeded")
	})
}

type ghostNodeConnectionSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps

	liveNodeID  string // node id of "First" (manual trigger, kept in draft)
	ghostNodeID string // node id of "Second" (noop, deleted from draft)
}

func (s *ghostNodeConnectionSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *ghostNodeConnectionSteps) givenACommittedCanvasWithTwoConnectedNodes() {
	s.canvas = shared.NewCanvasSteps("Ghost Node Connection", s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()

	s.canvas.AddManualTrigger("First", models.Position{X: 500, Y: 200})
	s.canvas.AddNoop("Second", models.Position{X: 1100, Y: 200})
	s.canvas.Connect("First", "Second")
	s.canvas.Save()
	s.canvas.CommitAndPublish()

	// Resolve stable node ids from the committed live version.
	live, err := models.FindLiveCanvasVersion(s.canvas.WorkflowID)
	require.NoError(s.t, err)
	for _, node := range live.Nodes {
		switch node.Name {
		case "First":
			s.liveNodeID = node.ID
		case "Second":
			s.ghostNodeID = node.ID
		}
	}
	require.NotEmpty(s.t, s.liveNodeID, "live version should contain node First")
	require.NotEmpty(s.t, s.ghostNodeID, "live version should contain node Second")
}

func (s *ghostNodeConnectionSteps) givenACommittedChainWithAMiddleNode() {
	s.canvas = shared.NewCanvasSteps("Ghost Node Output Handle", s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()

	s.canvas.AddManualTrigger("First", models.Position{X: 500, Y: 200})
	s.canvas.AddNoop("Second", models.Position{X: 1000, Y: 200})
	s.canvas.AddNoop("Third", models.Position{X: 1500, Y: 200})
	s.canvas.Connect("First", "Second")
	s.canvas.Connect("Second", "Third")
	s.canvas.Save()
	s.canvas.CommitAndPublish()

	// Resolve stable node ids from the committed live version.
	live, err := models.FindLiveCanvasVersion(s.canvas.WorkflowID)
	require.NoError(s.t, err)
	for _, node := range live.Nodes {
		switch node.Name {
		case "First":
			s.liveNodeID = node.ID
		case "Second":
			s.ghostNodeID = node.ID
		}
	}
	require.NotEmpty(s.t, s.liveNodeID, "live version should contain node First")
	require.NotEmpty(s.t, s.ghostNodeID, "live version should contain node Second")
}

func (s *ghostNodeConnectionSteps) whenIDeleteTheSecondNodeInTheDraft() {
	s.canvas.EnterEditMode()

	nodeHeader := q.TestID("node", "Second", "header")
	deleteButton := q.Locator(`.react-flow__node:has([data-testid="node-second-header"]) [data-testid="node-action-delete"]`)
	s.session.HoverOver(nodeHeader)
	s.session.Sleep(100)
	s.session.Click(deleteButton)

	// Wait until the staged draft no longer contains the deleted node.
	require.Eventually(s.t, func() bool {
		nodes, _ := s.canvas.DraftEffectiveSpec()
		for _, node := range nodes {
			if node.ID == s.ghostNodeID {
				return false
			}
		}
		return len(nodes) > 0
	}, 15*time.Second, 200*time.Millisecond, "deleted node should be removed from the staged draft spec")
}

func (s *ghostNodeConnectionSteps) thenTheDeletedNodeRendersAsAGhost(screenshotName string) {
	// Draft visual diff renders the deleted node as a ghost ("REMOVED" badge).
	// Diff X-Ray and "Show deleted nodes" both default to enabled.
	ghostNode := q.Locator(`.react-flow__node[data-id="` + s.ghostNodeID + `"]`)
	s.session.AssertVisible(ghostNode)
	s.session.AssertText("REMOVED")
	s.takeNamedScreenshot(screenshotName)
}

func (s *ghostNodeConnectionSteps) whenIDragAConnectionFromTheLiveNodeToTheGhostNode() {
	// The ghost's own handles are pointer-events:none, so a connection cannot
	// START on the ghost. But React Flow completes connections by handle
	// proximity, so dragging from a live handle and releasing on the ghost's
	// (visually rendered) target handle used to create the connection.
	sourceHandle := q.Locator(`.react-flow__node[data-id="` + s.liveNodeID + `"] .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node[data-id="` + s.ghostNodeID + `"] .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(500)
	s.takeNamedScreenshot("ghost-node-fixed-2-after-drag")
}

func (s *ghostNodeConnectionSteps) thenNoEdgeToTheDeletedNodeIsStaged() {
	require.Never(s.t, func() bool {
		_, edges := s.canvas.DraftEffectiveSpec()
		for _, edge := range edges {
			if edge.SourceID == s.liveNodeID && edge.TargetID == s.ghostNodeID {
				return true
			}
		}
		return false
	}, 5*time.Second, 200*time.Millisecond,
		"the draft must not stage an edge pointing at the deleted (ghost) node")

	s.takeNamedScreenshot("ghost-node-fixed-3-no-ghost-edge")
}

func (s *ghostNodeConnectionSteps) whenIDragFromTheGhostOutputHandleIntoEmptySpaceAndPickAComponent() {
	// The ghost still renders its output handle (it anchors the removed
	// edges), and every handle carries an enlarged ::before hit area. Dragging
	// out of that hit area and dropping in empty space used to open the
	// building-block picker and stage a placeholder edge whose source is the
	// deleted node.
	sourceHandle := q.Locator(`.react-flow__node[data-id="` + s.ghostNodeID + `"] .react-flow__handle-right`)
	emptySpace := q.TestID("rf__wrapper")

	s.session.DragAndDrop(sourceHandle, emptySpace, 500, 550)
	s.session.Sleep(500)
	s.takeNamedScreenshot("ghost-node-fixed-5-after-empty-space-drop")

	// Mirror the user's full path whenever the picker opens (the broken
	// behavior); with the fix the drag never starts, so it never appears.
	sidebar := q.TestID("building-blocks-sidebar").Run(s.session)
	pickerOpened := false
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if visible, _ := sidebar.IsVisible(); visible {
			pickerOpened = true
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	s.t.Logf("building-block picker opened after ghost drag: %v", pickerOpened)

	if pickerOpened {
		s.canvas.OpenBuildingBlockCategory("Debugging")
		s.session.Click(q.TestID("building-block-noop"))
		s.session.Sleep(500)
	}
}

func (s *ghostNodeConnectionSteps) thenNoEdgeTouchingTheDeletedNodeIsStaged() {
	require.Never(s.t, func() bool {
		_, edges := s.canvas.DraftEffectiveSpec()
		for _, edge := range edges {
			if edge.SourceID == s.ghostNodeID || edge.TargetID == s.ghostNodeID {
				return true
			}
		}
		return false
	}, 5*time.Second, 200*time.Millisecond,
		"the draft must not stage an edge referencing the deleted (ghost) node")

	s.takeNamedScreenshot("ghost-node-fixed-5-no-ghost-edge")
}

func (s *ghostNodeConnectionSteps) thenTheDraftCommitsSuccessfully(screenshotName string) {
	// CommitStaging waits for the staged files to clear, which only happens
	// when the API accepts the commit.
	s.canvas.CommitAndPublish()

	content, err := s.session.Page().Content()
	require.NoError(s.t, err)
	require.NotContains(s.t, content, "invalid canvas yaml",
		"committing the draft must not fail with the generic invalid canvas yaml error")

	s.canvas.AssertLiveVersionLacksNode("Second")
	s.takeNamedScreenshot(screenshotName)
}

func (s *ghostNodeConnectionSteps) takeNamedScreenshot(name string) {
	path := fmt.Sprintf("/app/tmp/screenshots/%s.png", name)
	s.t.Logf("Taking screenshot: %s", path)
	if _, err := s.session.Page().Screenshot(pw.PageScreenshotOptions{
		Path:     pw.String(path),
		FullPage: pw.Bool(true),
		Type:     pw.ScreenshotTypePng,
	}); err != nil {
		s.t.Logf("screenshot error: %v", err)
	}
}
