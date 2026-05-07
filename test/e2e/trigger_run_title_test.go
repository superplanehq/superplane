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

func TestTriggerRunTitle(t *testing.T) {
	t.Run("resolves run title expression when event is emitted", func(t *testing.T) {
		steps := &triggerRunTitleSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTrigger("RunTitle Resolve", "Start")

		// Set a run title that references the trigger payload.
		// The manual trigger emits a structured event: {"type": "manual.run", "data": {"message": "Hello, World!"}, "timestamp": "..."}
		steps.whenRunTitleToggleIsEnabled()
		steps.whenRunTitleIsSetTo("Run: {{ root().data.message }}")
		steps.waitForAutoSave()
		steps.thenRunTitleInDBEquals("Run: {{ root().data.message }}")

		// Publish and trigger an event
		steps.saveAndPublish()
		steps.runManualTrigger()

		// The emitted event should have the resolved run title.
		steps.thenEventRunTitleEquals("Run: Hello, World!")
	})
}

type triggerRunTitleSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
	nodeID  string
}

func (s *triggerRunTitleSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *triggerRunTitleSteps) givenACanvasWithManualTrigger(canvasName, triggerName string) {
	s.canvas = shared.NewCanvasSteps(canvasName, s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()
	s.canvas.AddManualTrigger(triggerName, models.Position{X: 500, Y: 250})
	s.nodeID = s.waitForNodeID()
}

func (s *triggerRunTitleSteps) whenRunTitleToggleIsEnabled() {
	runTitleSwitch := q.Locator(`div:has(> label:has-text("Run title")) button[role="switch"]`)
	s.session.Click(runTitleSwitch)
	s.session.Sleep(300)
}

func (s *triggerRunTitleSteps) whenRunTitleIsSetTo(value string) {
	s.session.FillIn(q.TestID("string-field-runtitletemplate"), value)
}

func (s *triggerRunTitleSteps) waitForAutoSave() {
	s.session.Sleep(500)
}

func (s *triggerRunTitleSteps) saveAndPublish() {
	s.canvas.Save()
	s.canvas.Publish()
}

func (s *triggerRunTitleSteps) runManualTrigger() {
	s.canvas.RunManualTrigger("Start")
	s.session.Sleep(2000)
}

func (s *triggerRunTitleSteps) thenRunTitleInDBEquals(expected string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		val, found := s.getRunTitleTemplate()
		if found && val != nil && *val == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, found := s.getRunTitleTemplate()
	require.True(s.t, found, "expected node to exist in DB")
	require.NotNil(s.t, val, "expected run title template to exist in DB")
	require.Equal(s.t, expected, *val, "expected runTitleTemplate=%q in DB but got %v", expected, val)
}

func (s *triggerRunTitleSteps) thenEventRunTitleEquals(expected string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		event := s.findLatestRootEvent()
		if event != nil && event.RunTitle != nil && *event.RunTitle == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	event := s.findLatestRootEvent()
	require.NotNil(s.t, event, "no root event found for canvas")
	require.NotNil(s.t, event.RunTitle, "event run title is nil, expected %q", expected)
	require.Equal(s.t, expected, *event.RunTitle)
}

func (s *triggerRunTitleSteps) findLatestRootEvent() *models.CanvasEvent {
	var event models.CanvasEvent
	err := database.Conn().
		Where("workflow_id = ?", s.canvas.WorkflowID).
		Where("execution_id IS NULL").
		Order("created_at DESC").
		First(&event).Error

	if err != nil {
		return nil
	}
	return &event
}

func (s *triggerRunTitleSteps) waitForNodeID() string {
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
	require.Len(s.t, draft.Nodes, 1, "expected exactly one node in draft")
	return draft.Nodes[0].ID
}

func (s *triggerRunTitleSteps) getRunTitleTemplate() (*string, bool) {
	if s.nodeID == "" {
		return nil, false
	}

	draft := s.canvas.FindCurrentDraft()
	if draft == nil {
		return nil, false
	}

	for _, node := range draft.Nodes {
		if node.ID == s.nodeID {
			return node.RunTitleTemplate, true
		}
	}

	return nil, false
}
