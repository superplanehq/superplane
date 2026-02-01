package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestTimeGateComponent(t *testing.T) {
	steps := &TimeGateSteps{t: t}

	weekendDays := []string{"saturday", "sunday"}
	workweekDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday"}

	t.Run("add a TimeGate that blocks on weekends", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists("Weekday Work Hours Gate")
		steps.addTimeGate()
		steps.setDaysTo(weekendDays)
		steps.setTimeWindow("00:00", "23:59")
		steps.setTimezone("0")
		steps.saveTimeGate()
		steps.assertTimeGateSavedToDB("00:00-23:59", "0", weekendDays)
	})

	t.Run("add a TimeGate that blocks on outside of work hours", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists("Work Hours Gate")
		steps.addTimeGate()
		steps.setDaysTo(workweekDays)
		steps.setTimeWindow("09:00", "17:00")
		steps.setTimezone("-5")
		steps.saveTimeGate()
		steps.assertTimeGateSavedToDB("09:00 - 17:00", "-5", workweekDays)
	})

	t.Run("push through the time gate item", func(t *testing.T) {
		steps.start()
		now := time.Now().UTC()
		tomorrow := now.Add(24 * time.Hour)
		activeDay := dayString(tomorrow.Weekday())
		steps.givenACanvasWithManualTriggerTimeGateAndOutput([]string{activeDay}, "00:00-23:59", "0")
		steps.runManualTrigger()
		steps.openSidebarForNode("timeGate")
		steps.pushThroughFirstItemFromSidebar()
		steps.assertTimeGateExecutionFinishedAndOutputNodeProcessed()
	})
}

type TimeGateSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *TimeGateSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *TimeGateSteps) givenACanvasExists(canvasName string) {
	s.canvas = shared.NewCanvasSteps(canvasName, s.t, s.session)
	s.canvas.Create()
}

func (s *TimeGateSteps) addTimeGate() {
	s.canvas.StartAddingTimeGate("timeGate", models.Position{X: 500, Y: 250})
	s.openNodeSettings("timeGate")
	s.session.AssertVisible(q.Locator(`button[aria-label="monday"]`))
}

func (s *TimeGateSteps) setDaysTo(days []string) {
	target := map[string]bool{}
	for _, d := range days {
		target[d] = true
	}

	allDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

	for _, day := range allDays {
		if target[day] {
			continue
		}

		s.session.Click(q.Locator(`button[aria-label="` + day + `"]`))
	}
}

func (s *TimeGateSteps) setTimeWindow(start, end string) {
	startInput := q.TestID("time-field-timeRange-start")
	endInput := q.TestID("time-field-timeRange-end")

	s.session.FillIn(startInput, start)
	s.session.FillIn(endInput, end)
}

func (s *TimeGateSteps) setTimezone(timezone string) {
	timezoneTrigger := q.TestID("field-timezone-select")
	s.session.Click(timezoneTrigger)

	// timezone options in the select use labels like "GMT+0 (London, Dublin, UTC)" or "GMT-5 (New York, Toronto)"
	// we match on the numeric offset prefix
	offsetSelector := `div[role="option"]:has-text("GMT` + timezone + `")`
	if timezone == "0" {
		offsetSelector = `div[role="option"]:has-text("GMT+0 (London, Dublin, UTC)")`
	}

	s.session.Click(q.Locator(offsetSelector))
}

func (s *TimeGateSteps) saveTimeGate() {
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *TimeGateSteps) openNodeSettings(node string) {
	s.canvas.StartEditingNode(node)
	s.session.Click(q.Text("Configuration"))
	s.session.Sleep(200)
}

func (s *TimeGateSteps) assertTimeGateSavedToDB(timeRange, timezoneLabel string, days []string) {
	node := s.canvas.GetNodeFromDB("timeGate")

	assert.Equal(s.t, timeRange, node.Configuration.Data()["timeRange"])
	assert.Equal(s.t, timezoneLabel, node.Configuration.Data()["timezone"])
	assert.Len(s.t, days, len(node.Configuration.Data()["days"].([]interface{})))

	for i, d := range days {
		assert.Equal(s.t, d, node.Configuration.Data()["days"].([]interface{})[i])
	}
}

func (s *TimeGateSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *TimeGateSteps) givenACanvasWithManualTriggerTimeGateAndOutput(days []string, timeRange string, timezone string) {
	s.canvas = shared.NewCanvasSteps("Time Gate Push Through", s.t, s.session)

	s.canvas.Create()
	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddTimeGate("timeGate", models.Position{X: 1000, Y: 250})
	s.canvas.AddNoop("Output", models.Position{X: 1400, Y: 200})

	s.openNodeSettings("timeGate")
	s.setDaysTo(days)
	parts := strings.SplitN(timeRange, "-", 2)
	if len(parts) == 2 {
		s.setTimeWindow(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	s.setTimezone(timezone)
	s.saveTimeGate()

	s.canvas.Connect("Start", "timeGate")
	s.canvas.Connect("timeGate", "Output")

	s.saveCanvas()
}

func (s *TimeGateSteps) runManualTrigger() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecutionInStates(
		"timeGate",
		[]string{models.CanvasNodeExecutionStatePending, models.CanvasNodeExecutionStateStarted},
		10*time.Second,
	)
}

func (s *TimeGateSteps) openSidebarForNode(node string) {
	s.session.Click(q.TestID("node", node, "header"))
}

func (s *TimeGateSteps) pushThroughFirstItemFromSidebar() {
	eventItem := q.Locator("h2:has-text('Latest') ~ div")
	s.session.HoverOver(eventItem)
	s.session.Click(q.Locator("h2:has-text('Latest') ~ div button[aria-label='Open actions']"))
	s.session.Click(q.TestID("push-through-item"))
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 15*time.Second)
}

func (s *TimeGateSteps) assertTimeGateExecutionFinishedAndOutputNodeProcessed() {
	timeGateExecs := s.canvas.GetExecutionsForNode("timeGate")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, timeGateExecs, 1, "expected one execution for time gate node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")

	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, timeGateExecs[0].State)
	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, outputExecs[0].State)
}

func dayString(weekday time.Weekday) string {
	days := map[time.Weekday]string{
		time.Sunday:    "sunday",
		time.Monday:    "monday",
		time.Tuesday:   "tuesday",
		time.Wednesday: "wednesday",
		time.Thursday:  "thursday",
		time.Friday:    "friday",
		time.Saturday:  "saturday",
	}

	return days[weekday]
}
