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
)

func TestResetCanvasAgentChat_ProvisionsFreshSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)
	newSessionID := uuid.New()

	svc := &stubService{
		resetSession: func(_ context.Context, o, u, c uuid.UUID) (*models.AgentSession, error) {
			assert.Equal(t, r.Organization.ID, o)
			assert.Equal(t, r.User, u)
			assert.Equal(t, canvas.ID, c)
			return &models.AgentSession{
				ID:        newSessionID,
				CanvasID:  canvas.ID,
				Provider:  "anthropic",
				Status:    models.AgentSessionStatusIdle,
				CreatedAt: now(),
				UpdatedAt: now(),
			}, nil
		},
	}

	resp, err := actionsagents.ResetCanvasAgentChat(
		context.Background(),
		svc,
		r.Organization.ID.String(),
		r.User.String(),
		&pb.ResetCanvasAgentChatRequest{CanvasId: canvas.ID.String()},
	)
	require.NoError(t, err)
	require.NotNil(t, resp.Chat)
	assert.Equal(t, newSessionID.String(), resp.Chat.Id)
	assert.Equal(t, canvas.ID.String(), resp.Chat.CanvasId)
	assert.Equal(t, "anthropic", resp.Chat.Provider)
}
