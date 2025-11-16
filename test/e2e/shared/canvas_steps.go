package shared

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

type CanvasSteps struct {
	t       *testing.T
	session *session.TestSession

	CanvasName string
	WorkflowID uuid.UUID
}

func NewCanvasSteps(name string, t *testing.T, session *session.TestSession) *CanvasSteps {
	return &CanvasSteps{t: t, session: session, CanvasName: name}
}

func (s *CanvasSteps) Create() {
	s.session.VisitHomePage()
	s.session.Click(q.Text("New Canvas"))
	s.session.FillIn(q.TestID("canvas-name-input"), s.CanvasName)
	s.session.Click(q.Text("Create canvas"))
	s.session.Sleep(500)

	wf, err := models.FindWorkflowByName(s.CanvasName, s.session.OrgID)
	require.NoError(s.t, err)
	s.WorkflowID = wf.ID
}

func (s *CanvasSteps) Visit() {
	s.session.Visit("/" + s.session.OrgID.String() + "/workflows/" + s.WorkflowID.String())
}

func (s *CanvasSteps) AddNoop(name string, pos models.Position) {
	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *CanvasSteps) Save() {
	s.session.Click(q.TestID("save-canvas-button"))
	s.session.AssertText("Canvas changes saved")
}

func (s *CanvasSteps) AddApproval(nodeName string, pos models.Position) {
	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *CanvasSteps) AddManualTrigger(name string, pos models.Position) {
	startSource := q.TestID("building-block-start")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(startSource, target, pos.X, pos.Y)
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("add-node-button"))
}

func (s *CanvasSteps) AddWait(name string, pos models.Position, duration int, unit string) {
	source := q.TestID("building-block-wait")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)
	s.session.FillIn(q.TestID("node-name-input"), name)

	valueInput := q.Locator(`label:has-text("How long should I wait?") + div input[type="number"]`)
	s.session.FillIn(valueInput, strconv.Itoa(duration))

	unitTrigger := q.Locator(`label:has-text("Unit") + div button`)
	s.session.Click(unitTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + unit + `")`))

	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *CanvasSteps) Connect(sourceName, targetName string) {
	sourceHandle := q.Locator(`.react-flow__node:has-text("` + sourceName + `") .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node:has-text("` + targetName + `") .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(300)
}

func (s *CanvasSteps) StartEditingNode(name string) {
	s.session.Click(q.TestID("node", name, "header-dropdown"))
	s.session.Click(q.TestID("node-action-edit"))
}

func (s *CanvasSteps) GetNodeFromDB(name string) *models.WorkflowNode {
	canvas, err := models.FindWorkflow(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindWorkflowNodes(canvas.ID)
	require.NoError(s.t, err)

	nodeID := ""
	for _, n := range nodes {
		if n.Name == name {
			nodeID = n.NodeID
			break
		}
	}

	if nodeID == "" {
		s.t.Fatalf("node %s not found in database", name)
		return nil
	}

	node, err := models.FindWorkflowNode(database.Conn(), s.WorkflowID, nodeID)
	require.NoError(s.t, err)

	return node
}
