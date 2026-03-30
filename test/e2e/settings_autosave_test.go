package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
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
		steps.assertExpressionFieldCleared("FilterPartial")
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
		steps.assertExpressionFieldCleared("FilterSwitch")
	})
}

type settingsAutoSaveSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
	nodeID  string
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
	s.nodeID = s.waitForSingleNodeID()
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
		val, exists, found := s.getExpressionField()
		if found && exists && val == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, exists, found := s.getExpressionField()
	require.True(s.t, found, "expected node %s to exist in DB", nodeName)
	require.True(s.t, exists, "expected expression key to exist in DB config for node %s", nodeName)
	require.Equal(s.t, expected, val, "expected expression=%q in DB for node %s but got %v", expected, nodeName, val)
}

// assertExpressionFieldCleared verifies the expression field was cleared.
// The ExpressionFieldRenderer converts "" → undefined, so the key is typically
// stripped by filterVisibleConfiguration. We accept both "key absent" and
// "key present with empty string" as valid cleared states.
func (s *settingsAutoSaveSteps) assertExpressionFieldCleared(nodeName string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		val, exists, found := s.getExpressionField()
		if found && (!exists || val == "") {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, exists, found := s.getExpressionField()
	require.True(s.t, found, "expected node %s to exist in DB", nodeName)
	if exists {
		require.Equal(
			s.t,
			"",
			val,
			"expected expression to be cleared for node %s but got %v",
			nodeName,
			val,
		)
	}
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

func (s *settingsAutoSaveSteps) waitForSingleNodeID() string {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)

		nodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(s.t, err)

		if len(nodes) == 1 {
			return nodes[0].NodeID
		}

		time.Sleep(300 * time.Millisecond)
	}

	canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(s.t, err)

	require.Len(s.t, nodes, 1, "expected exactly one node in canvas after adding filter")
	return nodes[0].NodeID
}

func (s *settingsAutoSaveSteps) getExpressionField() (any, bool, bool) {
	if s.nodeID == "" {
		return nil, false, false
	}

	node, err := models.FindCanvasNode(database.Conn(), s.canvas.WorkflowID, s.nodeID)
	if err != nil {
		return nil, false, false
	}

	val, exists := node.Configuration.Data()["expression"]
	return val, exists, true
}
