package e2e

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestWaitComponent(t *testing.T) {
	t.Run("configure Wait for seconds", func(t *testing.T) {
		steps := &WaitSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Wait Seconds")
		steps.addWaitWithDuration(10, "Seconds")
		steps.assertWaitSavedToDB(10, "seconds")
	})

	t.Run("configure Wait for minutes", func(t *testing.T) {
		steps := &WaitSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Wait Minutes")
		steps.addWaitWithDuration(5, "Minutes")
		steps.assertWaitSavedToDB(5, "minutes")
	})

	t.Run("configure Wait for hours", func(t *testing.T) {
		steps := &WaitSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Wait Hours")
		steps.addWaitWithDuration(2, "Hours")
		steps.assertWaitSavedToDB(2, "hours")
	})

	t.Run("push through the wait item", func(t *testing.T) {
		steps := &WaitSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerWaitAndOutput()
		steps.runManualTrigger()
		steps.openSidebarForNode("Wait")
		steps.pushThroughFirstItemFromSidebar()
		steps.assertWaitExecutionFinishedAndOutputNodeProcessed()
	})
}

type WaitSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps

	currentNodeName string
}

func (s *WaitSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *WaitSteps) givenACanvasExists(canvasName string) {
	s.canvas = shared.NewCanvasSteps(canvasName, s.t, s.session)
	s.canvas.Create()
}

func (s *WaitSteps) addWaitWithDuration(value int, unit string) {
	s.currentNodeName = "Wait"
	s.canvas.AddWait(s.currentNodeName, models.Position{X: 500, Y: 250}, value, unit)
}

func (s *WaitSteps) assertWaitSavedToDB(value int, unit string) {
	node := s.canvas.GetNodeFromDB("Wait")

	config := node.Configuration.Data()

	assert.Equal(s.t, "interval", config["mode"])
	assert.Equal(s.t, strconv.Itoa(value), config["waitFor"])
	assert.Equal(s.t, unit, config["unit"])
}

func (s *WaitSteps) givenACanvasWithManualTriggerWaitAndOutput() {
	s.canvas = shared.NewCanvasSteps("Wait Push Through", s.t, s.session)

	s.canvas.Create()
	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddWait("Wait", models.Position{X: 1000, Y: 200}, 10, "Seconds")
	s.canvas.AddNoop("Output", models.Position{X: 1400, Y: 200})

	s.canvas.Connect("Start", "Wait")
	s.canvas.Connect("Wait", "Output")

	s.canvas.Save()
}

func (s *WaitSteps) runManualTrigger() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecutionInStates(
		"Wait",
		[]string{
			models.CanvasNodeExecutionStatePending,
			models.CanvasNodeExecutionStateStarted,
		},
		10*time.Second,
	)
}

func (s *WaitSteps) openSidebarForNode(node string) {
	header := q.TestID("node", node, "header")
	s.session.AssertVisible(header)
	s.session.Click(header)
}

func (s *WaitSteps) pushThroughFirstItemFromSidebar() {
	eventItem := q.Locator(`[data-testid="sidebar-event-item"][data-event-state="running"]`)
	s.session.HoverOver(eventItem)
	s.session.Sleep(300) // Wait for hover to register and actions button to appear
	s.session.Click(q.Locator(`[data-testid="sidebar-event-item"][data-event-state="running"] button[aria-label="Open actions"]`))
	s.session.Sleep(300) // Wait for actions menu to open
	s.session.Click(q.TestID("push-through-item"))
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 15*time.Second)
}

func (s *WaitSteps) assertWaitExecutionFinishedAndOutputNodeProcessed() {
	waitExecs := s.canvas.GetExecutionsForNode("Wait")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, waitExecs, 1, "expected one execution for wait node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")

	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, waitExecs[0].State)
	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, outputExecs[0].State)
}
