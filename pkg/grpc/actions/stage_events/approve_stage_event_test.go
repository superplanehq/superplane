package stageevents

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const StageEventApprovedRoutingKey = "stage-event-approved"

func Test__ApproveStageEvent(t *testing.T) {
	r := support.Setup(t)
	event := support.CreateStageEvent(t, r.Source, r.Stage)
	userID := uuid.New().String()

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(context.Background(), &protos.ApproveStageEventRequest{
			StageIdOrName: uuid.New().String(),
			EventId:       event.ID.String(),
			RequesterId:   uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(context.Background(), &protos.ApproveStageEventRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			StageIdOrName:  uuid.New().String(),
			EventId:        event.ID.String(),
			RequesterId:    uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage event does not exist -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(context.Background(), &protos.ApproveStageEventRequest{
			CanvasIdOrName: r.Canvas.Name,
			StageIdOrName:  r.Stage.ID.String(),
			EventId:        uuid.New().String(),
			RequesterId:    uuid.New().String(),
		})

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

		res, err := ApproveStageEvent(context.Background(), &protos.ApproveStageEventRequest{
			CanvasIdOrName: r.Canvas.Name,
			StageIdOrName:  r.Stage.ID.String(),
			EventId:        event.ID.String(),
			RequesterId:    userID,
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Event)
		assert.Equal(t, event.ID.String(), res.Event.Id)
		assert.Equal(t, r.Source.ID.String(), res.Event.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, res.Event.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, res.Event.State)
		assert.NotNil(t, res.Event.CreatedAt)
		require.Len(t, res.Event.Approvals, 1)
		assert.Equal(t, userID, res.Event.Approvals[0].ApprovedBy)
		assert.NotNil(t, res.Event.Approvals[0].ApprovedAt)

		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("approves with same requester ID -> error", func(t *testing.T) {
		_, err := ApproveStageEvent(context.Background(), &protos.ApproveStageEventRequest{
			CanvasIdOrName: r.Canvas.Name,
			StageIdOrName:  r.Stage.ID.String(),
			EventId:        event.ID.String(),
			RequesterId:    userID,
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event already approved by requester", s.Message())
	})
}
