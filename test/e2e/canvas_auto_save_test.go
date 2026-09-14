package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasAutoSave(t *testing.T) {
	t.Run("versioned canvas auto-saves after moving a node", func(t *testing.T) {
		steps := &canvasAutoSaveSteps{t: t}
		steps.start()
		steps.givenCanvas("E2E Auto Save Versioning")
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
		steps.givenCanvas("E2E Auto Save Queue")
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
		steps.waitUntilNodeCenterNear("Queued Move Node", finalCenter)
	})

	t.Run("versioned canvas auto-saves note edits on blur", func(t *testing.T) {
		steps := &canvasAutoSaveSteps{t: t}
		steps.start()
		steps.givenCanvas("E2E Note Auto Save")
		steps.enterEditMode()
		steps.addNote()

		steps.startEditingNoteWithText("Double click to add and edit notes...")
		steps.fillNote("Initial note body")
		steps.blurNoteEditor()
		steps.waitForSaved()
		steps.assertNotePreview("Initial note body")

		updatedText := "Updated note body on blur"
		steps.startEditingNoteWithText("Initial note body")
		steps.fillNote(updatedText)
		steps.blurNoteEditor()
		steps.waitForSaved()
		steps.assertNotePreview(updatedText)
		steps.assertNoteTextInDB(updatedText)
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

func (s *canvasAutoSaveSteps) givenCanvas(name string) {
	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *canvasAutoSaveSteps) enterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *canvasAutoSaveSteps) addNoopNode(name string, pos models.Position) {
	s.canvas.AddNoop(name, pos)
	s.session.AssertText(name)
}

func (s *canvasAutoSaveSteps) addNote() {
	s.canvas.AddNote()
}

func (s *canvasAutoSaveSteps) dismissSidebar() {
	s.canvas.ClickOnEmptyCanvasArea()
	s.session.WaitUntil(func() bool {
		visible, err := q.TestID("node-name-input").Run(s.session).IsVisible()
		return err == nil && !visible
	}, "node sidebar did not close")
}

func (s *canvasAutoSaveSteps) startEditingNoteWithText(text string) {
	note := q.Text(text).Run(s.session)
	err := note.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	})
	require.NoError(s.t, err)
	require.NoError(s.t, note.Dblclick())
	s.session.AssertVisible(q.Locator(`textarea[aria-label="Note note"]`))
}

func (s *canvasAutoSaveSteps) fillNote(text string) {
	s.session.FillIn(q.Locator(`textarea[aria-label="Note note"]`), text)
}

func (s *canvasAutoSaveSteps) blurNoteEditor() {
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasAutoSaveSteps) assertNotePreview(text string) {
	s.session.AssertHidden(q.Locator(`textarea[aria-label="Note note"]`))
	s.session.AssertText(text)
}

func (s *canvasAutoSaveSteps) assertNoteTextInDB(expected string) {
	require.Eventually(s.t, func() bool {
		node, ok := s.canvas.DraftNodeByName("Note")
		if !ok {
			return false
		}
		text, _ := node.Configuration["text"].(string)
		return text == expected
	}, 10*time.Second, 200*time.Millisecond)
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

	require.Eventually(s.t, func() bool {
		moved, err := loc.BoundingBox()
		if err != nil || moved == nil {
			return false
		}
		centerX := moved.X + moved.Width/2
		centerY := moved.Y + moved.Height/2
		return absDelta(centerX, startX) > 5 || absDelta(centerY, startY) > 5
	}, 5*time.Second, 50*time.Millisecond, "node %s did not move", name)
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

// waitForSaved waits until the current user's staged canvas reflects the latest autosave.
func (s *canvasAutoSaveSteps) waitForSaved() {
	s.canvas.WaitForStaging(uuid.Nil)
}

func (s *canvasAutoSaveSteps) waitUntilNodeCenterNear(name string, expected *pw.Rect) {
	require.Eventually(s.t, func() bool {
		center := s.nodeCenter(name)
		return absDelta(center.X, expected.X) <= 2 && absDelta(center.Y, expected.Y) <= 2
	}, 5*time.Second, 100*time.Millisecond, "node %s should stay at the saved position", name)
}

func absDelta(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
