package eventsources

import (
	"context"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__DeleteEventSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := DeleteEventSource(context.Background(), uuid.NewString(), r.Source.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("source that does not exist -> error", func(t *testing.T) {
		_, err := DeleteEventSource(context.Background(), r.Canvas.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("delete event source by id successfully", func(t *testing.T) {
		source := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "test-source",
			Key:        []byte(`test-key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := source.Create()
		require.NoError(t, err)

		response, err := DeleteEventSource(context.Background(), r.Canvas.ID.String(), source.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindEventSource(source.ID)
		assert.Error(t, err)
		assert.True(t, err == gorm.ErrRecordNotFound)

		softDeletedSources, err := models.ListUnscopedSoftDeletedEventSources(10, time.Now().Add(time.Hour))
		require.NoError(t, err)
		found := false
		for _, s := range softDeletedSources {
			if s.ID == source.ID {
				found = true
				assert.Contains(t, s.Name, "deleted-")
				break
			}
		}
		assert.True(t, found, "Source should be in soft deleted list")
	})

	t.Run("delete event source by name successfully", func(t *testing.T) {
		source := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "test-source-by-name",
			Key:        []byte(`test-key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := source.Create()
		require.NoError(t, err)

		response, err := DeleteEventSource(context.Background(), r.Canvas.ID.String(), source.Name)
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindEventSource(source.ID)
		assert.Error(t, err)
		assert.True(t, err == gorm.ErrRecordNotFound)
	})

	t.Run("internal event source cannot be deleted", func(t *testing.T) {
		internalSource := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "internal-source",
			Key:        []byte(`key`),
			Scope:      models.EventSourceScopeInternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := internalSource.Create()
		require.NoError(t, err)

		_, err = DeleteEventSource(context.Background(), r.Canvas.ID.String(), internalSource.Name)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})
}
