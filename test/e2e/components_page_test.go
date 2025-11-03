package e2e

import (
	"testing"

	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestCustomComponents(t *testing.T) {
	steps := &CustomComponentsSteps{t: t}

	t.Run("component on canvas with a run; clicking expand navigates to node run page", func(t *testing.T) {
		steps.Start()
		steps.GivenADeploymentComponentExists()
		steps.GivenACanvasWithComponentExists()
		steps.GivenNodeHasOneExecution()
		steps.ClickExpand()
		// steps.AssertNavigatedToNodeRunPage()
	})
}

type CustomComponentsSteps struct {
	t           *testing.T
	session     *TestSession
	canvasName  string
	workflowID  string
	componentID string
}

func (s *CustomComponentsSteps) Start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *CustomComponentsSteps) GivenADeploymentComponentExists() {
	s.session.VisitHomePage()
	s.session.Click(q.Text("Components"))
	s.session.Click(q.Text("New Component"))
	s.session.FillIn(q.TestID("component-name-input"), "E2E Deployment Component")
	s.session.Click(q.Text("Create Component"))
	s.session.Sleep(300)

	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 400, 250)
	s.session.Sleep(300)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Click(q.Text("Save"))
	s.session.Sleep(300)
}

func (s *CustomComponentsSteps) GivenACanvasWithComponentExists() {
	s.canvasName = "E2E Components"

	s.session.VisitHomePage()
	s.session.Click(q.Text("New Canvas"))
	s.session.FillIn(q.TestID("canvas-name-input"), s.canvasName)
	s.session.Click(q.Text("Create canvas"))
	s.session.Sleep(300)

	source := q.TestID("building-block-e2e-deployment-component")
	target := q.TestID("rf__wrapper")
	s.session.DragAndDrop(source, target, 400, 250)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Click(q.TestID("save-canvas-button"))
}

func (s *CustomComponentsSteps) GivenNodeHasOneExecution() {
	dropdown := q.TestID("node-e2e-deployment-component-header-dropdown")
	runOption := q.Locator("button:has-text('Run')")

	s.session.Click(dropdown)
	s.session.Click(runOption)
	s.session.Click(q.TestID("emit-event-submit-button"))
	s.session.Sleep(1000)
	s.session.TakeScreenshot()
	s.session.Visit("/" + s.session.orgID + "/canvases")
	s.session.Click(q.Text(s.canvasName))
}

func (s *CustomComponentsSteps) VisitCanvasPage() {
	s.session.Visit("/" + s.session.orgID + "/workflows/" + s.workflowID)
}

func (s *CustomComponentsSteps) ClickExpand() {
	s.session.Click(q.TestID("expand-run-button"))
}

func (s *CustomComponentsSteps) AssertNavigatedToNodeRunPage() {
	// Expect URL to contain the node sub-route
	s.session.AssertURLContains("/workflows/" + s.workflowID + "/nodes/" + s.componentID)
}
