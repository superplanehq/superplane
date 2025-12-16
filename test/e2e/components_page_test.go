package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
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

	t.Run("setting up outputs", func(t *testing.T) {
		steps.Start()
		steps.StartCreatingComponent()
		steps.AddTwoNodesAndConnect()
		steps.SetUpOutputs()
		steps.SaveComponent()
		steps.AssertComponentHasOutputs()
	})

	t.Run("set up configuration options", func(t *testing.T) {
		steps.Start()
		steps.StartCreatingComponent()
		steps.AddTwoNodesAndConnect()
		steps.SetUpConfigurationOptions()
		steps.SaveComponent()
		steps.AssertComponentHasConfiguration()
	})
}

type CustomComponentsSteps struct {
	t       *testing.T
	session *session.TestSession

	canvas    *shared.CanvasSteps
	component *shared.ComponentSteps
}

func (s *CustomComponentsSteps) Start() {
	s.session = ctx.NewSession(s.t)
	s.canvas = shared.NewCanvasSteps("E2E Components", s.t, s.session)
	s.component = shared.NewComponentSteps("E2E Deployment Component", s.t, s.session)

	s.session.Start()
	s.session.Login()
}

func (s *CustomComponentsSteps) StartCreatingComponent() {
	s.component.Create()
}

func (s *CustomComponentsSteps) AddTwoNodesAndConnect() {
	s.component.AddNoop("First", models.Position{X: 200, Y: 250})
	s.component.AddNoop("Second", models.Position{X: 600, Y: 250})
	s.component.Connect("First", "Second")
}

func (s *CustomComponentsSteps) SetUpOutputs() {
	s.component.OpenComponentSettings()
	s.component.ClickOutputChannelsTab()
	s.component.AddOutputChannel("success", "Second", "default")
}

func (s *CustomComponentsSteps) SaveComponent() {
	s.component.Save()
}

func (s *CustomComponentsSteps) SetUpConfigurationOptions() {
	s.component.OpenComponentSettings()
	s.component.ClickAddConfig()
	s.component.AddConfigurationField("environment", "Environment")
}

func (s *CustomComponentsSteps) AssertComponentHasConfiguration() {
	s.component.AssertConfigurationFieldExists("environment", "Environment", "string")
}

func (s *CustomComponentsSteps) AssertComponentHasOutputs() {
	s.component.AssertOutputChannelExists("success", "Second", "default")
}

func (s *CustomComponentsSteps) GivenADeploymentComponentExists() {
	s.component.Create()
	s.component.AddNoop("Prepare", models.Position{X: 200, Y: 250})
	s.component.AddNoop("Deploy", models.Position{X: 600, Y: 250})
	s.component.Connect("Prepare", "Deploy")
	s.component.Save()
}

func (s *CustomComponentsSteps) GivenACanvasWithComponentExists() {
	s.canvas.Create()
	s.canvas.Visit()
	s.canvas.AddManualTrigger("Start", models.Position{X: 500, Y: 250})

	source2 := q.TestID("building-block-e2e-deployment-component")
	target2 := q.TestID("rf__wrapper")
	s.session.DragAndDrop(source2, target2, 900, 250)
	s.session.Click(q.TestID("add-node-button"))

	s.canvas.Connect("Start", "E2E Deployment Component")
	s.canvas.Save()
}

func (s *CustomComponentsSteps) GivenNodeHasOneExecution() {
	s.canvas.RunManualTrigger("Start")
	s.session.Sleep(1000)

	s.canvas.Visit()
	s.session.Sleep(500)
}

func (s *CustomComponentsSteps) ClickExpand() {
	s.session.Click(q.TestID("expand-run-button"))
}

func (s *CustomComponentsSteps) AssertNavigatedToNodeRunPage() {
	node := s.canvas.GetNodeFromDB("E2E Deployment Component")
	require.NotNil(s.t, node, "component node not found in DB")

	s.session.AssertURLContains("/workflows/" + s.canvas.WorkflowID.String() + "/nodes/" + node.NodeID)
}
