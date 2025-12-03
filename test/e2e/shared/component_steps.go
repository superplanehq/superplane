package shared

import (
	"strconv"
	"testing"

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
	s.session.Click(q.Text("Components"))
	s.session.Click(q.Text("New Component"))
	s.session.FillIn(q.TestID("component-name-input"), s.ComponentName)
	s.session.Click(q.Text("Create Component"))
	s.session.Sleep(300)
}

func (s *ComponentSteps) AddNoop(name string, pos models.Position) {
	source := q.TestID("building-block-noop")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *ComponentSteps) Save() {
	s.session.Click(q.TestID("save-canvas-button"))
	s.session.Sleep(500)
}

func (s *ComponentSteps) AddApproval(nodeName string, pos models.Position) {
	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *ComponentSteps) AddManualTrigger(name string, pos models.Position) {
	startSource := q.TestID("building-block-start")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(startSource, target, pos.X, pos.Y)
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("add-node-button"))
}

func (s *ComponentSteps) AddWait(name string, pos models.Position, duration int, unit string) {
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

func (s *ComponentSteps) StartAddingTimeGate(name string, pos models.Position) {
	source := q.TestID("building-block-time_gate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *ComponentSteps) AddTimeGate(name string, pos models.Position) {
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

func (s *ComponentSteps) Connect(sourceName, targetName string) {
	sourceHandle := q.Locator(`.react-flow__node:has-text("` + sourceName + `") .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node:has-text("` + targetName + `") .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(300)
}

func (s *ComponentSteps) StartEditingNode(name string) {
	s.session.Click(q.TestID("node", name, "header-dropdown"))
	s.session.Click(q.TestID("node-action-edit"))
}

func (s *ComponentSteps) RunManualTrigger(name string) {
	dropdown := q.TestID("node", name, "header-dropdown")
	runOption := q.TestID("node-action-run")

	s.session.Click(dropdown)
	s.session.Click(runOption)
	s.session.Click(q.TestID("emit-event-submit-button"))
}
