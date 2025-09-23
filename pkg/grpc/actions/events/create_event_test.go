package events

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateEvent(t *testing.T) {
	r := support.Setup(t)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("no source type -> error", func(t *testing.T) {
		_, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, r.Source.ID.String(), "webhook", map[string]any{"test": "data"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid source type", s.Message())
	})

	t.Run("invalid canvas ID -> error", func(t *testing.T) {
		_, err := CreateEvent(ctx, "invalid-uuid", protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, r.Source.ID.String(), "webhook", map[string]any{"test": "data"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid canvas ID", s.Message())
	})

	t.Run("invalid source ID -> error", func(t *testing.T) {
		_, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, "invalid-uuid", "webhook", map[string]any{"test": "data"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid source ID", s.Message())
	})

	t.Run("empty event type -> error", func(t *testing.T) {
		_, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, r.Source.ID.String(), "", map[string]any{"test": "data"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event type is required", s.Message())
	})

	t.Run("source not found -> error", func(t *testing.T) {
		nonExistentID := uuid.NewString()
		_, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, nonExistentID, "webhook", map[string]any{"test": "data"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "source not found")
	})

	t.Run("create event for event source", func(t *testing.T) {
		rawData := map[string]any{
			"test":   "data",
			"number": 123,
			"nested": map[string]any{
				"key": "value",
			},
		}

		res, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, r.Source.ID.String(), "webhook", rawData)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Event)

		assert.Equal(t, r.Source.ID.String(), res.Event.SourceId)
		assert.Equal(t, r.Source.Name, res.Event.SourceName)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, res.Event.SourceType)
		assert.Equal(t, "webhook", res.Event.Type)
		assert.Equal(t, protos.Event_STATE_PENDING, res.Event.State)
		assert.NotNil(t, res.Event.ReceivedAt)
		assert.NotNil(t, res.Event.Raw)

		rawStruct := res.Event.Raw.AsMap()
		assert.Equal(t, "data", rawStruct["test"])
		assert.Equal(t, float64(123), rawStruct["number"])
		nested := rawStruct["nested"].(map[string]any)
		assert.Equal(t, "value", nested["key"])
	})

	t.Run("create event for stage", func(t *testing.T) {
		rawData := map[string]any{
			"stage_data": "test",
		}

		res, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_STAGE, r.Stage.ID.String(), "execution_complete", rawData)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Event)

		assert.Equal(t, r.Stage.ID.String(), res.Event.SourceId)
		assert.Equal(t, r.Stage.Name, res.Event.SourceName)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_STAGE, res.Event.SourceType)
		assert.Equal(t, "execution_complete", res.Event.Type)
		assert.Equal(t, protos.Event_STATE_PENDING, res.Event.State)
		assert.NotNil(t, res.Event.ReceivedAt)
		assert.NotNil(t, res.Event.Raw)

		rawStruct := res.Event.Raw.AsMap()
		assert.Equal(t, "test", rawStruct["stage_data"])
	})

	t.Run("invalid raw data -> error", func(t *testing.T) {
		invalidData := map[string]any{
			"invalid": make(chan int),
		}

		_, err := CreateEvent(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, r.Source.ID.String(), "webhook", invalidData)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid raw data")
	})
}
