package e2e

import (
	"strings"
	"testing"
	"time"

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

	t.Run("versioned canvas keeps the latest position after two quick moves", func(t *testing.T) {
		steps := &canvasAutoSaveSteps{t: t}
		steps.start()
		steps.givenCanvasWithVersioningEnabled("E2E Auto Save Queue")
		steps.enterEditMode()
		steps.addNoopNode("Queued Move Node", models.Position{X: 500, Y: 220})
		steps.waitForSaved()
		steps.dismissSidebar()

		initialCenter := steps.nodeCenter("Queued Move Node")
		steps.moveNode("Queued Move Node", 140, 60)
		steps.moveNode("Queued Move Node", 90, 55)
		steps.waitForSaved()

		finalCenter := steps.nodeCenter("Queued Move Node")
		require.Greater(t, finalCenter.X, initialCenter.X+180)
		require.Greater(t, finalCenter.Y, initialCenter.Y+80)

		steps.session.Sleep(1500)

		stableCenter := steps.nodeCenter("Queued Move Node")
		require.InDelta(t, finalCenter.X, stableCenter.X, 2)
		require.InDelta(t, finalCenter.Y, stableCenter.Y, 2)
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
	editButton := q.Locator(`header button:has-text("Edit")`).Run(s.session)
	deadline := time.Now().Add(15 * time.Second)

	for {
		disabled, err := editButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("edit button did not become enabled")
		}

		time.Sleep(200 * time.Millisecond)
	}

	require.NoError(s.t, editButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
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

func (s *canvasAutoSaveSteps) nodeCenter(name string) *pw.Rect {
	loc := nodeHeaderSelector(name).Run(s.session)

	err := loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	})
	require.NoError(s.t, err)

	box, err := loc.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, box)

	return &pw.Rect{
		X:      box.X + box.Width/2,
		Y:      box.Y + box.Height/2,
		Width:  box.Width,
		Height: box.Height,
	}
}

// waitForSaved polls the canvas save status indicator until it reports "saved".
func (s *canvasAutoSaveSteps) waitForSaved() {
	s.canvas.WaitForCanvasSaveStatusSaved()
}
