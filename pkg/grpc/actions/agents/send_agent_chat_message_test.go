package agents_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
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
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestSendAgentChatMessage_ProjectsSuccess(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	chatID := uuid.New()
	persistedID := uuid.New()

	svc := &stubService{
		sendMessage: func(_ context.Context, _, _, sid uuid.UUID, content string, _ []agentservice.MessageImage, options agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			assert.Equal(t, chatID, sid)
			assert.Equal(t, "operator", options.Mode)
			assert.False(t, options.AutoLayoutOnUpdateEnabled)
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
		sendMessage: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []agentservice.MessageImage, agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "x",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
}

func TestSendAgentChatMessage_TranslatesBusySession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{
		sendMessage: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []agentservice.MessageImage, agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			return nil, agentservice.ErrSessionBusy
		},
	}
	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "x",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
}

func TestSendAgentChatMessage_TranslatesInvalidRequest(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{
		sendMessage: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []agentservice.MessageImage, agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			return nil, agentservice.ErrInvalidRequest
		},
	}
	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "x",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestSendAgentChatMessage_MapsBuilderMode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{
		sendMessage: func(_ context.Context, _, _, _ uuid.UUID, _ string, _ []agentservice.MessageImage, options agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			assert.Equal(t, "builder", options.Mode)
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

func TestSendAgentChatMessage_ForwardsAutoLayoutPreference(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{
		sendMessage: func(_ context.Context, _, _, _ uuid.UUID, _ string, _ []agentservice.MessageImage, options agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			assert.True(t, options.AutoLayoutOnUpdateEnabled)
			return &models.AgentSessionMessage{
				ID:        uuid.New(),
				Role:      models.AgentMessageRoleUser,
				Content:   "build it",
				CreatedAt: now(),
			}, nil
		},
	}

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:                    uuid.NewString(),
		Content:                   "build it",
		AutoLayoutOnUpdateEnabled: true,
	})
	require.NoError(t, err)
}

func TestSendAgentChatMessage_ForwardsAndSerializesImages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	var forwarded []agentservice.MessageImage
	svc := &stubService{
		sendMessage: func(_ context.Context, _, _, _ uuid.UUID, content string, images []agentservice.MessageImage, _ agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
			forwarded = images
			return &models.AgentSessionMessage{
				ID:        uuid.New(),
				Role:      models.AgentMessageRoleUser,
				Content:   content,
				Images:    []models.AgentSessionImage{{MediaType: "image/png", Data: "aGVsbG8="}},
				CreatedAt: now(),
			}, nil
		},
	}

	resp, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "",
		Images:  []*pb.AgentChatImage{{MediaType: pb.AgentChatImageMediaType_MEDIA_TYPE_PNG, Data: []byte("hello")}},
	})
	require.NoError(t, err)
	require.Len(t, forwarded, 1)
	assert.Equal(t, "image/png", forwarded[0].MediaType)
	assert.Equal(t, "aGVsbG8=", forwarded[0].Data)
	require.Len(t, resp.Message.Images, 1)
	assert.Equal(t, pb.AgentChatImageMediaType_MEDIA_TYPE_PNG, resp.Message.Images[0].MediaType)
	assert.Empty(t, resp.Message.Images[0].Data)
}

func TestSendAgentChatMessage_RejectsInvalidImages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{}

	cases := []struct {
		name  string
		image *pb.AgentChatImage
	}{
		{"unsupported media type", &pb.AgentChatImage{MediaType: pb.AgentChatImageMediaType_MEDIA_TYPE_UNSPECIFIED, Data: []byte("hello")}},
		{"empty data", &pb.AgentChatImage{MediaType: pb.AgentChatImageMediaType_MEDIA_TYPE_PNG}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
				ChatId: uuid.NewString(),
				Images: []*pb.AgentChatImage{tc.image},
			})
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		})
	}
}

func TestSendAgentChatMessage_RejectsImagesOverPayloadLimit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{}

	big := bytes.Repeat([]byte{0}, 2*1024*1024)
	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
		ChatId: uuid.NewString(),
		Images: []*pb.AgentChatImage{
			{MediaType: pb.AgentChatImageMediaType_MEDIA_TYPE_PNG, Data: big},
			{MediaType: pb.AgentChatImageMediaType_MEDIA_TYPE_PNG, Data: big},
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}
