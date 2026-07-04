package agents_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestInterruptAgentChat_InvalidChatID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{}
	_, err := actionsagents.InterruptAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.InterruptAgentChatRequest{
		ChatId: "not-a-uuid",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestInterruptAgentChat_NotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{
		interruptErr: fmt.Errorf("get session: record not found"),
	}
	_, err := actionsagents.InterruptAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.InterruptAgentChatRequest{
		ChatId: uuid.NewString(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, grpcerrors.Code(err))
}

func TestInterruptAgentChat_Success(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{}
	resp, err := actionsagents.InterruptAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.InterruptAgentChatRequest{
		ChatId: uuid.NewString(),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
