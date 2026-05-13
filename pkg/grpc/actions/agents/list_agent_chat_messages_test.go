package agents_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestListAgentChatMessages_RequiresOwnership(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{
		getSession: func(_, _, _ uuid.UUID) (*models.AgentSession, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	_, err := actionsagents.ListAgentChatMessages(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ListAgentChatMessagesRequest{
		ChatId: uuid.NewString(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestListAgentChatMessages_ReturnsMessages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)
	chatID := uuid.New()

	svc := &stubService{
		getSession: func(_, _, id uuid.UUID) (*models.AgentSession, error) {
			assert.Equal(t, chatID, id)
			return &models.AgentSession{ID: chatID, CanvasID: canvas.ID, CreatedAt: now()}, nil
		},
		listMessages: func(id uuid.UUID) ([]models.AgentSessionMessage, error) {
			assert.Equal(t, chatID, id)
			return []models.AgentSessionMessage{
				{ID: uuid.New(), Role: "user", Content: "hi", CreatedAt: now()},
				{ID: uuid.New(), Role: "assistant", Content: "hey", CreatedAt: now()},
			}, nil
		},
	}
	resp, err := actionsagents.ListAgentChatMessages(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ListAgentChatMessagesRequest{
		ChatId: chatID.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Messages, 2)
	assert.Equal(t, "user", resp.Messages[0].Role)
	assert.Equal(t, "assistant", resp.Messages[1].Role)
}
