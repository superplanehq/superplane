package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestWaitComponent(t *testing.T) {
	steps := &WaitSteps{t: t}

	t.Run("configure Wait for seconds", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists("Wait Seconds")
		steps.addWaitWithDuration(10, "Seconds")
		steps.saveCanvas()
		steps.assertWaitSavedToDB(10, "seconds")
	})

	t.Run("configure Wait for minutes", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists("Wait Minutes")
		steps.addWaitWithDuration(5, "Minutes")
		steps.saveCanvas()
		steps.assertWaitSavedToDB(5, "minutes")
	})

	t.Run("configure Wait for hours", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists("Wait Hours")
		steps.addWaitWithDuration(2, "Hours")
		steps.saveCanvas()
		steps.assertWaitSavedToDB(2, "hours")
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

func (s *WaitSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *WaitSteps) assertWaitSavedToDB(value int, unit string) {
	node := s.canvas.GetNodeFromDB("Wait")

	duration := node.Configuration.Data()["duration"].(map[string]any)

	assert.Equal(s.t, float64(value), duration["value"])
	assert.Equal(s.t, unit, duration["unit"])
}
