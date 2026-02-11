package shared

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

type ComponentSteps struct {
	t       *testing.T
	session *session.TestSession

	ComponentName string
}

func NewComponentSteps(name string, t *testing.T, session *session.TestSession) *ComponentSteps {
	return &ComponentSteps{t: t, session: session, ComponentName: name}
}

func (s *ComponentSteps) Create() {
	s.session.VisitHomePage()
	s.session.Click(q.Text("Bundles"))
	s.session.Click(q.Text("New Bundle"))
	s.session.FillIn(q.TestID("component-name-input"), s.ComponentName)
	s.session.Click(q.Text("Create Bundle"))
	s.session.Sleep(300)
}

func (s *ComponentSteps) OpenBuildingBlocksSidebar() {
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

func (s *ComponentSteps) AddNoop(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *ComponentSteps) Save() {
	saveButton := q.TestID("save-canvas-button")
	loc := saveButton.Run(s.session)

	if isVisible, _ := loc.IsVisible(); isVisible {
		s.session.Click(saveButton)
		s.session.Sleep(500)
		return
	}

	// Auto-save may have already persisted the changes.
	s.session.Sleep(500)
}

func (s *ComponentSteps) AddApproval(nodeName string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *ComponentSteps) AddManualTrigger(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	startSource := q.TestID("building-block-start")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(startSource, target, pos.X, pos.Y)
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *ComponentSteps) AddWait(name string, pos models.Position, duration int, unit string) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-wait")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)
	s.session.FillIn(q.TestID("node-name-input"), name)

	valueInput := q.Locator(`label:has-text("How long should I wait?")~div input[type="number"]`)
	s.session.FillIn(valueInput, strconv.Itoa(duration))

	unitTrigger := q.TestID("field-unit-select")
	s.session.Click(unitTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + unit + `")`))

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *ComponentSteps) StartAddingTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *ComponentSteps) AddTimeGate(name string, pos models.Position) {
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

func (s *ComponentSteps) Connect(sourceName, targetName string) {
	sourceHandle := q.Locator(`.react-flow__node:has-text("` + sourceName + `") .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node:has-text("` + targetName + `") .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(300)
}

func (s *ComponentSteps) StartEditingNode(name string) {
	// Click on the node header to open the sidebar where settings can be accessed
	nodeHeader := q.TestID("node", name, "header")
	s.session.Click(nodeHeader)
	s.session.Sleep(300)
}

func (s *ComponentSteps) RunManualTrigger(name string) {
	// Use the Start node's template Run button (in the default payload template) instead of the removed header Run button
	startTemplateRun := q.Locator(`.react-flow__node:has([data-testid="node-` + strings.ToLower(name) + `-header"]) [data-testid="start-template-run"]`)
	s.session.Click(startTemplateRun)
	s.session.Click(q.TestID("emit-event-submit-button"))
}

func (s *ComponentSteps) OpenComponentSettings() {
	// Click the settings button to open the component settings sidebar
	s.session.Click(q.Locator(`button[aria-label="Open settings"]`))
	s.session.Sleep(300)
}

func (s *ComponentSteps) ClickOutputChannelsTab() {
	// Click the "Output Channels" tab
	s.session.Click(q.Text("Output Channels"))
	s.session.Sleep(300)
}

func (s *ComponentSteps) ClickAddConfig() {
	s.session.Click(q.TestID("add-config-field-btn"))
}

func (s *ComponentSteps) AddOutputChannel(channelName, nodeName, nodeOutputChannel string) {
	// Click "Add Output Channel" button to open modal
	s.session.Click(q.Text("Add Output Channel"))
	s.session.Sleep(500)

	// Fill in the output channel name - find input by placeholder
	nameInput := q.Locator(`input[placeholder*="success"]`)
	s.session.FillIn(nameInput, channelName)

	// Select the node from dropdown - look within the dialog
	// The dropdown shows: "NodeName (node-id)"
	nodeSelectButton := q.Locator(`div[role="dialog"] button[role="combobox"]:has-text("Select a node")`)
	s.session.Click(nodeSelectButton)
	s.session.Sleep(300)
	// Match on option that contains the node name
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + nodeName + `")`))
	s.session.Sleep(200)

	// Select the node output channel from dropdown
	channelSelectButton := q.Locator(`div[role="dialog"] div:has(> label:has-text("Node Output Channel")) button[role="combobox"]`)
	s.session.Click(channelSelectButton)
	s.session.Sleep(300)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + nodeOutputChannel + `")`))
	s.session.Sleep(200)

	// Click the save button in the dialog footer
	s.session.Click(q.Locator(`div[role="dialog"] button:has-text("Add Output Channel")`))
	s.session.Sleep(300)
}

func (s *ComponentSteps) AssertOutputChannelExists(channelName, nodeName, nodeOutputChannel string) {
	// Fetch the blueprint from the database
	blueprint, err := models.FindBlueprintByName(s.ComponentName, s.session.OrgID)
	require.NoError(s.t, err, "failed to find blueprint in database")
	require.NotNil(s.t, blueprint, "blueprint not found in database")

	// Find the node by name to get its ID
	var targetNode *models.Node
	for _, node := range blueprint.Nodes {
		if node.Name == nodeName {
			targetNode = &node
			break
		}
	}
	require.NotNil(s.t, targetNode, "node '%s' not found in blueprint", nodeName)

	// Find the output channel with the given name
	var foundChannel *models.BlueprintOutputChannel
	for _, channel := range blueprint.OutputChannels {
		if channel.Name == channelName {
			foundChannel = &channel
			break
		}
	}

	require.NotNil(s.t, foundChannel, "output channel '%s' not found in blueprint", channelName)
	require.Equal(s.t, targetNode.ID, foundChannel.NodeID, "output channel points to wrong node")
	require.Equal(s.t, nodeOutputChannel, foundChannel.NodeOutputChannel, "output channel uses wrong node output channel")
}

func (s *ComponentSteps) AddConfigurationField(fieldName, fieldLabel string) {
	nameInput := q.TestID("config-field-name-input")
	labelInput := q.TestID("config-field-label-input")
	defaultValueInput := q.TestID("config-field-default-value-input")
	saveButton := q.TestID("add-config-field-submit-button")

	s.session.FillIn(nameInput, fieldName)
	s.session.FillIn(labelInput, fieldLabel)
	s.session.FillIn(defaultValueInput, "default")
	s.session.Click(saveButton)
}

func (s *ComponentSteps) AssertConfigurationFieldExists(fieldName, fieldLabel, fieldType string) {
	// Fetch the blueprint from the database
	blueprint, err := models.FindBlueprintByName(s.ComponentName, s.session.OrgID)
	require.NoError(s.t, err, "failed to find blueprint in database")
	require.NotNil(s.t, blueprint, "blueprint not found in database")

	// Find the configuration field with the given name
	var foundField *configuration.Field
	for _, field := range blueprint.Configuration {
		if field.Name == fieldName {
			foundField = &field
			break
		}
	}

	require.NotNil(s.t, foundField, "configuration field '%s' not found in blueprint", fieldName)
	require.Equal(s.t, fieldLabel, foundField.Label, "configuration field label mismatch")
	require.Equal(s.t, fieldType, foundField.Type, "configuration field type mismatch")
}
