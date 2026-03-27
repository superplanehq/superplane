package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestSettingsAutoSave(t *testing.T) {
	t.Run("auto-saves partial configuration when a required field is cleared", func(t *testing.T) {
		steps := &settingsAutoSaveSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Autosave Partial")
		steps.addWaitNodeWithName("WaitPartial")
		steps.clearWaitForField()
		steps.waitForAutoSave()
		steps.assertWaitForFieldSavedAsEmpty("WaitPartial")
	})

	t.Run("partial configuration persists after switching to Runs tab", func(t *testing.T) {
		steps := &settingsAutoSaveSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Autosave Tab Switch")
		steps.addWaitNodeWithName("WaitSwitch")
		steps.clearWaitForField()
		steps.waitForAutoSave()
		steps.switchToRunsTab()
		steps.switchToConfigurationTab()
		steps.assertWaitForFieldSavedAsEmpty("WaitSwitch")
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

func (s *settingsAutoSaveSteps) addWaitNodeWithName(name string) {
	s.canvas.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-wait")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 500, 250)
	s.session.Sleep(500)
	s.session.FillIn(q.TestID("node-name-input"), name)

	s.canvas.WaitForCanvasSaveStatusSaved()
	s.session.Sleep(300)
}

func (s *settingsAutoSaveSteps) clearWaitForField() {
	valueInput := q.Locator("textarea[data-testid='string-field-waitfor']")
	s.session.FillIn(valueInput, "")
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

func (s *settingsAutoSaveSteps) assertWaitForFieldSavedAsEmpty(nodeName string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		node := s.canvas.GetNodeFromDB(nodeName)
		config := node.Configuration.Data()

		val, exists := config["waitFor"]
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
	val, exists := config["waitFor"]
	if !exists {
		return
	}
	require.Equal(s.t, "", val, "expected waitFor to be empty in DB but got %v", val)
}
