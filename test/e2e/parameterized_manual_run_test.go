package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestParameterizedManualRun(t *testing.T) {
	t.Run("creates and runs a parameterized manual run", func(t *testing.T) {
		steps := &parameterizedManualRunSteps{t: t}
		steps.start()
		steps.givenACanvasWithParameterizedManualTrigger()
		steps.whenIRunWithParameter("Hello from E2E")
		steps.thenTheRunCompletedWithMessage("Hello from E2E")
	})
}

type parameterizedManualRunSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *parameterizedManualRunSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *parameterizedManualRunSteps) givenACanvasWithParameterizedManualTrigger() {
	s.canvas = shared.NewCanvasSteps("Parameterized Manual Run", s.t, s.session)
	s.canvas.CreatePublishedWithParameterizedManualRun()
}

func (s *parameterizedManualRunSteps) whenIRunWithParameter(message string) {
	s.canvas.RunParameterizedManualTrigger("Start", map[string]string{
		"message": message,
	})
}

func (s *parameterizedManualRunSteps) thenTheRunCompletedWithMessage(expected string) {
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 30*time.Second)

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		event := s.findLatestRootEvent()
		if event != nil {
			message, ok := rootEventMessage(event)
			if ok && message == expected {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	event := s.findLatestRootEvent()
	require.NotNil(s.t, event, "no root event found for canvas")
	message, ok := rootEventMessage(event)
	require.True(s.t, ok, "expected root event payload to include message")
	require.Equal(s.t, expected, message)
}

func (s *parameterizedManualRunSteps) findLatestRootEvent() *models.CanvasEvent {
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

func rootEventMessage(event *models.CanvasEvent) (string, bool) {
	data, ok := event.Data.Data().(map[string]any)
	if !ok {
		return "", false
	}

	inner, ok := data["data"].(map[string]any)
	if !ok {
		return "", false
	}

	message, ok := inner["message"].(string)
	return message, ok
}
