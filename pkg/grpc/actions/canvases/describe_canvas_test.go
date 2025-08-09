package canvases

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := DescribeCanvas(context.Background(), &protos.DescribeCanvasRequest{
			Id: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("empty canvas", func(t *testing.T) {
		response, err := DescribeCanvas(context.Background(), &protos.DescribeCanvasRequest{
			Id: r.Canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Metadata)
		assert.Equal(t, r.Canvas.ID.String(), response.Canvas.Metadata.Id)
		assert.Equal(t, *r.Canvas.CreatedAt, response.Canvas.Metadata.CreatedAt.AsTime())
		assert.Equal(t, "test", response.Canvas.Metadata.Name)
		assert.Equal(t, r.Canvas.CreatedBy.String(), response.Canvas.Metadata.CreatedBy)
	})
}
