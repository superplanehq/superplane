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

func TestTriggerRunTitle(t *testing.T) {
	t.Run("sets run title on a manual trigger and persists it", func(t *testing.T) {
		steps := &triggerRunTitleSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTrigger("RunTitle Canvas", "MyTrigger")
		steps.whenRunTitleToggleIsEnabled()
		steps.whenRunTitleIsSetTo("Deploy {{ root().data.version }}")
		steps.waitForAutoSave()
		steps.thenRunTitleInDBEquals("MyTrigger", "Deploy {{ root().data.version }}")
	})

	t.Run("clearing run title removes it from configuration", func(t *testing.T) {
		steps := &triggerRunTitleSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTrigger("RunTitle Clear", "ClearTrigger")
		steps.whenRunTitleToggleIsEnabled()
		steps.whenRunTitleIsSetTo("some title")
		steps.waitForAutoSave()
		steps.thenRunTitleInDBEquals("ClearTrigger", "some title")
		steps.whenRunTitleIsSetTo("")
		steps.waitForAutoSave()
		steps.thenRunTitleIsAbsentInDB("ClearTrigger")
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
	// The Run title field is togglable — click the switch next to "Run title" label to enable it.
	runTitleSwitch := q.Locator(`div:has(> label:has-text("Run title")) button[role="switch"]`)
	s.session.Click(runTitleSwitch)
	s.session.Sleep(300)
}

func (s *triggerRunTitleSteps) whenRunTitleIsSetTo(value string) {
	runTitleInput := q.TestID("string-field-customname")
	s.session.FillIn(runTitleInput, value)
}

func (s *triggerRunTitleSteps) waitForAutoSave() {
	s.session.Sleep(500)
}

func (s *triggerRunTitleSteps) thenRunTitleInDBEquals(nodeName, expected string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		val, exists, found := s.getCustomNameField()
		if found && exists && val == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, exists, found := s.getCustomNameField()
	require.True(s.t, found, "expected node to exist in DB")
	require.True(s.t, exists, "expected customName key to exist in DB config")
	require.Equal(s.t, expected, val, "expected customName=%q in DB but got %v", expected, val)
}

func (s *triggerRunTitleSteps) thenRunTitleIsAbsentInDB(nodeName string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		val, exists, found := s.getCustomNameField()
		if found && (!exists || val == "") {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	val, exists, found := s.getCustomNameField()
	require.True(s.t, found, "expected node to exist in DB")
	if exists {
		require.Equal(s.t, "", val, "expected customName to be absent or empty but got %v", val)
	}
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

func (s *triggerRunTitleSteps) getCustomNameField() (any, bool, bool) {
	if s.nodeID == "" {
		return nil, false, false
	}

	draft := s.canvas.FindCurrentDraft()
	if draft == nil {
		return nil, false, false
	}

	for _, node := range draft.Nodes {
		if node.ID == s.nodeID {
			val, exists := node.Configuration["customName"]
			return val, exists, true
		}
	}

	return nil, false, false
}
