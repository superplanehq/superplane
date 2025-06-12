package actions

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeCanvas(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	userID := uuid.New()

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
		canvas, err := models.CreateCanvas(userID, "test")
		require.NoError(t, err)

		response, err := DescribeCanvas(context.Background(), &protos.DescribeCanvasRequest{
			Id: canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Metadata)
		assert.Equal(t, canvas.ID.String(), response.Canvas.Metadata.Id)
		assert.Equal(t, *canvas.CreatedAt, response.Canvas.Metadata.CreatedAt.AsTime())
		assert.Equal(t, "test", response.Canvas.Metadata.Name)
		assert.Equal(t, canvas.CreatedBy.String(), response.Canvas.Metadata.CreatedBy)
	})
}
