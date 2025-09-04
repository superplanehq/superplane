package stageevents

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const StageEventCancelledRoutingKey = "stage-event-cancelled"

func Test__CancelStageEvent(t *testing.T) {
	r := support.Setup(t)
	event := support.CreateStageEvent(t, r.Source, r.Stage)
	userID := uuid.New().String()
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID)

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := CancelStageEvent(ctx, uuid.NewString(), r.Stage.ID.String(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := CancelStageEvent(ctx, r.Canvas.ID.String(), uuid.NewString(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage event does not exist -> error", func(t *testing.T) {
		_, err := CancelStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event not found", s.Message())
	})

	t.Run("cancels and returns event", func(t *testing.T) {
		// Create a new event for this test
		newEvent := support.CreateStageEvent(t, r.Source, r.Stage)
		
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, StageEventCancelledRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		res, err := CancelStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), newEvent.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Event)
		assert.Equal(t, newEvent.ID.String(), res.Event.Id)
		assert.Equal(t, r.Source.ID.String(), res.Event.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, res.Event.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PROCESSED, res.Event.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_CANCELLED, res.Event.StateReason)
		assert.NotNil(t, res.Event.CreatedAt)
		assert.Equal(t, userID, res.Event.CancelledBy)
		assert.NotNil(t, res.Event.CancelledAt)

		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("cancel already cancelled event -> error", func(t *testing.T) {
		// Create a new event and cancel it first
		cancelledEvent := support.CreateStageEvent(t, r.Source, r.Stage)
		_, err := CancelStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), cancelledEvent.ID.String())
		require.NoError(t, err)

		// Try to cancel again
		_, err = CancelStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), cancelledEvent.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event already cancelled", s.Message())
	})

	t.Run("cancel processed event -> error", func(t *testing.T) {
		// Create a new event and mark it as processed
		processedEvent := support.CreateStageEvent(t, r.Source, r.Stage)
		err := processedEvent.UpdateState(models.StageEventStateProcessed, models.StageEventStateReasonExecution)
		require.NoError(t, err)

		// Try to cancel
		_, err = CancelStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), processedEvent.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event cannot be cancelled", s.Message())
	})
}