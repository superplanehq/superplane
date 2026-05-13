package agents_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
)

func TestListAgentChats_ProjectsSessions(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)

	svc := &stubService{
		listSessions: func(_, _, _ uuid.UUID) ([]models.AgentSession, error) {
			return []models.AgentSession{
				{ID: uuid.New(), CanvasID: canvas.ID, Provider: "anthropic", Status: "idle", CreatedAt: now(), UpdatedAt: now()},
				{ID: uuid.New(), CanvasID: canvas.ID, Provider: "anthropic", Status: "streaming", CreatedAt: now(), UpdatedAt: now()},
			}, nil
		},
	}
	resp, err := actionsagents.ListAgentChats(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ListAgentChatsRequest{CanvasId: canvas.ID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Chats, 2)
}
