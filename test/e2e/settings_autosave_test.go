package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestSettingsAutoSave(t *testing.T) {
	t.Run("auto-saves partial configuration when a required field is cleared", func(t *testing.T) {
		steps := &settingsAutoSaveSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Autosave Partial")
		steps.addFilterNode("FilterPartial")
		steps.assertExpressionFieldEquals("FilterPartial", "true")
		steps.clearExpressionField()
		steps.waitForAutoSave()
		steps.assertExpressionFieldEquals("FilterPartial", "")
	})

	t.Run("partial configuration persists after switching to Runs tab", func(t *testing.T) {
		steps := &settingsAutoSaveSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Autosave Tab Switch")
		steps.addFilterNode("FilterSwitch")
		steps.assertExpressionFieldEquals("FilterSwitch", "true")
		steps.clearExpressionField()
		steps.waitForAutoSave()
		steps.switchToRunsTab()
		steps.switchToConfigurationTab()
		steps.assertExpressionFieldEquals("FilterSwitch", "")
	})
}

type settingsAutoSaveSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *settingsAutoSaveSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *settingsAutoSaveSteps) givenACanvasExists(canvasName string) {
	s.canvas = shared.NewCanvasSteps(canvasName, s.t, s.session)
	s.canvas.Create()
}

func (s *settingsAutoSaveSteps) addFilterNode(name string) {
	s.canvas.AddFilter(name, models.Position{X: 500, Y: 250})
}

func (s *settingsAutoSaveSteps) clearExpressionField() {
	expressionInput := q.TestID("expression-field-expression")
	s.session.FillIn(expressionInput, "")
}

func (s *settingsAutoSaveSteps) waitForAutoSave() {
	s.canvas.WaitForCanvasSaveStatusSaved()
	s.session.Sleep(500)
}

func (s *settingsAutoSaveSteps) switchToRunsTab() {
	s.session.Click(q.Text("Runs"))
	s.session.Sleep(500)
}

func (s *settingsAutoSaveSteps) switchToConfigurationTab() {
	s.session.Click(q.Text("Configuration"))
	s.session.Sleep(500)
}

func (s *settingsAutoSaveSteps) assertExpressionFieldEquals(nodeName string, expected string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		node := s.canvas.GetNodeFromDB(nodeName)
		config := node.Configuration.Data()

		val, exists := config["expression"]
		if exists && val == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	node := s.canvas.GetNodeFromDB(nodeName)
	config := node.Configuration.Data()
	val, exists := config["expression"]
	require.True(s.t, exists, "expected expression key to exist in DB config for node %s", nodeName)
	require.Equal(s.t, expected, val, "expected expression=%q in DB for node %s but got %v", expected, nodeName, val)
}
