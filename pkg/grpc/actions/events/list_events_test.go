package events

import (
	"context"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test__ListEvents(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas with no events -> empty list", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
		assert.Equal(t, int64(0), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.NextTimestamp)
	})

	t.Run("canvas with events - list all", func(t *testing.T) {
		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data1"}`), []byte(`{"x-header": "value1"}`))
		require.NoError(t, err)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data2"}`), []byte(`{"x-header": "value2"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 2)
		assert.Equal(t, int64(2), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.NextTimestamp)

		e := res.Events[0]
		assert.Equal(t, event2.ID.String(), e.Id)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, r.Source.Name, e.SourceName)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, "webhook", e.Type)
		assert.Equal(t, protos.Event_STATE_PENDING, e.State)
		assert.NotNil(t, e.ReceivedAt)
		assert.NotNil(t, e.Raw)
		assert.NotNil(t, e.Headers)

		e = res.Events[1]
		assert.Equal(t, event1.ID.String(), e.Id)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, r.Source.Name, e.SourceName)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, "webhook", e.Type)
		assert.Equal(t, protos.Event_STATE_PENDING, e.State)
		assert.NotNil(t, e.ReceivedAt)
		assert.NotNil(t, e.Raw)
		assert.NotNil(t, e.Headers)
	})

	t.Run("filter by source type", func(t *testing.T) {
		_, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data1"}`), []byte(`{"x-header": "value1"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, "", 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 3)

		res, err = ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_STAGE, "", 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
	})

	t.Run("filter by source ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, r.Source.ID.String(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 3)

		res, err = ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, uuid.NewString(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
	})

	t.Run("limit parameter", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 2, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 2)

		res, err = ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 3)

		res, err = ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 100, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 3)
	})

	t.Run("before parameter", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "before1"}`), []byte(`{}`))
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "before2"}`), []byte(`{}`))
		require.NoError(t, err)

		beforeTime := timestamppb.New(*event1.ReceivedAt)
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 0, beforeTime)
		require.NoError(t, err)
		require.NotNil(t, res)

		require.Greater(t, len(res.Events), 0)
		for _, event := range res.Events {
			eventTime := event.ReceivedAt.AsTime()
			assert.True(t, eventTime.Before(*event1.ReceivedAt))
		}

		assert.NotContains(t, getEventIDs(res.Events), event2.ID.String())
	})

	t.Run("pagination fields", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

		for i := 0; i < 3; i++ {
			_, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "pagination"}`), []byte(`{}`))
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 2, nil)
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Greater(t, res.TotalCount, int64(0))
		assert.True(t, res.HasNextPage)
		assert.NotNil(t, res.NextTimestamp)
		require.Len(t, res.Events, 2)

		res2, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "", 2, res.NextTimestamp)
		require.NoError(t, err)
		require.NotNil(t, res2)

		assert.Greater(t, len(res2.Events), 0)
	})
}

func getEventIDs(events []*protos.Event) []string {
	ids := make([]string, len(events))
	for i, event := range events {
		ids[i] = event.Id
	}
	return ids
}
