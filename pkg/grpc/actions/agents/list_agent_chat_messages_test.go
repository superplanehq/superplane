package agents_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestListAgentChatMessages_RequiresOwnership(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{
		getSession: func(uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	_, err := actionsagents.ListAgentChatMessages(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ListAgentChatMessagesRequest{
		ChatId: uuid.NewString(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
}

func TestListAgentChatMessages_ReturnsMessagesAndForwardsCursor(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	chatID := uuid.New()
	beforeID := uuid.New()

	svc := &stubService{
		getSession: func(_, _, id uuid.UUID) (*models.AgentSession, error) {
			return &models.AgentSession{ID: chatID, CreatedAt: now()}, nil
		},
		listMessages: func(id, before uuid.UUID, limit int) ([]models.AgentSessionMessage, error) {
			assert.Equal(t, chatID, id)
			assert.Equal(t, beforeID, before)
			assert.Equal(t, 25, limit)
			return []models.AgentSessionMessage{
				{ID: uuid.New(), Role: "user", Content: "hi", CreatedAt: now()},
			}, nil
		},
	}
	resp, err := actionsagents.ListAgentChatMessages(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ListAgentChatMessagesRequest{
		ChatId:   chatID.String(),
		BeforeId: beforeID.String(),
		Limit:    25,
	})
	require.NoError(t, err)
	require.Len(t, resp.Messages, 1)
	assert.False(t, resp.HasMore, "fewer rows than limit ⇒ no more pages")
}

func TestListAgentChatMessages_SignalsMorePagesWhenFull(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	chatID := uuid.New()

	svc := &stubService{
		getSession: func(uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error) {
			return &models.AgentSession{ID: chatID, CreatedAt: now()}, nil
		},
		listMessages: func(uuid.UUID, uuid.UUID, int) ([]models.AgentSessionMessage, error) {
			return []models.AgentSessionMessage{
				{ID: uuid.New(), Role: "user", Content: "a", CreatedAt: now()},
				{ID: uuid.New(), Role: "user", Content: "b", CreatedAt: now()},
			}, nil
		},
	}
	resp, err := actionsagents.ListAgentChatMessages(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ListAgentChatMessagesRequest{
		ChatId: chatID.String(),
		Limit:  2,
	})
	require.NoError(t, err)
	assert.True(t, resp.HasMore)
}
