package agents_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestGetCanvasAgentChat_SerializesSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)
	sessionID := uuid.New()

	svc := &stubService{
		ensureSession: func(_ context.Context, o, u, c uuid.UUID) (*models.AgentSession, error) {
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

	resp, err := actionsagents.GetCanvasAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.GetCanvasAgentChatRequest{CanvasId: canvas.ID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.Chat)
	assert.Equal(t, sessionID.String(), resp.Chat.Id)
	assert.Equal(t, "anthropic", resp.Chat.Provider)
}

func TestGetCanvasAgentChat_RejectsInvalidCanvas(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{}
	_, err := actionsagents.GetCanvasAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.GetCanvasAgentChatRequest{CanvasId: "nope"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestGetCanvasAgentChat_NotFoundWhenCanvasMissing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	svc := &stubService{}
	_, err := actionsagents.GetCanvasAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.GetCanvasAgentChatRequest{CanvasId: uuid.NewString()})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
}

func TestGetCanvasAgentChat_PermissionDenied(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)

	svc := &stubService{
		ensureSession: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error) {
			return nil, agents.ErrSessionForbidden
		},
	}
	_, err := actionsagents.GetCanvasAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.GetCanvasAgentChatRequest{CanvasId: canvas.ID.String()})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
}
