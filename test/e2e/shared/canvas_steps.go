package shared

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/playwright-community/playwright-go"
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
	s.session.Sleep(500)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(1000)
}

func (s *CanvasSteps) Save() {
	s.session.Click(q.TestID("save-canvas-button"))
	s.session.AssertText("Canvas changes saved")
	s.session.Sleep(500)
}

func (s *CanvasSteps) AddApproval(nodeName string, pos models.Position) {
	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)

	s.session.Click(q.Locator(`button:has-text("Add Item")`))
	s.session.Click(q.Locator(`button:has-text("Select Type")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("User")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

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

func (s *CanvasSteps) StartAddingTimeGate(name string, pos models.Position) {
	source := q.TestID("building-block-time_gate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *CanvasSteps) AddTimeGate(name string, pos models.Position) {
	source := q.TestID("building-block-time_gate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)

	s.session.Click(q.Locator(`label:has-text("Mode") + div button`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Exclude Range")`))

	s.session.FillIn(q.Locator(`label:has-text("Start Time") + div input[type="time"]`), "00:00")
	s.session.FillIn(q.Locator(`label:has-text("End Time") + div input[type="time"]`), "23:59")

	s.session.Click(q.Locator(`label:has-text("Timezone") + div button`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("GMT+0 (London, Dublin, UTC)")`))

	s.session.Click(q.TestID("add-node-button"))
}

func (s *CanvasSteps) Connect(sourceName, targetName string) {
	sourceHandle := q.Locator(`.react-flow__node:has-text("` + sourceName + `") .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node:has-text("` + targetName + `") .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(300)
}

func (s *CanvasSteps) DeleteConnection(sourceName, targetName string) {
	targetHandle := q.Locator(`.react-flow__node:has-text("` + targetName + `") .react-flow__handle-left`)

	loc := targetHandle.Run(s.session)
	box, err := loc.BoundingBox()
	if err != nil || box == nil {
		s.t.Fatalf("getting bounding box for edge %q: %v", loc, err)
	}

	// This is not very precise, but Playwright does not support clicking on SVG paths directly
	// so we click a bit left (20px) from the center of the right handle

	centerX := box.X + box.Width/2 - 20
	centerY := box.Y + box.Height/2

	if err := s.session.Page().Mouse().Click(centerX-20, centerY, pw.MouseClickOptions{}); err != nil {
		s.t.Fatalf("clicking edge %q at center: %v", loc, err)
	}

	deleteButton := q.Locator(`button[aria-label="Delete edge"]`)
	s.session.Click(deleteButton)

	s.session.Sleep(300)
}

func (s *CanvasSteps) StartEditingNode(name string) {
	s.session.Click(q.TestID("node", name, "header-dropdown"))
	s.session.Click(q.TestID("node-action-edit"))
}

func (s *CanvasSteps) RunManualTrigger(name string) {
	dropdown := q.TestID("node", name, "header-dropdown")
	runOption := q.TestID("node-action-run")

	s.session.Click(dropdown)
	s.session.Click(runOption)
	s.session.Click(q.TestID("emit-event-submit-button"))
}

func (s *CanvasSteps) RenameNode(name string, newName string) {
	node := s.GetNodeFromDB(name)

	query := database.Conn().
		Model(&models.WorkflowNode{}).
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Update("name", newName)

	err := query.Error
	require.NoError(s.t, err)
}

func (s *CanvasSteps) GetWorkflowFromDB() *models.Workflow {
	workflow, err := models.FindWorkflow(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	return workflow
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

func (s *CanvasSteps) GetExecutionsForNode(name string) []models.WorkflowNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.WorkflowNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) GetExecutionsForNodeInState(name string, state string) []models.WorkflowNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.WorkflowNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state = ?", state).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) WaitForExecution(name string, state string, timeout time.Duration) {
	found := false
	start := time.Now()

	for time.Since(start) < timeout {
		executions := s.GetExecutionsForNodeInState(name, state)
		if len(executions) > 0 {
			found = true
			break
		}

		s.t.Log("waiting for execution of node", name)
		s.session.Sleep(1000)
	}

	require.True(s.t, found, "timed out waiting for execution of node %s", name)
}
