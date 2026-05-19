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

func TestSendAgentChatMessage_RequiresContent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{}
	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSendAgentChatMessage_ProjectsSuccess(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	chatID := uuid.New()
	persistedID := uuid.New()

	svc := &stubService{
		sendMessage: func(_ context.Context, _, _, sid uuid.UUID, content string, mode string) (*models.AgentSessionMessage, error) {
			assert.Equal(t, chatID, sid)
			assert.Equal(t, "operator", mode)
			return &models.AgentSessionMessage{
				ID:        persistedID,
				Role:      models.AgentMessageRoleUser,
				Content:   content,
				CreatedAt: now(),
			}, nil
		},
	}
	resp, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  chatID.String(),
		Content: "hi",
	})
	require.NoError(t, err)
	assert.Equal(t, persistedID.String(), resp.Message.Id)
	assert.Equal(t, "hi", resp.Message.Content)
}

func TestSendAgentChatMessage_TranslatesNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{
		sendMessage: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (*models.AgentSessionMessage, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "x",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestSendAgentChatMessage_MapsBuilderMode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{
		sendMessage: func(_ context.Context, _, _, _ uuid.UUID, _ string, mode string) (*models.AgentSessionMessage, error) {
			assert.Equal(t, "builder", mode)
			return &models.AgentSessionMessage{
				ID:        uuid.New(),
				Role:      models.AgentMessageRoleUser,
				Content:   "build it",
				CreatedAt: now(),
			}, nil
		},
	}

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "build it",
		Mode:    pb.AgentMode_MODE_BUILDER,
	})
	require.NoError(t, err)
}
