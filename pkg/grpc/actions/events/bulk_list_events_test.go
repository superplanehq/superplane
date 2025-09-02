package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

func Test__BulkListEvents(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas with no events -> empty results", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   r.Source.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}
		res, err := BulkListEvents(ctx, r.Canvas.ID.String(), sources, 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		assert.Empty(t, res.Results[0].Events)
		assert.Equal(t, r.Source.ID.String(), res.Results[0].SourceId)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, res.Results[0].SourceType)
	})

	t.Run("canvas with events - bulk list multiple sources", func(t *testing.T) {
		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data1"}`), []byte(`{"x-header": "value1"}`))
		require.NoError(t, err)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data2"}`), []byte(`{"x-header": "value2"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   r.Source.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}

		res, err := BulkListEvents(ctx, r.Canvas.ID.String(), sources, 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		require.Len(t, res.Results[0].Events, 2)

		result := res.Results[0]
		assert.Equal(t, r.Source.ID.String(), result.SourceId)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, result.SourceType)

		e := result.Events[0]
		assert.Equal(t, event2.ID.String(), e.Id)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, r.Source.Name, e.SourceName)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, "webhook", e.Type)
		assert.Equal(t, protos.Event_STATE_PENDING, e.State)
		assert.NotNil(t, e.ReceivedAt)
		assert.NotNil(t, e.Raw)
		assert.NotNil(t, e.Headers)

		e = result.Events[1]
		assert.Equal(t, event1.ID.String(), e.Id)
	})

	t.Run("limit per source is respected", func(t *testing.T) {
		_, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data3"}`), []byte(`{"x-header": "value3"}`))
		require.NoError(t, err)

		_, err = models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data4"}`), []byte(`{"x-header": "value4"}`))
		require.NoError(t, err)

		_, err = models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data5"}`), []byte(`{"x-header": "value5"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   r.Source.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}

		res, err := BulkListEvents(ctx, r.Canvas.ID.String(), sources, 2, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		assert.Len(t, res.Results[0].Events, 2)
	})

	t.Run("multiple sources", func(t *testing.T) {
		source2 := &models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "test-source-2",
			Key:        []byte(`my-key-2`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := source2.Create()
		require.NoError(t, err)

		event3, err := models.CreateEvent(source2.ID, source2.CanvasID, source2.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data-source2"}`), []byte(`{"x-header": "value-source2"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   r.Source.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
			{
				SourceId:   source2.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}

		res, err := BulkListEvents(ctx, r.Canvas.ID.String(), sources, 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 2)

		result1 := res.Results[0]
		assert.Equal(t, r.Source.ID.String(), result1.SourceId)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, result1.SourceType)
		assert.Len(t, result1.Events, 5)
		result2 := res.Results[1]
		assert.Equal(t, source2.ID.String(), result2.SourceId)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, result2.SourceType)
		require.Len(t, result2.Events, 1)
		assert.Equal(t, event3.ID.String(), result2.Events[0].Id)
	})

	t.Run("invalid canvas ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   r.Source.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}

		_, err := BulkListEvents(ctx, "invalid-canvas-id", sources, 10, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid canvas ID")
	})

	t.Run("filter by source type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   "",
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}

		res, err := BulkListEvents(ctx, r.Canvas.ID.String(), sources, 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		result := res.Results[0]
		assert.Equal(t, "", result.SourceId)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, result.SourceType)

		assert.Greater(t, len(result.Events), 0)
	})

	t.Run("filter by before timestamp", func(t *testing.T) {
		// Create an event, wait, then create another event
		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "before_data1"}`), []byte(`{"x-header": "before_value1"}`))
		require.NoError(t, err)

		// Wait a bit to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
		beforeTime := time.Now()
		time.Sleep(10 * time.Millisecond)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "before_data2"}`), []byte(`{"x-header": "before_value2"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		sources := []*protos.EventSourceItemRequest{
			{
				SourceId:   r.Source.ID.String(),
				SourceType: protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE,
			},
		}

		// Test with before filter - should only return events created before the timestamp
		beforeTimestamp := timestamppb.New(beforeTime)
		res, err := BulkListEvents(ctx, r.Canvas.ID.String(), sources, 10, beforeTimestamp)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		// Should contain event1 but not event2
		foundEvent1 := false
		foundEvent2 := false
		for _, event := range res.Results[0].Events {
			if event.Id == event1.ID.String() {
				foundEvent1 = true
			}
			if event.Id == event2.ID.String() {
				foundEvent2 = true
			}
		}

		assert.True(t, foundEvent1, "Should find event1 (created before the timestamp)")
		assert.False(t, foundEvent2, "Should not find event2 (created after the timestamp)")
	})
}
