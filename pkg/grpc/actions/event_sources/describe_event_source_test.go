package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeEventSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("invalid canvas ID -> error", func(t *testing.T) {
		_, err := DescribeEventSource(context.Background(), &protos.DescribeEventSourceRequest{
			Id: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := DescribeEventSource(context.Background(), &protos.DescribeEventSourceRequest{
			CanvasIdOrName: uuid.New().String(),
			Id:             uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("source that does not exist -> error", func(t *testing.T) {
		_, err := DescribeEventSource(context.Background(), &protos.DescribeEventSourceRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Id:             uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "event source not found", s.Message())
	})

	t.Run("no name and no ID -> error", func(t *testing.T) {
		_, err := DescribeEventSource(context.Background(), &protos.DescribeEventSourceRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "must specify one of: id or name", s.Message())
	})

	t.Run("using id", func(t *testing.T) {
		response, err := DescribeEventSource(context.Background(), &protos.DescribeEventSourceRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Id:             r.Source.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.Equal(t, r.Source.ID.String(), response.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, *r.Source.CreatedAt, response.EventSource.Metadata.CreatedAt.AsTime())
		assert.Equal(t, r.Source.Name, response.EventSource.Metadata.Name)
	})

	t.Run("using name", func(t *testing.T) {
		response, err := DescribeEventSource(context.Background(), &protos.DescribeEventSourceRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Name:           r.Source.Name,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.Equal(t, r.Source.ID.String(), response.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, *r.Source.CreatedAt, response.EventSource.Metadata.CreatedAt.AsTime())
		assert.Equal(t, r.Source.Name, response.EventSource.Metadata.Name)
	})
}
