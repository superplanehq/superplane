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

func Test__GetCanvasReadme(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns readme of live version when version_id is empty", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
		require.NoError(t, err)
		require.NoError(t, models.UpdateCanvasVersionReadmeInTransaction(database.Conn(), liveVersion, "# Hello live"))

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := GetCanvasReadme(ctx, r.Organization.ID.String(), canvas.ID.String(), "")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "# Hello live", resp.Content)
		assert.Equal(t, liveVersion.ID.String(), resp.VersionId)
	})

	t.Run("draft returns the caller's draft readme", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draft, err := models.SaveCanvasDraftWithReadmeInTransaction(
			database.Conn(), canvas.ID, r.User, nil, nil, "draft readme",
		)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := GetCanvasReadme(ctx, r.Organization.ID.String(), canvas.ID.String(), "draft")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "draft readme", resp.Content)
		assert.Equal(t, draft.ID.String(), resp.VersionId)
	})

	t.Run("invalid canvas id -> InvalidArgument", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := GetCanvasReadme(ctx, r.Organization.ID.String(), "not-a-uuid", "")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("unknown canvas -> NotFound", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := GetCanvasReadme(ctx, r.Organization.ID.String(), uuid.New().String(), "")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("missing draft -> NotFound", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := GetCanvasReadme(ctx, r.Organization.ID.String(), canvas.ID.String(), "draft")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})
}
