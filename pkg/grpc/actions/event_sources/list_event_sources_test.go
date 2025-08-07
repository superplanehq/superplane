package eventsources

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListEventSources(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	t.Run("invalid canvas -> error", func(t *testing.T) {
		_, err := ListEventSources(context.Background(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no event sources -> empty list", func(t *testing.T) {
		res, err := ListEventSources(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.EventSources)
	})

	t.Run("lists only external event sources", func(t *testing.T) {
		external := models.EventSource{
			CanvasID:    r.Canvas.ID,
			Name:        "external",
			Description: "external",
			Key:         []byte(`key`),
			Scope:       models.EventSourceScopeExternal,
		}

		err := external.Create([]models.EventType{}, nil)
		require.NoError(t, err)

		internal := models.EventSource{
			CanvasID:    r.Canvas.ID,
			Name:        "internal",
			Description: "internal",
			Key:         []byte(`key`),
			Scope:       models.EventSourceScopeInternal,
		}

		err = internal.Create([]models.EventType{}, nil)
		require.NoError(t, err)

		res, err := ListEventSources(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.EventSources, 1)
		assert.Equal(t, external.ID.String(), res.EventSources[0].Metadata.Id)
		assert.Equal(t, external.Name, res.EventSources[0].Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), res.EventSources[0].Metadata.CanvasId)
		assert.NotEmpty(t, res.EventSources[0].Metadata.CreatedAt)
	})
}
