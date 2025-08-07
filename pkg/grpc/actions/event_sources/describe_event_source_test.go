package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeEventSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := DescribeEventSource(context.Background(), uuid.New().String(), r.Source.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("source that does not exist -> error", func(t *testing.T) {
		_, err := DescribeEventSource(context.Background(), r.Canvas.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("using id", func(t *testing.T) {
		response, err := DescribeEventSource(context.Background(), r.Canvas.ID.String(), r.Source.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.Equal(t, r.Source.ID.String(), response.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, *r.Source.CreatedAt, response.EventSource.Metadata.CreatedAt.AsTime())
		assert.Equal(t, r.Source.Name, response.EventSource.Metadata.Name)
	})

	t.Run("using name", func(t *testing.T) {
		response, err := DescribeEventSource(context.Background(), r.Canvas.ID.String(), r.Source.Name)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.Equal(t, r.Source.ID.String(), response.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, *r.Source.CreatedAt, response.EventSource.Metadata.CreatedAt.AsTime())
		assert.Equal(t, r.Source.Name, response.EventSource.Metadata.Name)
	})

	t.Run("internal event source cannot be described", func(t *testing.T) {
		internalSource := models.EventSource{
			CanvasID:    r.Canvas.ID,
			Name:        "internal",
			Description: "internal",
			Key:         []byte(`key`),
			Scope:       models.EventSourceScopeInternal,
		}

		err := internalSource.Create([]models.EventType{}, nil)
		require.NoError(t, err)

		_, err = DescribeEventSource(context.Background(), r.Canvas.ID.String(), internalSource.Name)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})
}
