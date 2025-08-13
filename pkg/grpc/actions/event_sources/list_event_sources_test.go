package eventsources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListEventSources(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	t.Run("no event sources -> empty list", func(t *testing.T) {
		res, err := ListEventSources(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.EventSources)
	})

	t.Run("lists only external event sources", func(t *testing.T) {
		external := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "external",
			Key:        []byte(`key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}

		err := external.Create()
		require.NoError(t, err)

		internal := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "internal",
			Key:        []byte(`key`),
			Scope:      models.EventSourceScopeInternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}

		err = internal.Create()
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
