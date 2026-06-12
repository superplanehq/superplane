package shared

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"

	"github.com/superplanehq/superplane/test/e2e/queries"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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

// EnterEditMode clicks the Edit button in the header to create or continue a draft version.
// This must be called before making any canvas changes.
func (s *CanvasSteps) EnterEditMode() {
	s.waitForEnabledEditButton()
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	require.NoError(s.t, editButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(400)

	menu := q.TestID("start-editing-menu").Run(s.session)
	if visible, _ := menu.IsVisible(); visible {
		continueButton := q.TestID("start-editing-continue").Run(s.session)
		require.NoError(s.t, continueButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	}

	s.session.Sleep(500)
	s.waitForEnabledExitEditButton()
}

// CreateNewDraftFromEditMenu opens the Edit menu and creates an additional draft branch.
func (s *CanvasSteps) CreateNewDraftFromEditMenu() {
	s.waitForEnabledEditButton()
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	require.NoError(s.t, editButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertVisible(q.TestID("start-editing-menu"))

	createButton := q.TestID("start-editing-create").Run(s.session)
	require.NoError(s.t, createButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(500)
	s.waitForEnabledExitEditButton()
}

// ExitEditMode leaves the current draft and returns to the live canvas view.
func (s *CanvasSteps) ExitEditMode() {
	s.waitForEnabledExitEditButton()
	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	require.NoError(s.t, exitEditButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
	s.session.Sleep(500)
}

// OpenVersionsSidebar opens the Versions view via the canvas header tab.
func (s *CanvasSteps) OpenVersionsSidebar() {
	s.session.Click(q.TestID("canvas-view-mode-versions"))
	s.session.AssertVisible(q.TestID("canvas-versions-sidebar"))
	s.session.AssertVisible(q.Locator(`[data-testid="canvas-view-mode-versions"][aria-current="page"]`))
	s.session.Sleep(300)
}

func (s *CanvasSteps) waitForToolSidebarOpen() {
	deadline := time.Now().Add(15 * time.Second)
	sidebar := q.TestID("canvas-tool-sidebar").Run(s.session)
	openButton := q.TestID("canvas-tool-sidebar-toggle").Run(s.session)

	for time.Now().Before(deadline) {
		visible, err := sidebar.IsVisible()
		require.NoError(s.t, err)
		if visible {
			return
		}

		visible, err = openButton.IsVisible()
		require.NoError(s.t, err)
		if visible {
			err = openButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(1000)})
			if err == nil {
				continue
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	s.session.AssertVisible(q.TestID("canvas-tool-sidebar"))
}

// OpenDraftBranchInSidebar selects a draft branch from the Versions sidebar by display name.
func (s *CanvasSteps) OpenDraftBranchInSidebar(displayName string) {
	s.OpenVersionsSidebar()
	selector := q.Locator(fmt.Sprintf(`[data-testid="canvas-draft-branch-row"]:has-text("%s") button`, displayName))
	s.session.Click(selector)
	s.session.Sleep(500)
	s.waitForEnabledExitEditButton()
}

// ListDraftVersions returns all draft versions for this canvas, newest first.
func (s *CanvasSteps) ListDraftVersions() []models.CanvasVersion {
	drafts, err := models.ListDraftCanvasVersions(s.WorkflowID)
	require.NoError(s.t, err)
	return drafts
}

// AssertDraftCount waits until the canvas has the expected number of draft branches.
func (s *CanvasSteps) AssertDraftCount(expected int) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		if len(s.ListDraftVersions()) == expected {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("expected %d draft branches, got %d", expected, len(s.ListDraftVersions()))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// AssertDraftBranchesInSidebar verifies draft branch labels appear in the Versions sidebar.
func (s *CanvasSteps) AssertDraftBranchesInSidebar(displayNames ...string) {
	s.OpenVersionsSidebar()
	s.session.AssertVisible(q.TestID("canvas-drafts-section"))
	for _, displayName := range displayNames {
		s.session.AssertVisible(q.Locator(fmt.Sprintf(`[data-testid="canvas-drafts-section"] :text-is("%s")`, displayName)))
	}
}

func (s *CanvasSteps) waitForEnabledEditButton() {
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	deadline := time.Now().Add(15 * time.Second)
	for {
		disabled, err := editButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("edit button did not become enabled")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *CanvasSteps) waitForEnabledExitEditButton() {
	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	deadline := time.Now().Add(15 * time.Second)
	for {
		disabled, err := exitEditButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("exit edit button did not become enabled")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Publish clicks the Publish button in the header to publish the current draft version.
// This should be called after making and saving canvas changes.
func (s *CanvasSteps) Publish() {
	publishButton := q.Locator(`header button:has-text("Publish")`).Run(s.session)

	deadline := time.Now().Add(15 * time.Second)
	for {
		disabled, err := publishButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("publish button did not become enabled")
		}
		time.Sleep(200 * time.Millisecond)
	}

	require.NoError(s.t, publishButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(1000)
}

// FindCurrentDraft returns the most recently created draft version for this canvas, or nil if none exists.
func (s *CanvasSteps) FindCurrentDraft() *models.CanvasVersion {
	drafts, err := models.ListDraftCanvasVersions(s.WorkflowID)
	require.NoError(s.t, err)
	if len(drafts) == 0 {
		return nil
	}

	return &drafts[0]
}

func (s *CanvasSteps) Create() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, nil, nil)
	s.WorkflowID = canvas.ID

	err = database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.WorkflowID).
		Update("name", s.CanvasName).Error
	require.NoError(s.t, err)

	s.Visit()
}

func (s *CanvasSteps) CreatePublishedWithParameterizedManualRun() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	startNodeID := "start-trigger"
	outputNodeID := "noop-output"

	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: startNodeID,
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"templates": []any{
					map[string]any{
						"name": "Hello World",
						"payload": map[string]any{
							"message": `{{ parameters["message"] }}`,
						},
						"parameters": []any{
							map[string]any{
								"name": "message",
								"type": "string",
							},
						},
					},
				},
			}),
			Position: datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID: outputNodeID,
			Name:   "Output",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
	}, []models.Edge{
		{SourceID: startNodeID, TargetID: outputNodeID, Channel: "default"},
	})
	s.WorkflowID = canvas.ID

	err = database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.WorkflowID).
		Update("name", s.CanvasName).Error
	require.NoError(s.t, err)

	s.Visit()
}

func (s *CanvasSteps) Visit() {
	s.session.Visit("/" + s.session.OrgID.String() + "/apps/" + s.WorkflowID.String())
}

func (s *CanvasSteps) OpenBuildingBlocksSidebar() {
	sidebar := q.TestID("building-blocks-sidebar").Run(s.session)
	if isVisible, _ := sidebar.IsVisible(); isVisible {
		return
	}

	editButton := q.TestID("canvas-edit-button").Run(s.session)
	addComponentButton := q.TestID("canvas-add-component-button").Run(s.session)
	openButton := q.TestID("open-sidebar-button").Run(s.session)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if isVisible, _ := sidebar.IsVisible(); isVisible {
			return
		}

		// Newer canvas UI keeps the component sidebar open after selecting a node.
		// Deselecting the node reveals the floating Components button again.
		s.ClickOnEmptyCanvasArea()
		s.session.Sleep(150)

		if isVisible, _ := sidebar.IsVisible(); isVisible {
			return
		}

		if isVisible, _ := editButton.IsVisible(); isVisible {
			if err := editButton.Click(); err == nil {
				s.session.Sleep(250)
			}
		}

		if isVisible, _ := addComponentButton.IsVisible(); isVisible {
			if err := addComponentButton.Click(); err == nil {
				s.session.Sleep(250)
			}
		}

		if isVisible, _ := openButton.IsVisible(); isVisible {
			if err := openButton.Click(); err == nil {
				s.session.Sleep(250)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	s.session.AssertVisible(q.TestID("building-blocks-sidebar"))
}

func (s *CanvasSteps) OpenBuildingBlockCategory(categoryName string) {
	s.OpenBuildingBlocksSidebar()

	details := q.Locator(fmt.Sprintf(
		`[data-testid="building-blocks-sidebar"] details:has(summary :text-is("%s"))`,
		categoryName,
	)).Run(s.session)

	open, err := details.GetAttribute("open")
	require.NoError(s.t, err)
	if open != "" {
		return
	}

	s.session.Click(q.Locator(fmt.Sprintf(
		`[data-testid="building-blocks-sidebar"] details:has(summary :text-is("%s")) summary`,
		categoryName,
	)))
	s.session.Sleep(200)
}

// ClickOnEmptyCanvasArea clicks on an empty area of the canvas to dismiss
// any open sidebars and deselect all nodes.
func (s *CanvasSteps) ClickOnEmptyCanvasArea() {
	target := q.TestID("rf__wrapper")
	el := target.Run(s.session)
	box, _ := el.BoundingBox()
	if box != nil {
		_ = s.session.Page().Mouse().Click(box.X+600, box.Y+50)
	}
}

// SelectAllNodes performs a rubber-band drag selection across the entire visible
// canvas area to select all nodes. The sidebar must be closed before calling this.
func (s *CanvasSteps) SelectAllNodes() {
	target := q.TestID("rf__wrapper")
	s.session.DragSelectOnCanvas(target, 10, 10, 1100, 700)
}

func (s *CanvasSteps) AddNoop(name string, pos models.Position) {
	s.OpenBuildingBlockCategory("Debugging")

	source := q.TestID("building-block-noop")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(500)

	s.selectLatestNoopNode()
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Sleep(300)
	s.waitForDraftNodeID(name)
}

func (s *CanvasSteps) AddNote() {
	// The "Add Note" button only appears in the closed building blocks sidebar.
	// If the sidebar is currently open, close it first by clicking on empty canvas area.
	sidebar := q.TestID("building-blocks-sidebar").Run(s.session)
	if isVisible, _ := sidebar.IsVisible(); isVisible {
		s.ClickOnEmptyCanvasArea()
		s.session.Sleep(300)
	}

	s.session.Click(q.TestID("add-note-button"))
	require.Eventually(s.t, func() bool {
		draft := s.FindCurrentDraft()
		if draft == nil {
			return false
		}
		for _, node := range draft.Nodes {
			if node.Name == "Note" {
				return true
			}
		}
		return false
	}, 10*time.Second, 200*time.Millisecond)
	s.session.AssertVisible(q.Text("Double click to add and edit notes..."))
	s.session.Sleep(300)
}

// AddNoopWithDefaultName adds a noop node using the auto-generated name and returns that name.
func (s *CanvasSteps) AddNoopWithDefaultName(pos models.Position) string {
	s.OpenBuildingBlockCategory("Debugging")

	source := q.TestID("building-block-noop")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(500)

	s.selectLatestNoopNode()

	// Get the auto-generated name from the input field
	nameInput := q.TestID("node-name-input")
	loc := nameInput.Run(s.session)
	generatedName, err := loc.InputValue()
	require.NoError(s.t, err)

	s.session.Sleep(300)

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

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddApproval(nodeName string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-approval")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-approval")

	s.session.FillIn(q.TestID("node-name-input"), nodeName)

	s.session.Click(q.TestID("field-type-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddManualTrigger(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	startSource := q.TestID("building-block-start")
	s.addBlockFromSidebar(startSource, pos)
	s.openComponentSidebarForLatestBlock("building-block-start")
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Sleep(300)
}

func (s *CanvasSteps) AddWait(name string, pos models.Position, duration int, unit string) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-wait")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-wait")
	s.session.FillIn(q.TestID("node-name-input"), name)

	modeSelector := q.TestID("field-mode-select")
	s.session.Click(modeSelector)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Interval")`))

	valueInput := q.Locator("textarea[data-testid='string-field-waitfor']")
	s.session.FillIn(valueInput, strconv.Itoa(duration))

	unitTrigger := q.TestID("field-unit-select")
	s.session.Click(unitTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + unit + `")`))

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddFilter(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-filter")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-filter")
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.FillIn(q.TestID("expression-field-expression"), "true")
	s.session.Sleep(300)
}

func (s *CanvasSteps) StartAddingTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-timeGate")

	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *CanvasSteps) AddTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-timeGate")

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.FillIn(q.TestID("time-field-timerange-start"), "00:00")
	s.session.FillIn(q.TestID("time-field-timerange-end"), "23:59")

	s.session.Click(q.TestID("field-timezone-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("GMT+0 (London, Dublin, UTC)")`))

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddBuildingBlockByTestID(blockTestID string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()
	s.addBlockFromSidebar(q.TestID(blockTestID), pos)
	s.session.Sleep(500)
	s.openComponentSidebarForLatestBlock(blockTestID)
}

func (s *CanvasSteps) addBlockFromSidebar(source queries.Query, pos models.Position) {
	target := q.TestID("rf__wrapper")
	s.session.DragAndDrop(source, target, pos.X, pos.Y)
}

func (s *CanvasSteps) openComponentSidebarForLatestBlock(blockTestID string) {
	slug := strings.ToLower(strings.TrimPrefix(blockTestID, "building-block-"))
	headers := s.session.Page().Locator(fmt.Sprintf(
		`.react-flow__node [data-testid^="node-%s"][data-testid$="-header"]`,
		slug,
	))
	count, err := headers.Count()
	require.NoError(s.t, err)
	require.Greater(s.t, count, 0, "expected at least one %s node after dropping block", slug)

	require.NoError(s.t, headers.Nth(count-1).Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(150)
}

func (s *CanvasSteps) selectLatestNoopNode() {
	s.openComponentSidebarForLatestBlock("building-block-noop")
}

func (s *CanvasSteps) Connect(sourceName, targetName string) {
	sourceNodeID := s.waitForDraftNodeID(sourceName)
	targetNodeID := s.waitForDraftNodeID(targetName)

	sourceHandle := q.Locator(`.react-flow__node[data-id="` + sourceNodeID + `"] .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node[data-id="` + targetNodeID + `"] .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(300)
}

func (s *CanvasSteps) waitForDraftNodeID(nodeName string) string {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		for _, draft := range s.ListDraftVersions() {
			for _, node := range draft.Nodes {
				if node.Name == nodeName {
					return node.ID
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	s.t.Fatalf("node %q not found in any draft branch", nodeName)
	return ""
}

func (s *CanvasSteps) DeleteConnection(sourceName, targetName string) {
	sourceNodeID := s.waitForDraftNodeID(sourceName)
	targetNodeID := s.waitForDraftNodeID(targetName)

	edge := q.Locator(`.react-flow__edge`).Run(s.session)
	require.Eventually(s.t, func() bool {
		count, err := edge.Count()
		return err == nil && count > 0
	}, 10*time.Second, 200*time.Millisecond)

	// The edge midpoint lies on the source node's right handle and the target
	// node's left handle line. Computing it from the handle positions gives a
	// point that is reliably on the (mostly horizontal) edge path. Playwright's
	// Locator.Hover()/Click() target an element's bounding-box center, which is
	// unreliable for an SVG path: the geometric center of the bounding box can
	// fall off the actual stroke, so the action never lands on the edge.
	sourceHandle := q.Locator(`.react-flow__node[data-id="` + sourceNodeID + `"] .react-flow__handle-right`).Run(s.session)
	targetHandle := q.Locator(`.react-flow__node[data-id="` + targetNodeID + `"] .react-flow__handle-left`).Run(s.session)

	sourceBox, err := sourceHandle.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, sourceBox)
	targetBox, err := targetHandle.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, targetBox)

	midX := (sourceBox.X + sourceBox.Width/2 + targetBox.X + targetBox.Width/2) / 2
	midY := (sourceBox.Y + sourceBox.Height/2 + targetBox.Y + targetBox.Height/2) / 2

	// In edit mode the wide transparent delete hit-area path is always present
	// (canDelete = isEditMode && !isReadOnly), so a hover is not required to
	// reveal it. Move the mouse onto the edge to set the hovered state, then
	// dispatch a raw click at the same on-edge point. Using raw mouse events
	// avoids the unreliable element-center actionability checks.
	hitArea := q.Locator(`.react-flow__renderer [data-testid="edge-delete-hit-area"]`).Run(s.session)
	require.Eventually(s.t, func() bool {
		count, err := hitArea.Count()
		return err == nil && count > 0
	}, 10*time.Second, 200*time.Millisecond)

	mouse := s.session.Page().Mouse()
	require.NoError(s.t, mouse.Move(midX, midY))
	s.session.Sleep(300)
	require.NoError(s.t, mouse.Click(midX, midY))
	s.session.Sleep(500)
	s.waitForDraftEdgeCount(0)
}

func (s *CanvasSteps) waitForDraftEdgeCount(expected int) {
	require.Eventually(s.t, func() bool {
		draft := s.FindCurrentDraft()
		if draft == nil {
			return false
		}
		return len(draft.Edges) == expected
	}, 10*time.Second, 200*time.Millisecond, "draft edge count to reach %d", expected)
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

func (s *CanvasSteps) RunParameterizedManualTrigger(name string, parameters map[string]string) {
	startTemplateRun := q.Locator(`.react-flow__node:has([data-testid="node-` + strings.ToLower(name) + `-header"]) [data-testid="start-template-run"]`)
	s.session.Click(startTemplateRun)

	for paramName, value := range parameters {
		s.session.FillIn(q.Locator("#start-run-param-"+paramName), value)
	}

	s.session.Click(q.TestID("emit-event-submit-button"))
}

func (s *CanvasSteps) EmitManualTrigger(name string) {
	node := s.GetNodeFromDB(name)
	context := contexts.NewEventContext(database.Conn(), node, func(events []models.CanvasEvent) {
		for i := range events {
			require.NoError(s.t, messages.PublishCanvasEventCreatedMessage(&events[i]))
		}
	})

	require.NoError(s.t, context.Emit("manual.run", map[string]any{"message": "Hello, World!"}))
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
