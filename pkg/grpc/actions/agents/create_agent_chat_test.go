package agents_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateAgentChat_SerializesSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)

	sessionID := uuid.New()
	svc := &stubService{
		createSession: func(ctx context.Context, o, u, c uuid.UUID) (*models.AgentSession, error) {
			assert.Equal(t, r.Organization.ID, o)
			assert.Equal(t, r.User, u)
			assert.Equal(t, canvas.ID, c)
			return &models.AgentSession{
				ID:        sessionID,
				CanvasID:  canvas.ID,
				Provider:  "anthropic",
				Status:    models.AgentSessionStatusIdle,
				CreatedAt: now(),
				UpdatedAt: now(),
			}, nil
		},
	}

	resp, err := actionsagents.CreateAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.CreateAgentChatRequest{CanvasId: canvas.ID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.Chat)
	assert.Equal(t, sessionID.String(), resp.Chat.Id)
	assert.Equal(t, canvas.ID.String(), resp.Chat.CanvasId)
	assert.Equal(t, "anthropic", resp.Chat.Provider)
}

func TestCreateAgentChat_RejectsInvalidCanvas(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{}
	_, err := actionsagents.CreateAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.CreateAgentChatRequest{CanvasId: "not-a-uuid"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateAgentChat_NotFoundWhenCanvasMissing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{}
	_, err := actionsagents.CreateAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.CreateAgentChatRequest{CanvasId: uuid.NewString()})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateAgentChat_PermissionDenied(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)

	svc := &stubService{
		createSession: func(_ context.Context, _, _, _ uuid.UUID) (*models.AgentSession, error) {
			return nil, agents.ErrSessionForbidden
		},
	}
	_, err := actionsagents.CreateAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.CreateAgentChatRequest{CanvasId: canvas.ID.String()})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}
