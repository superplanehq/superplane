package e2e

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestCanvasPage(t *testing.T) {
	steps := &CanvasPageSteps{t: t}

	t.Run("adding a node to canvas displays custom node name", func(t *testing.T) {
		steps.Start()
		steps.GivenACanvasExists()
		steps.VisitCanvasPage()
		steps.AddNoopToCanvas("Hello")
		steps.SaveCanvas()
		steps.AssertNodeIsAdded("Hello")
	})

	t.Run("run is disabled when you have unsaved changes", func(t *testing.T) {
		steps.Start()
		steps.GivenACanvasExists()
		steps.VisitCanvasPage()
		steps.AddNoopToCanvas("")
		steps.AssertUnsavedChangesNoteIsVisible()
		steps.AssertCantRunNode()
		steps.AssertExplainationIsShownWhenHoverOverRun()
	})

	t.Run("adding an approval component to canvas and testing add item to list 3 times", func(t *testing.T) {
		steps.Start()
		steps.GivenACanvasExists()
		steps.VisitCanvasPage()
		steps.AddApprovalToCanvas("Test Approval")
		steps.ClickAddItemButton()
		steps.ClickAddItemButton()
		steps.ClickAddItemButton()
	})
}

type CanvasPageSteps struct {
	t          *testing.T
	session    *TestSession
	canvasName string
	workflowID string
}

func (s *CanvasPageSteps) Start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *CanvasPageSteps) GivenACanvasExists() {
	s.canvasName = "E2E Canvas"

	s.session.VisitHomePage()
	s.session.Click(q.Text("New Canvas"))
	s.session.FillIn(q.TestID("canvas-name-input"), s.canvasName)
	s.session.Click(q.Text("Create canvas"))
	s.session.Sleep(300)

	orgUUID := uuid.MustParse(s.session.orgID)
	wf, err := models.FindWorkflowByName(s.canvasName, orgUUID)
	require.NoError(s.t, err)
	s.workflowID = wf.ID.String()
}

func (s *CanvasPageSteps) VisitCanvasPage() {
	s.session.Visit("/" + s.session.orgID + "/workflows/" + s.workflowID)
}

func (s *CanvasPageSteps) AddNoopToCanvas(nodeName string) {
	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 400, 250)
	s.session.Sleep(300)

	// Use default name if empty string provided (node name is required)
	if nodeName == "" {
		nodeName = "noop"
	}

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) AssertUnsavedChangesNoteIsVisible() {
	s.session.AssertText("You have unsaved changes")
}

func (s *CanvasPageSteps) AssertCantRunNode() {
	// The dropdown testID is based on the node name
	// Since we use "noop" as the default name, the testID is "node-noop-header-dropdown"
	dropdown := q.TestID("node-noop-header-dropdown")
	runOption := q.Locator("button:has-text('Run')")

	s.session.Click(dropdown)
	s.session.AssertVisible(runOption)
	s.session.AssertDisabled(runOption)
}

func (s *CanvasPageSteps) AssertExplainationIsShownWhenHoverOverRun() {
	runOption := q.Locator("button:has-text('Run')")

	s.session.HoverOver(runOption)
	s.session.AssertText("Save canvas changes before running")
}

func (s *CanvasPageSteps) SaveCanvas() {
	s.session.Click(q.TestID("save-canvas-button"))
	s.session.Sleep(500)
	s.session.AssertText("Canvas changes saved")
}

func (s *CanvasPageSteps) AddApprovalToCanvas(nodeName string) {
	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 400, 250)
	s.session.Sleep(300)

	// Use default name if empty string provided (node name is required)
	if nodeName == "" {
		nodeName = "approval"
	}

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) ClickAddItemButton() {
	// Click the "Add Item" button to test the list functionality
	s.session.Click(q.Text("Add Item"))
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) AssertNodeIsAdded(nodeName string) {
	// Verify the node displays the custom name, not the generic component label
	s.session.AssertText(nodeName)
}
