package stageevents

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DiscardStageEvent(t *testing.T) {
	r := support.Setup(t)
	event := support.CreateStageEvent(t, r.Source, r.Stage)
	userID := uuid.New().String()
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID)

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := DiscardStageEvent(ctx, uuid.NewString(), r.Stage.ID.String(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := DiscardStageEvent(ctx, r.Canvas.ID.String(), uuid.NewString(), event.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage event does not exist -> error", func(t *testing.T) {
		_, err := DiscardStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event not found", s.Message())
	})

	t.Run("discards and returns event", func(t *testing.T) {
		// Create a new event for this test
		newEvent := support.CreateStageEvent(t, r.Source, r.Stage)

		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, messages.StageEventDiscardedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		res, err := DiscardStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), newEvent.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Event)
		assert.Equal(t, newEvent.ID.String(), res.Event.Id)
		assert.Equal(t, r.Source.ID.String(), res.Event.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, res.Event.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_DISCARDED, res.Event.State)
		assert.NotNil(t, res.Event.CreatedAt)
		assert.Equal(t, userID, res.Event.DiscardedBy)
		assert.NotNil(t, res.Event.DiscardedAt)

		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("discard already discarded event -> error", func(t *testing.T) {
		// Create a new event and discard it first
		event := support.CreateStageEvent(t, r.Source, r.Stage)
		_, err := DiscardStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), event.ID.String())
		require.NoError(t, err)

		// Try to discard it again
		_, err = DiscardStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), event.ID.String())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event already discarded", s.Message())
	})

	t.Run("discard processed event -> error", func(t *testing.T) {
		// Create a new event and mark it as processed
		processedEvent := support.CreateStageEvent(t, r.Source, r.Stage)
		err := processedEvent.UpdateState(models.StageEventStateProcessed, "")
		require.NoError(t, err)

		// Try to discard
		_, err = DiscardStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), processedEvent.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event cannot be discarded", s.Message())
	})
}
