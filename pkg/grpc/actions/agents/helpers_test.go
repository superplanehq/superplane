package agents_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

type stubService struct {
	ensureSession func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error)
	resetSession  func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error)
	getSession    func(uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error)
	listMessages  func(uuid.UUID, uuid.UUID, int) ([]models.AgentSessionMessage, error)
	sendMessage   func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []agentservice.MessageImage, string) (*models.AgentSessionMessage, error)
	interruptErr  error
	defineErr     error
}

func (s *stubService) EnsureSession(ctx context.Context, o, u, c uuid.UUID) (*models.AgentSession, error) {
	return s.ensureSession(ctx, o, u, c)
}
func (s *stubService) ResetSession(ctx context.Context, o, u, c uuid.UUID) (*models.AgentSession, error) {
	return s.resetSession(ctx, o, u, c)
}
func (s *stubService) GetSession(o, u, id uuid.UUID) (*models.AgentSession, error) {
	return s.getSession(o, u, id)
}
func (s *stubService) ListMessages(id, before uuid.UUID, limit int) ([]models.AgentSessionMessage, error) {
	return s.listMessages(id, before, limit)
}
func (s *stubService) SendMessage(ctx context.Context, o, u, id uuid.UUID, content string, images []agentservice.MessageImage, mode ...string) (*models.AgentSessionMessage, error) {
	selectedMode := ""
	if len(mode) > 0 {
		selectedMode = mode[0]
	}
	return s.sendMessage(ctx, o, u, id, content, images, selectedMode)
}

func (s *stubService) InterruptSession(ctx context.Context, o, u, id uuid.UUID) error {
	return s.interruptErr
}

func (s *stubService) DefineOutcome(ctx context.Context, o, u, id uuid.UUID, description, rubric string, maxIterations int) error {
	return s.defineErr
}

func now() *time.Time {
	t := time.Now()
	return &t
}

func setupCanvas(t *testing.T, r *support.ResourceRegistry) *models.Canvas {
	t.Helper()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	return canvas
}
