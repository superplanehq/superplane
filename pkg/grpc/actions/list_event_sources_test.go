package actions

import (
	"context"
	"testing"

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

	t.Run("no org ID -> error", func(t *testing.T) {
		_, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{
			CanvasId: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{
			OrganizationId: r.Org.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("no event sources -> empty list", func(t *testing.T) {
		canvas, err := models.CreateCanvas(r.Org, r.User, "empty-canvas")
		require.NoError(t, err)

		res, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{
			CanvasId:       canvas.ID.String(),
			OrganizationId: r.Org.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.EventSources)
	})

	t.Run("with event source -> list", func(t *testing.T) {
		res, err := ListEventSources(context.Background(), &protos.ListEventSourcesRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.EventSources, 1)
		assert.Equal(t, r.Source.ID.String(), res.EventSources[0].Id)
		assert.Equal(t, r.Canvas.ID.String(), res.EventSources[0].CanvasId)
		assert.Equal(t, r.Org.String(), res.EventSources[0].OrganizationId)
		assert.NotEmpty(t, res.EventSources[0].CreatedAt)
	})
}
