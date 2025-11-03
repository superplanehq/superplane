package e2e

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestCustomComponents(t *testing.T) {
	steps := &CustomComponentsSteps{t: t}

	t.Run("expand last run", func(t *testing.T) {
		steps.Start()
		steps.GivenADeploymentComponentExists()
		steps.GivenACanvasWithComponentExists()
		steps.GivenNodeHasOneExecution()
		steps.ClickExpand()
		steps.AssertNavigatedToNodeRunPage()
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

	source1 := q.TestID("building-block-start")
	target1 := q.TestID("rf__wrapper")
	s.session.DragAndDrop(source1, target1, 200, 250)
	s.session.Click(q.TestID("add-node-button"))

	source2 := q.TestID("building-block-e2e-deployment-component")
	target2 := q.TestID("rf__wrapper")
	s.session.DragAndDrop(source2, target2, 600, 250)
	s.session.Click(q.TestID("add-node-button"))

	// connect: drag from Start node's output handle to the component's input handle
	sourceHandle := q.Locator(`.react-flow__node:has-text("start") .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node:has-text("E2E Deployment Component") .react-flow__handle-left`)
	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(200)

	// save canvas
	s.session.Click(q.TestID("save-canvas-button"))
}

func (s *CustomComponentsSteps) GivenNodeHasOneExecution() {
	// Run the Start node instead of the component
	dropdown := q.TestID("node-start-header-dropdown")
	runOption := q.Locator("button:has-text('Run')")

	s.session.Click(dropdown)
	s.session.Click(runOption)
	s.session.Click(q.TestID("emit-event-submit-button"))
	s.session.Sleep(1000)

	// hack to refresh the page
	s.session.Visit("/" + s.session.orgID + "/")
	s.session.Click(q.Text(s.canvasName))
	s.session.Sleep(500)
}

func (s *CustomComponentsSteps) ClickExpand() {
	s.session.Click(q.TestID("expand-run-button"))
}

func (s *CustomComponentsSteps) AssertNavigatedToNodeRunPage() {
	orgUUID := uuid.MustParse(s.session.orgID)
	wf, err := models.FindWorkflowByName(s.canvasName, orgUUID)
	require.NoError(s.t, err)
	s.workflowID = wf.ID.String()

	nodes, err := models.FindWorkflowNodes(wf.ID)
	require.NoError(s.t, err)
	for _, n := range nodes {
		if n.Name == "E2E Deployment Component" {
			s.componentID = n.NodeID
			break
		}
	}

	s.session.AssertURLContains("/workflows/" + s.workflowID + "/nodes/" + s.componentID)
}
