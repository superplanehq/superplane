package e2e

import (
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasAutoSave(t *testing.T) {
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

func (s *canvasAutoSaveSteps) addNote() {
	s.canvas.AddNote()
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

// waitForSaved waits until the current user's staged canvas reflects the latest autosave.
func (s *canvasAutoSaveSteps) waitForSaved() {
	s.canvas.WaitForStaging(uuid.Nil)
	s.session.Sleep(800)
}
