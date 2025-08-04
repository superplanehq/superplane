package eventsources

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListEventSources(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	t.Run("invalid canvas -> error", func(t *testing.T) {
		_, err := ListEventSources(context.Background(), uuid.NewString(), &protos.ListEventSourcesRequest{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no event sources -> empty list", func(t *testing.T) {
		res, err := ListEventSources(context.Background(), r.Canvas.ID.String(), &protos.ListEventSourcesRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.EventSources)
	})

	t.Run("lists only external event sources", func(t *testing.T) {
		external, err := r.Canvas.CreateEventSource("external", "external", []byte("key"), models.EventSourceScopeExternal, []models.EventType{}, nil)
		require.NoError(t, err)

		_, err = r.Canvas.CreateEventSource("internal", "internal", []byte(`key`), models.EventSourceScopeInternal, []models.EventType{}, nil)
		require.NoError(t, err)

		res, err := ListEventSources(context.Background(), r.Canvas.ID.String(), &protos.ListEventSourcesRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.EventSources, 1)
		assert.Equal(t, external.ID.String(), res.EventSources[0].Metadata.Id)
		assert.Equal(t, external.Name, res.EventSources[0].Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), res.EventSources[0].Metadata.CanvasId)
		assert.NotEmpty(t, res.EventSources[0].Metadata.CreatedAt)
	})
}
