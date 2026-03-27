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
		steps.clearExpressionField()
		steps.waitForAutoSave()
		steps.assertExpressionFieldSavedAsEmpty("FilterPartial")
	})

	t.Run("partial configuration persists after switching to Runs tab", func(t *testing.T) {
		steps := &settingsAutoSaveSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Autosave Tab Switch")
		steps.addFilterNode("FilterSwitch")
		steps.clearExpressionField()
		steps.waitForAutoSave()
		steps.switchToRunsTab()
		steps.switchToConfigurationTab()
		steps.assertExpressionFieldSavedAsEmpty("FilterSwitch")
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

func (s *settingsAutoSaveSteps) assertExpressionFieldSavedAsEmpty(nodeName string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		node := s.canvas.GetNodeFromDB(nodeName)
		config := node.Configuration.Data()

		val, exists := config["expression"]
		if exists && val == "" {
			return
		}
		if !exists {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	node := s.canvas.GetNodeFromDB(nodeName)
	config := node.Configuration.Data()
	val, exists := config["expression"]
	if !exists {
		return
	}
	require.Equal(s.t, "", val, "expected expression to be empty in DB but got %v", val)
}
