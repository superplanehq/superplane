package agents_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeProviderHTTPError struct {
	code int
	msg  string
}

func (e *fakeProviderHTTPError) Error() string       { return fmt.Sprintf("upstream %d: %s", e.code, e.msg) }
func (e *fakeProviderHTTPError) HTTPStatusCode() int { return e.code }

var _ agentservice.ProviderHTTPError = (*fakeProviderHTTPError)(nil)

func sendMessageRequest() *pb.SendAgentChatMessageRequest {
	return &pb.SendAgentChatMessageRequest{
		ChatId:  uuid.NewString(),
		Content: "ping",
	}
}

func sendMessageStub(err error) *stubService {
	return &stubService{
		sendMessage: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (*models.AgentSessionMessage, error) {
			return nil, err
		},
	}
}

func TestSendAgentChatMessage_MapsContextCanceledToCanceled(t *testing.T) {
	svc := sendMessageStub(fmt.Errorf("forward to provider: %w", context.Canceled))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

func TestSendAgentChatMessage_MapsDeadlineExceeded(t *testing.T) {
	svc := sendMessageStub(fmt.Errorf("forward to provider: %w", context.DeadlineExceeded))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
}

func TestSendAgentChatMessage_MapsUpstreamRateLimit(t *testing.T) {
	svc := sendMessageStub(fmt.Errorf("forward to provider: %w", &fakeProviderHTTPError{code: http.StatusTooManyRequests, msg: "slow down"}))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestSendAgentChatMessage_MapsUpstream5xxToUnavailable(t *testing.T) {
	svc := sendMessageStub(fmt.Errorf("forward to provider: %w", &fakeProviderHTTPError{code: http.StatusBadGateway, msg: "bad gateway"}))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestSendAgentChatMessage_MapsUpstreamAuthToFailedPrecondition(t *testing.T) {
	svc := sendMessageStub(fmt.Errorf("forward to provider: %w", &fakeProviderHTTPError{code: http.StatusUnauthorized, msg: "bad token"}))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestSendAgentChatMessage_MapsUpstreamGatewayTimeoutToDeadline(t *testing.T) {
	svc := sendMessageStub(fmt.Errorf("forward to provider: %w", &fakeProviderHTTPError{code: http.StatusGatewayTimeout, msg: "timeout"}))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
}

func TestSendAgentChatMessage_FallsBackToInternal(t *testing.T) {
	svc := sendMessageStub(errors.New("unexpected bug"))

	_, err := actionsagents.SendAgentChatMessage(context.Background(), svc, uuid.NewString(), uuid.NewString(), sendMessageRequest())
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestInterruptAgentChat_MapsUpstreamErrors(t *testing.T) {
	svc := &stubService{
		interruptErr: fmt.Errorf("interrupt: %w", &fakeProviderHTTPError{code: http.StatusServiceUnavailable, msg: "down"}),
	}

	_, err := actionsagents.InterruptAgentChat(context.Background(), svc, uuid.NewString(), uuid.NewString(), &pb.InterruptAgentChatRequest{
		ChatId: uuid.NewString(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestDefineAgentOutcome_MapsContextCanceled(t *testing.T) {
	svc := &stubService{
		defineErr: fmt.Errorf("define outcome: %w", context.Canceled),
	}

	_, err := actionsagents.DefineAgentOutcome(context.Background(), svc, uuid.NewString(), uuid.NewString(), &pb.DefineAgentOutcomeRequest{
		ChatId: uuid.NewString(),
		Rubric: "must pass",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}
