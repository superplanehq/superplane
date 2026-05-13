package agents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	actionsagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestDeleteAgentChat_TranslatesNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{
		archiveSession: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
			return gorm.ErrRecordNotFound
		},
	}
	_, err := actionsagents.DeleteAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.DeleteAgentChatRequest{
		ChatId: uuid.NewString(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeleteAgentChat_BubblesUnknownError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	svc := &stubService{
		archiveSession: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
			return errors.New("boom")
		},
	}
	_, err := actionsagents.DeleteAgentChat(context.Background(), svc, r.Organization.ID.String(), r.User.String(), &pb.DeleteAgentChatRequest{
		ChatId: uuid.NewString(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
