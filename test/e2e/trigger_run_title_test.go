package e2e

import (
	"strings"
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
	testCases := []struct {
		name          string
		titleTemplate string
		expectedTitle string
	}{
		{
			name:          "static title",
			titleTemplate: "Manual run title",
			expectedTitle: "Manual run title",
		},
		{
			name:          "root function",
			titleTemplate: "Run: {{ root().data.message }}",
			expectedTitle: "Run: Hello, World!",
		},
		{
			name:          "previous function",
			titleTemplate: "Run: {{ previous().data.message }}",
			expectedTitle: "Run: Hello, World!",
		},
		{
			name:          "node reference",
			titleTemplate: `Run: {{ $["Start"].data.message }}`,
			expectedTitle: "Run: Hello, World!",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			steps := &triggerRunTitleSteps{t: t}
			steps.start()
			steps.givenACanvasWithManualTrigger("RunTitle "+testCase.name, "Start")

			steps.whenRunTitleToggleIsEnabled()
			steps.whenRunTitleIsSetTo(testCase.titleTemplate)
			steps.waitForAutoSave()
			steps.thenRunTitleInDBEquals(testCase.titleTemplate)

			steps.saveAndPublish()
			steps.runManualTrigger()

			steps.thenEventCustomNameEquals(testCase.expectedTitle)
		})
	}
}

type triggerRunTitleSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
	nodeID  string
	trigger string
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
	s.trigger = triggerName
	s.nodeID = s.waitForNodeID()
}

func (s *triggerRunTitleSteps) whenRunTitleToggleIsEnabled() {
	runTitleSwitch := q.Locator(`div:has(> label:has-text("Run title")) button[role="switch"]`)
	s.session.Click(runTitleSwitch)
	s.session.Sleep(300)
}

func (s *triggerRunTitleSteps) whenRunTitleIsSetTo(value string) {
	s.session.FillIn(q.TestID("string-field-customname"), value)
}

func (s *triggerRunTitleSteps) waitForAutoSave() {
	s.session.Sleep(500)
}

func (s *triggerRunTitleSteps) saveAndPublish() {
	s.canvas.Save()
	s.canvas.CommitAndPublish()
}

func (s *triggerRunTitleSteps) runManualTrigger() {
	s.session.Click(q.Locator(`.react-flow__node:has([data-testid="node-` + strings.ToLower(s.trigger) + `-header"]) [data-testid="start-template-run"]`))
	s.session.Sleep(2000)
}

func (s *triggerRunTitleSteps) thenRunTitleInDBEquals(expected string) {
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

func (s *triggerRunTitleSteps) thenEventCustomNameEquals(expected string) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		event := s.findLatestRootEvent()
		if event != nil && event.CustomName != nil && *event.CustomName == expected {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	event := s.findLatestRootEvent()
	require.NotNil(s.t, event, "no root event found for canvas")
	require.NotNil(s.t, event.CustomName, "event custom name is nil, expected %q", expected)
	require.Equal(s.t, expected, *event.CustomName)
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
		if node, ok := s.canvas.DraftNodeByName(s.trigger); ok {
			return node.ID
		}
		time.Sleep(300 * time.Millisecond)
	}

	require.FailNow(s.t, "expected trigger node in draft")
	return ""
}

func (s *triggerRunTitleSteps) getCustomNameField() (any, bool, bool) {
	if s.nodeID == "" {
		return nil, false, false
	}

	nodes, _ := s.canvas.DraftEffectiveSpec()
	for _, node := range nodes {
		if node.ID == s.nodeID {
			val, exists := node.Configuration["customName"]
			return val, exists, true
		}
	}

	return nil, false, false
}
