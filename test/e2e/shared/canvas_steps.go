package shared

import (
	"strconv"
	"strings"
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

	wf, err := models.FindCanvasByName(s.CanvasName, s.session.OrgID)
	require.NoError(s.t, err)
	s.WorkflowID = wf.ID
}

func (s *CanvasSteps) Visit() {
	s.session.Visit("/" + s.session.OrgID.String() + "/canvases/" + s.WorkflowID.String())
}

func (s *CanvasSteps) OpenBuildingBlocksSidebar() {
	// Try to open the sidebar if it's not already open
	// The button only appears when sidebar is closed
	openButton := q.TestID("open-sidebar-button")
	loc := openButton.Run(s.session)

	// Check if the button is visible (sidebar is closed)
	if isVisible, _ := loc.IsVisible(); isVisible {
		s.session.Click(openButton)
		s.session.Sleep(300)
	}
}

func (s *CanvasSteps) AddNoop(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(500)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(1000)
}

// AddNoopWithDefaultName adds a noop node using the auto-generated name and returns that name.
func (s *CanvasSteps) AddNoopWithDefaultName(pos models.Position) string {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(500)

	// Get the auto-generated name from the input field
	nameInput := q.TestID("node-name-input")
	loc := nameInput.Run(s.session)
	generatedName, err := loc.InputValue()
	require.NoError(s.t, err)

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(1000)

	return generatedName
}

func (s *CanvasSteps) Save() {
	saveButton := q.TestID("save-canvas-button")
	loc := saveButton.Run(s.session)

	if isVisible, _ := loc.IsVisible(); isVisible {
		s.session.Click(saveButton)
		s.session.AssertText("Canvas changes saved")
		s.session.Sleep(500)
		return
	}

	// Auto-save may have already persisted the changes.
	s.session.Sleep(500)
}

func (s *CanvasSteps) AddApproval(nodeName string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)

	s.session.Click(q.TestID("field-type-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.Click(q.TestID("save-node-button"))

	s.session.Sleep(500)
}

func (s *CanvasSteps) AddManualTrigger(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	startSource := q.TestID("building-block-start")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(startSource, target, pos.X, pos.Y)
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *CanvasSteps) AddWait(name string, pos models.Position, duration int, unit string) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-wait")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)
	s.session.FillIn(q.TestID("node-name-input"), name)

	modeSelector := q.TestID("field-mode-select")
	s.session.Click(modeSelector)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Interval")`))

	valueInput := q.Locator("textarea[data-testid='string-field-waitfor']")
	s.session.FillIn(valueInput, strconv.Itoa(duration))

	unitTrigger := q.TestID("field-unit-select")
	s.session.Click(unitTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + unit + `")`))

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *CanvasSteps) AddFilter(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-filter")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.FillIn(q.TestID("expression-field-expression"), "true")
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *CanvasSteps) StartAddingTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *CanvasSteps) AddTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.FillIn(q.TestID("time-field-timerange-start"), "00:00")
	s.session.FillIn(q.TestID("time-field-timerange-end"), "23:59")

	s.session.Click(q.TestID("field-timezone-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("GMT+0 (London, Dublin, UTC)")`))

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
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

	// Click on the edge to delete it (edges now delete on click instead of requiring a separate delete button)
	// Click a bit left (40px) from the center of the target handle to hit the edge

	centerX := box.X + box.Width/2 - 40
	centerY := box.Y + box.Height/2

	if err := s.session.Page().Mouse().Click(centerX, centerY, pw.MouseClickOptions{}); err != nil {
		s.t.Fatalf("clicking edge %q at center: %v", loc, err)
	}

	s.session.Sleep(300)
}

func (s *CanvasSteps) StartEditingNode(name string) {
	// Click on the node header to open the sidebar where settings can be accessed
	nodeHeader := q.TestID("node", name, "header")
	s.session.Click(nodeHeader)
	s.session.Sleep(300)
}

func (s *CanvasSteps) RunManualTrigger(name string) {
	// Use the Start node's template Run button (in the default payload template) instead of the removed header Run button
	startTemplateRun := q.Locator(`.react-flow__node:has([data-testid="node-` + strings.ToLower(name) + `-header"]) [data-testid="start-template-run"]`)
	s.session.Click(startTemplateRun)
	s.session.Click(q.TestID("emit-event-submit-button"))
}

func (s *CanvasSteps) RenameNode(name string, newName string) {
	node := s.GetNodeFromDB(name)

	query := database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Update("name", newName)

	err := query.Error
	require.NoError(s.t, err)
}

func (s *CanvasSteps) GetWorkflowFromDB() *models.Canvas {
	workflow, err := models.FindCanvas(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	return workflow
}

func (s *CanvasSteps) GetNodeFromDB(name string) *models.CanvasNode {
	canvas, err := models.FindCanvas(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
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

	node, err := models.FindCanvasNode(database.Conn(), s.WorkflowID, nodeID)
	require.NoError(s.t, err)

	return node
}

func (s *CanvasSteps) GetExecutionsForNode(name string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) GetExecutionsForNodeInState(name string, state string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state = ?", state).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) GetExecutionsForNodeInStates(name string, states []string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state IN ?", states).
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

func (s *CanvasSteps) WaitForExecutionInStates(name string, states []string, timeout time.Duration) {
	found := false
	start := time.Now()

	for time.Since(start) < timeout {
		executions := s.GetExecutionsForNodeInStates(name, states)
		if len(executions) > 0 {
			found = true
			break
		}

		s.t.Log("waiting for execution of node", name)
		s.session.Sleep(1000)
	}

	require.True(s.t, found, "timed out waiting for execution of node %s", name)
}
