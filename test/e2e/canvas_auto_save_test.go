package e2e

import (
	"strings"
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasAutoSave(t *testing.T) {
	t.Run("versioned canvas auto-saves after moving a node", func(t *testing.T) {
		steps := &canvasAutoSaveSteps{t: t}
		steps.start()
		steps.givenCanvasWithVersioningEnabled("E2E Auto Save Versioning")
		steps.enterEditMode()
		steps.addNoopNode("Auto Save Node", models.Position{X: 500, Y: 220})
		steps.waitForSaved()
		steps.dismissSidebar()
		steps.moveNode("Auto Save Node", 100, 80)
		steps.waitForSaved()
	})

	t.Run("node position preserved when moved again during autosave", func(t *testing.T) {
		steps := &canvasAutoSaveSteps{t: t}
		steps.start()
		steps.givenCanvasWithVersioningEnabled("E2E Auto Save Race")
		steps.enterEditMode()
		steps.addNoopNode("Race Node", models.Position{X: 300, Y: 200})
		steps.waitForSaved()
		steps.dismissSidebar()

		// First move triggers autosave (100ms debounce).
		// Use moveNodeFast (no trailing sleep) so the second move
		// happens while the first save is likely still in-flight.
		steps.moveNodeFast("Race Node", 100, 0)
		// Second move happens immediately — races with the first save response.
		steps.moveNode("Race Node", 100, 0)

		steps.waitForSaved()

		// The persisted position must reflect BOTH moves.
		steps.assertNodePositionInDB("Race Node", 500, 200)
	})
}

type canvasAutoSaveSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasAutoSaveSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasAutoSaveSteps) givenCanvasWithVersioningEnabled(name string) {
	err := database.Conn().
		Model(&models.Organization{}).
		Where("id = ?", s.session.OrgID).
		Update("versioning_enabled", true).
		Error
	require.NoError(s.t, err)

	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.Locator(`header button:has-text("Edit")`))
}

func (s *canvasAutoSaveSteps) enterEditMode() {
	s.session.Click(q.Locator(`header button:has-text("Edit")`))
	s.session.AssertVisible(q.Locator(`header button:has-text("Propose Change")`))
}

func (s *canvasAutoSaveSteps) addNoopNode(name string, pos models.Position) {
	s.canvas.AddNoop(name, pos)
	s.session.AssertText(name)
}

func (s *canvasAutoSaveSteps) dismissSidebar() {
	s.canvas.ClickOnEmptyCanvasArea()
	s.session.Sleep(300)
}

// nodeHeaderSelector builds the correct data-testid selector for a node header,
// matching the DOM convention of lowercase, space-to-dash conversion.
func nodeHeaderSelector(name string) q.Query {
	safe := strings.ToLower(name)
	safe = strings.ReplaceAll(safe, " ", "-")
	return q.Locator(`[data-testid="node-` + safe + `-header"]`)
}

// moveNode grabs a node by its header and drags it by the given offset.
func (s *canvasAutoSaveSteps) moveNode(name string, deltaX, deltaY int) {
	loc := nodeHeaderSelector(name).Run(s.session)

	err := loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	})
	require.NoError(s.t, err)

	box, err := loc.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, box)

	startX := box.X + box.Width/2
	startY := box.Y + box.Height/2

	require.NoError(s.t, s.session.Page().Mouse().Move(startX, startY))
	require.NoError(s.t, s.session.Page().Mouse().Down())
	require.NoError(s.t, s.session.Page().Mouse().Move(
		startX+float64(deltaX),
		startY+float64(deltaY),
		pw.MouseMoveOptions{Steps: pw.Int(10)},
	))
	require.NoError(s.t, s.session.Page().Mouse().Up())

	s.session.Sleep(300)
}

// moveNodeFast is like moveNode but without the trailing sleep,
// allowing a subsequent move to race with the autosave triggered by this one.
func (s *canvasAutoSaveSteps) moveNodeFast(name string, deltaX, deltaY int) {
	loc := nodeHeaderSelector(name).Run(s.session)

	err := loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	})
	require.NoError(s.t, err)

	box, err := loc.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, box)

	startX := box.X + box.Width/2
	startY := box.Y + box.Height/2

	require.NoError(s.t, s.session.Page().Mouse().Move(startX, startY))
	require.NoError(s.t, s.session.Page().Mouse().Down())
	require.NoError(s.t, s.session.Page().Mouse().Move(
		startX+float64(deltaX),
		startY+float64(deltaY),
		pw.MouseMoveOptions{Steps: pw.Int(10)},
	))
	require.NoError(s.t, s.session.Page().Mouse().Up())
}

// assertNodePositionInDB verifies that the persisted node position matches
// the expected coordinates (draft version when versioning is enabled, live
// version when versioning is disabled).
func (s *canvasAutoSaveSteps) assertNodePositionInDB(name string, expectedX, expectedY int) {
	canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err, "finding canvas")

	organizationVersioningEnabled, err := models.IsCanvasVersioningEnabled(s.session.OrgID)
	require.NoError(s.t, err, "finding organization versioning state")

	isVersioningEnabled := organizationVersioningEnabled || canvas.VersioningEnabled

	var version *models.CanvasVersion
	if isVersioningEnabled {
		draft, draftErr := models.FindCanvasDraftInTransaction(
			database.Conn(), s.canvas.WorkflowID, s.session.UserID,
		)
		require.NoError(s.t, draftErr, "finding user draft")

		version, err = models.FindCanvasVersion(s.canvas.WorkflowID, draft.VersionID)
		require.NoError(s.t, err, "finding canvas version")
	} else {
		version, err = models.FindLiveCanvasVersionInTransaction(database.Conn(), s.canvas.WorkflowID)
		require.NoError(s.t, err, "finding live canvas version")
	}

	var found bool
	for _, node := range version.Nodes {
		if node.Name != name {
			continue
		}
		found = true
		require.Equal(s.t, expectedX, node.Position.X,
			"node %q X position: want %d, got %d", name, expectedX, node.Position.X)
		require.Equal(s.t, expectedY, node.Position.Y,
			"node %q Y position: want %d, got %d", name, expectedY, node.Position.Y)
		break
	}
	require.True(s.t, found, "node %q not found in version nodes", name)
}

// waitForSaved polls the canvas save status indicator until it reports "saved".
func (s *canvasAutoSaveSteps) waitForSaved() {
	s.canvas.WaitForCanvasSaveStatusSaved()
}
