package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListEventSources(t *testing.T) {
	r := support.Setup(t)

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no event sources -> empty list", func(t *testing.T) {
		org, err := models.CreateOrganization(uuid.New(), "test", "test")
		require.NoError(t, err)

		canvas, err := models.CreateCanvas(r.User, org.ID, "empty-canvas")
		require.NoError(t, err)

		res, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{
			CanvasIdOrName: canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.EventSources)
	})

	t.Run("with event source -> list", func(t *testing.T) {
		res, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.EventSources, 1)
		assert.Equal(t, r.Source.ID.String(), res.EventSources[0].Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), res.EventSources[0].Metadata.CanvasId)
		assert.NotEmpty(t, res.EventSources[0].Metadata.CreatedAt)
	})
}
