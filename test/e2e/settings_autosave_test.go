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
	s.canvas.EnterEditMode()
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
		draft := s.canvas.FindCurrentDraft()
		if draft != nil && len(draft.Nodes) == 1 {
			return draft.Nodes[0].ID
		}
		time.Sleep(300 * time.Millisecond)
	}

	draft := s.canvas.FindCurrentDraft()
	require.NotNil(s.t, draft, "no draft version found")
	require.Len(s.t, draft.Nodes, 1, "expected exactly one node in draft version after adding filter")
	return draft.Nodes[0].ID
}

func (s *settingsAutoSaveSteps) getExpressionField() (any, bool, bool) {
	if s.nodeID == "" {
		return nil, false, false
	}

	draft := s.canvas.FindCurrentDraft()
	if draft == nil {
		return nil, false, false
	}

	for _, node := range draft.Nodes {
		if node.ID == s.nodeID {
			val, exists := node.Configuration["expression"]
			return val, exists, true
		}
	}

	return nil, false, false
}
