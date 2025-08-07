package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ResetEventSourceKey(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := ResetEventSourceKey(context.Background(), r.Encryptor, uuid.NewString(), r.Source.Name)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("source that does not exist -> error", func(t *testing.T) {
		_, err := ResetEventSourceKey(context.Background(), r.Encryptor, r.Canvas.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("key is reset using source id", func(t *testing.T) {
		response, err := ResetEventSourceKey(context.Background(), r.Encryptor, r.Canvas.ID.String(), r.Source.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.Equal(t, r.Source.ID.String(), response.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, r.Source.Name, response.EventSource.Metadata.Name)
		assert.NotEmpty(t, response.EventSource.Metadata.UpdatedAt)
		assert.NotEmpty(t, response.Key)
	})

	t.Run("key is reset using source name", func(t *testing.T) {
		response, err := ResetEventSourceKey(context.Background(), r.Encryptor, r.Canvas.ID.String(), r.Source.Name)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.Equal(t, r.Source.ID.String(), response.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, r.Source.Name, response.EventSource.Metadata.Name)
		assert.NotEmpty(t, response.EventSource.Metadata.UpdatedAt)
		assert.NotEmpty(t, response.Key)
	})
}
