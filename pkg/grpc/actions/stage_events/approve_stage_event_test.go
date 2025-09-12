package stageevents

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const StageEventApprovedRoutingKey = "stage-event-approved"

func Test__ApproveStageEvent(t *testing.T) {
	r := support.Setup(t)
	event := support.CreateStageEvent(t, r.Source, r.Stage)
	userID := uuid.New().String()
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID)

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(ctx, uuid.NewString(), r.Stage.ID.String(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(ctx, r.Canvas.ID.String(), uuid.NewString(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage event does not exist -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event not found", s.Message())
	})

	t.Run("approves and returns event", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, StageEventApprovedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		res, err := ApproveStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), event.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Event)
		assert.Equal(t, event.ID.String(), res.Event.Id)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, res.Event.State)
		assert.NotNil(t, res.Event.CreatedAt)
		require.Len(t, res.Event.Approvals, 1)
		assert.Equal(t, userID, res.Event.Approvals[0].ApprovedBy)
		assert.NotNil(t, res.Event.Approvals[0].ApprovedAt)

		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("approves with same requester ID -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event already approved by requester", s.Message())
	})
}
