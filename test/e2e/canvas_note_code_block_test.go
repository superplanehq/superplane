package e2e

import (
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

// A single long line so the rendered code block must scroll horizontally
// instead of wrapping. Intentionally a harmless, obviously-fake sample.
const noteCodeContent = `echo "hello from superplane" && curl https://example.com/api/sample?foo=bar&baz=qux&page=1&limit=100&sort=asc&filter=none # just a sample line`

func TestCanvasNoteCodeBlock(t *testing.T) {
	t.Run("note code block is copyable and horizontally scrollable", func(t *testing.T) {
		steps := &noteCodeBlockSteps{t: t}
		steps.start()
		steps.givenCanvas("E2E Note Code Block")
		steps.enterEditMode()
		steps.addNote()

		steps.startEditingNote()
		steps.fillNote("```\n" + noteCodeContent + "\n```")
		steps.blurNoteEditor()

		steps.assertCopyButtonVisible()
		steps.assertCopyButtonCopiesCode()
		steps.assertCodeBlockScrollsHorizontally()
	})
}

type noteCodeBlockSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *noteCodeBlockSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()

	// Copying uses navigator.clipboard, which requires explicit permission in Chromium.
	require.NoError(s.t, s.session.Page().Context().GrantPermissions(
		[]string{"clipboard-read", "clipboard-write"},
	))
}

func (s *noteCodeBlockSteps) givenCanvas(name string) {
	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *noteCodeBlockSteps) enterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *noteCodeBlockSteps) addNote() {
	s.canvas.AddNote()
}

func (s *noteCodeBlockSteps) startEditingNote() {
	note := q.Text("Double click to add and edit notes...").Run(s.session)
	require.NoError(s.t, note.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}))
	require.NoError(s.t, note.Dblclick(pw.LocatorDblclickOptions{Timeout: pw.Float(10000)}))
	s.session.AssertVisible(q.Locator(`textarea[aria-label="Note note"]`))
}

func (s *noteCodeBlockSteps) fillNote(text string) {
	s.session.FillIn(q.Locator(`textarea[aria-label="Note note"]`), text)
}

func (s *noteCodeBlockSteps) blurNoteEditor() {
	s.canvas.ClickOnEmptyCanvasArea()
	s.session.AssertHidden(q.Locator(`textarea[aria-label="Note note"]`))
}

func (s *noteCodeBlockSteps) assertCopyButtonVisible() {
	s.session.AssertVisible(q.TestID("note-code-copy"))
}

func (s *noteCodeBlockSteps) assertCopyButtonCopiesCode() {
	s.session.Click(q.TestID("note-code-copy"))

	require.Eventually(s.t, func() bool {
		clipboard, err := s.session.Page().Evaluate(`() => navigator.clipboard.readText()`)
		if err != nil {
			return false
		}
		text, ok := clipboard.(string)
		return ok && text == noteCodeContent
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *noteCodeBlockSteps) assertCodeBlockScrollsHorizontally() {
	pre := q.Locator("pre").Run(s.session)
	require.NoError(s.t, pre.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}))

	overflowing, err := pre.Evaluate(`el => el.scrollWidth > el.clientWidth`, nil)
	require.NoError(s.t, err)
	require.Equal(s.t, true, overflowing, "code block should overflow horizontally and scroll")
}
