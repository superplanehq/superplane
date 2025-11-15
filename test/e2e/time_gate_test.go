package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		steps.setModeToExcludeRange()
		steps.setDaysTo(weekendDays)
		steps.setTimeWindow("00:00", "23:59")
		steps.setTimezone("0")
		steps.saveTimeGate()
		steps.saveCanvas()
		steps.assertTimeGateSavedToDB("exclude_range", "00:00", "23:59", "0", weekendDays)
	})

	t.Run("add a TimeGate that blocks on outside of work hours", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists("Work Hours Gate")
		steps.addTimeGate()
		steps.setModeToIncludeRange()
		steps.setDaysTo(workweekDays)
		steps.setTimeWindow("09:00", "17:00")
		steps.setTimezone("-5")
		steps.saveTimeGate()
		steps.saveCanvas()
		steps.assertTimeGateSavedToDB("include_range", "09:00", "17:00", "-5", workweekDays)
	})
}

type TimeGateSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps

	currentNodeName string
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
	s.currentNodeName = "Time Gate"

	source := q.TestID("building-block-time_gate")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 500, 250)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), s.currentNodeName)
}

func (s *TimeGateSteps) setModeToIncludeRange() {
	modeTrigger := q.Locator(`label:has-text("Mode") + div button`)
	s.session.Click(modeTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Include Range")`))
}

func (s *TimeGateSteps) setModeToExcludeRange() {
	modeTrigger := q.Locator(`label:has-text("Mode") + div button`)
	s.session.Click(modeTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Exclude Range")`))
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

		s.session.Click(q.TestID("remove", day))
	}
}

func (s *TimeGateSteps) setTimeWindow(start, end string) {
	startInput := q.Locator(`label:has-text("Start Time") + div input[type="time"]`)
	endInput := q.Locator(`label:has-text("End Time") + div input[type="time"]`)

	s.session.FillIn(startInput, start)
	s.session.FillIn(endInput, end)
}

func (s *TimeGateSteps) setTimezone(timezone string) {
	timezoneTrigger := q.Locator(`label:has-text("Timezone") + div button`)
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
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(500)
}

func (s *TimeGateSteps) assertTimeGateSavedToDB(modeLabel, startTime, endTime, timezoneLabel string, days []string) {
	node := s.canvas.GetNodeFromDB("Time Gate")

	assert.Equal(s.t, modeLabel, node.Configuration.Data()["mode"])
	assert.Equal(s.t, startTime, node.Configuration.Data()["startTime"])
	assert.Equal(s.t, endTime, node.Configuration.Data()["endTime"])
	assert.Equal(s.t, timezoneLabel, node.Configuration.Data()["timezone"])
	assert.Len(s.t, days, len(node.Configuration.Data()["days"].([]interface{})))

	for i, d := range days {
		assert.Equal(s.t, d, node.Configuration.Data()["days"].([]interface{})[i])
	}
}

func (s *TimeGateSteps) saveCanvas() {
	s.canvas.Save()
}
