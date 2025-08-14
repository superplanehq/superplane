package events

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListEvents(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas with no events -> empty list", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
	})

	t.Run("canvas with events - list all", func(t *testing.T) {
		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data1"}`), []byte(`{"x-header": "value1"}`))
		require.NoError(t, err)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data2"}`), []byte(`{"x-header": "value2"}`))
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, "")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 2)

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
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, "")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 3)

		res, err = ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_STAGE, "")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
	})

	t.Run("filter by source ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		res, err := ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, r.Source.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Events, 3)

		res, err = ListEvents(ctx, r.Canvas.ID.String(), protos.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN, uuid.NewString())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
	})
}
