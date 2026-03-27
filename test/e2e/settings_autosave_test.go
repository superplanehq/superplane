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
		steps.assertExpressionFieldMissing("FilterPartial")
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
		steps.assertExpressionInputEquals("")
		steps.assertExpressionFieldMissing("FilterSwitch")
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
		val, exists, found := s.getExpressionField(nodeName)
		if found && exists && val == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, exists, found := s.getExpressionField(nodeName)
	require.True(s.t, found, "expected node %s to exist in DB", nodeName)
	require.True(s.t, exists, "expected expression key to exist in DB config for node %s", nodeName)
	require.Equal(s.t, expected, val, "expected expression=%q in DB for node %s but got %v", expected, nodeName, val)
}

func (s *settingsAutoSaveSteps) assertExpressionFieldMissing(nodeName string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		_, exists, found := s.getExpressionField(nodeName)
		if found && !exists {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, exists, found := s.getExpressionField(nodeName)
	require.True(s.t, found, "expected node %s to exist in DB", nodeName)
	require.False(
		s.t,
		exists,
		"expected expression key to be removed from DB config for node %s but got %v",
		nodeName,
		val,
	)
}

func (s *settingsAutoSaveSteps) assertExpressionInputEquals(expected string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		value, err := q.TestID("expression-field-expression").Run(s.session).InputValue()
		if err == nil && value == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	value, err := q.TestID("expression-field-expression").Run(s.session).InputValue()
	require.NoError(s.t, err)
	require.Equal(s.t, expected, value)
}

func (s *settingsAutoSaveSteps) getExpressionField(nodeName string) (any, bool, bool) {
	canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(s.t, err)

	for _, node := range nodes {
		if node.Name != nodeName {
			continue
		}

		val, exists := node.Configuration.Data()["expression"]
		return val, exists, true
	}

	return nil, false, false
}
