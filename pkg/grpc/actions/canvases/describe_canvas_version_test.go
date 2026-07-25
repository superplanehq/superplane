package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__DescribeCanvasVersion(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("invalid version id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := DescribeCanvasVersion(ctx, database.DB(t.Context()), canvas, "invalid-id")
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("version not found -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := DescribeCanvasVersion(ctx, database.DB(t.Context()), canvas, uuid.New().String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("returns version metadata and spec", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
		require.NoError(t, err)

		response, err := DescribeCanvasVersion(ctx, database.DB(t.Context()), canvas, liveVersion.ID.String())
		require.NoError(t, err)
		assert.Equal(t, liveVersion.ID.String(), response.GetVersion().GetMetadata().GetId())
		require.NotNil(t, response.GetVersion().GetSpec())
		assert.Len(t, response.GetVersion().GetSpec().GetNodes(), len(liveVersion.Nodes))
		assert.Len(t, response.GetVersion().GetSpec().GetEdges(), len(liveVersion.Edges))
	})

	t.Run("returns version console spec", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
		require.NoError(t, err)

		panels := []models.ConsolePanel{
			{
				ID:      "notes",
				Type:    "markdown",
				Content: map[string]any{"body": "Hello"},
			},
		}
		layout := []models.ConsoleLayoutItem{
			{I: "notes", X: 0, Y: 0, W: 4, H: 2},
		}
		_, err = models.UpdateCanvasVersionConsoleInTransaction(database.DB(t.Context()), liveVersion, panels, layout)
		require.NoError(t, err)

		response, err := DescribeCanvasVersion(ctx, database.DB(t.Context()), canvas, liveVersion.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.GetVersion().GetSpec())
		assert.Equal(t, canvas.ID.String(), response.GetVersion().GetMetadata().GetCanvasId())
		require.Len(t, response.GetVersion().GetSpec().GetPanels(), 1)
		assert.Equal(t, "notes", response.GetVersion().GetSpec().GetPanels()[0].GetId())
		assert.Equal(t, "markdown", response.GetVersion().GetSpec().GetPanels()[0].GetType())
		require.Len(t, response.GetVersion().GetSpec().GetLayout(), 1)
		assert.Equal(t, "notes", response.GetVersion().GetSpec().GetLayout()[0].GetI())
	})
}
