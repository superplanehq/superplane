package agents_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestAgentActionsRequirePaymentMethod(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas := setupCanvas(t, r)
	chatID := uuid.NewString()
	missingCard := func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []agentservice.MessageImage, agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
		return nil, billingmodels.ErrPaymentMethodRequired
	}

	t.Run("load chat", func(t *testing.T) {
		svc := &stubService{
			ensureSession: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error) {
				return nil, billingmodels.ErrPaymentMethodRequired
			},
		}
		_, err := actionsagents.GetCanvasAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.GetCanvasAgentChatRequest{
			CanvasId: canvas.ID.String(),
		})
		requirePaymentMethod(t, err)
	})

	t.Run("reset chat", func(t *testing.T) {
		svc := &stubService{
			resetSession: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*models.AgentSession, error) {
				return nil, billingmodels.ErrPaymentMethodRequired
			},
		}
		_, err := actionsagents.ResetCanvasAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.ResetCanvasAgentChatRequest{
			CanvasId: canvas.ID.String(),
		})
		requirePaymentMethod(t, err)
	})

	t.Run("send message", func(t *testing.T) {
		_, err := actionsagents.SendAgentChatMessage(context.Background(), &stubService{sendMessage: missingCard}, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
			ChatId:  chatID,
			Content: "hello",
		})
		requirePaymentMethod(t, err)
	})

	t.Run("define outcome", func(t *testing.T) {
		_, err := actionsagents.DefineAgentOutcome(context.Background(), &stubService{defineErr: billingmodels.ErrPaymentMethodRequired}, r.Organization.ID.String(), r.User.String(), &pb.DefineAgentOutcomeRequest{
			ChatId: chatID,
			Rubric: "The change is complete",
		})
		requirePaymentMethod(t, err)
	})
}

func requirePaymentMethod(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	message, ok := grpcerrors.HandlerMessage(err)
	require.True(t, ok)
	assert.Equal(t, billingmodels.ErrPaymentMethodRequired.Error(), message)
}

func TestAgentActionsRequireActivePlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	chatID := uuid.NewString()

	t.Run("send message", func(t *testing.T) {
		svc := &stubService{
			sendMessage: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []agentservice.MessageImage, agentservice.SendMessageRequestOptions) (*models.AgentSessionMessage, error) {
				return nil, billingmodels.ErrHostedSubscriptionRequired
			},
		}
		_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.SendAgentChatMessageRequest{
			ChatId:  chatID,
			Content: "hello",
		})
		requireActivePlan(t, err)
	})

	t.Run("define outcome", func(t *testing.T) {
		svc := &stubService{defineErr: billingmodels.ErrHostedSubscriptionRequired}
		_, err := actionsagents.DefineAgentOutcome(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.DefineAgentOutcomeRequest{
			ChatId: chatID,
			Rubric: "The change is complete",
		})
		requireActivePlan(t, err)
	})
}

func requireActivePlan(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	message, ok := grpcerrors.HandlerMessage(err)
	require.True(t, ok)
	assert.Equal(t, billingmodels.ErrHostedSubscriptionRequired.Error(), message)
}
