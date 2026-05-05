package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__GetCanvasLaunchpad(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns empty launchpad when none has been saved", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := GetCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Launchpad)
		assert.Equal(t, canvas.ID.String(), resp.Launchpad.CanvasId)
		assert.Empty(t, resp.Launchpad.Panels)
		assert.Empty(t, resp.Launchpad.Layout)
	})

	t.Run("returns saved panels and layout", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := models.UpsertCanvasLaunchpadInTransaction(database.Conn(), canvas.ID,
			[]models.LaunchpadPanel{
				{ID: "p1", Type: "markdown", Content: map[string]any{"body": "# Hello"}},
			},
			[]models.LaunchpadLayoutItem{
				{I: "p1", X: 0, Y: 0, W: 6, H: 4},
			},
		)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := GetCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, resp.Launchpad)
		require.Len(t, resp.Launchpad.Panels, 1)
		assert.Equal(t, "p1", resp.Launchpad.Panels[0].Id)
		assert.Equal(t, "markdown", resp.Launchpad.Panels[0].Type)
		require.NotNil(t, resp.Launchpad.Panels[0].Content)
		require.Len(t, resp.Launchpad.Layout, 1)
		assert.Equal(t, "p1", resp.Launchpad.Layout[0].I)
		assert.Equal(t, int32(6), resp.Launchpad.Layout[0].W)
		assert.Equal(t, int32(4), resp.Launchpad.Layout[0].H)
	})

	t.Run("invalid canvas id -> InvalidArgument", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := GetCanvasLaunchpad(ctx, r.Organization.ID.String(), "not-a-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("unknown canvas -> NotFound", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := GetCanvasLaunchpad(ctx, r.Organization.ID.String(), uuid.New().String())
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("surfaces auto_height when stored", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		autoHeight := true
		_, err := models.UpsertCanvasLaunchpadInTransaction(database.Conn(), canvas.ID,
			[]models.LaunchpadPanel{
				{ID: "p1", Type: "markdown", Content: map[string]any{"body": "table"}},
			},
			[]models.LaunchpadLayoutItem{
				{I: "p1", X: 0, Y: 0, W: 6, H: 4, AutoHeight: &autoHeight},
			},
		)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := GetCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.Len(t, resp.Launchpad.Layout, 1)
		require.NotNil(t, resp.Launchpad.Layout[0].AutoHeight)
		assert.Equal(t, true, *resp.Launchpad.Layout[0].AutoHeight)
	})
}
