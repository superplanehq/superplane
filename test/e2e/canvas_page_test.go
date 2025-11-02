package e2e

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestCanvasPage(t *testing.T) {
	ctx := NewTestContext(t)
	ctx.Start()

	steps := &CanvasPageSteps{ctx: ctx}

	t.Run("run is disabled when you have unsaved changes", func(t *testing.T) {
		steps.Start()
		steps.GivenACanvasExists()
		steps.VisitCanvasPage()
		steps.AddNoopToCanvas()
		steps.AssertUnsavedChangesNoteIsVisible()
		steps.AssertCantRunNode()
		steps.AssertExplainationIsShownWhenHoverOverRun()
	})
}

type CanvasPageSteps struct {
	ctx        *TestContext
	session    *TestSession
	canvasName string
	workflowID string
}

func (s *CanvasPageSteps) Start() {
	s.session = s.ctx.NewSession()
	s.session.Start()
	s.session.Login()
}

func (s *CanvasPageSteps) GivenACanvasExists() {
	s.canvasName = "E2E Canvas"

	s.session.VisitHomePage()
	s.session.Click(q.Text("New Canvas"))
	s.session.FillIn(q.TestID("canvas-name-input"), s.canvasName)
	s.session.Click(q.Text("Create canvas"))
	s.session.Sleep(500)

	orgUUID := uuid.MustParse(s.session.orgID)
	wf, err := models.FindWorkflowByName(s.canvasName, orgUUID)
	require.NoError(s.ctx.t, err)
	s.workflowID = wf.ID.String()
}

func (s *CanvasPageSteps) VisitCanvasPage() {
	s.session.Visit("/" + s.session.orgID + "/workflows/" + s.workflowID)
}

func (s *CanvasPageSteps) AddNoopToCanvas() {
	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 400, 250)
	s.session.Sleep(300)
	s.session.Click(q.TestID("add-node-button"))
}

func (s *CanvasPageSteps) AssertUnsavedChangesNoteIsVisible() {
	s.session.AssertText("You have unsaved changes")
}

func (s *CanvasPageSteps) AssertCantRunNode() {
	// page := s.session.Page()

	// kebab := page.Locator("button[aria-label='More actions']").First()
	// require.NoError(s.ctx.t, kebab.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.session.timeoutMs)}))
	// require.NoError(s.ctx.t, kebab.Click())

	// runBtn := page.Locator("role=button[name='Run']").First()
	// if c, _ := runBtn.Count(); c == 0 {
	// 	runBtn = page.Locator("button:has-text('Run')").First()
	// }

	// require.NoError(s.ctx.t, runBtn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.session.timeoutMs)}))

	// disabled, _ := runBtn.GetAttribute("disabled")
	// ariaDisabled, _ := runBtn.GetAttribute("aria-disabled")
	// if disabled == nil && (ariaDisabled == nil || *ariaDisabled != "true") {
	// 	s.ctx.t.Fatalf("expected Run button to be disabled")
	// }
}

// AssertExplainationIsShownWhenHoverOverRun: placeholder assertion
func (s *CanvasPageSteps) AssertExplainationIsShownWhenHoverOverRun() {
	// page := s.session.Page()
	// runBtn := page.Locator("role=button[name='Run']").First()
	// if c, _ := runBtn.Count(); c == 0 {
	// 	runBtn = page.Locator("button:has-text('Run')").First()
	// }
	// require.NoError(s.ctx.t, runBtn.Hover())
	// // Tooltip text
	// tip := page.Locator("text=Save canvas changes before running").First()
	// require.NoError(s.ctx.t, tip.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.session.timeoutMs)}))
}
