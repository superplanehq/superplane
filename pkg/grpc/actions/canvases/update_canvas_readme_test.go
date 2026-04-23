package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateCanvasReadme(t *testing.T) {
	r := support.Setup(t)

	t.Run("auto-creates a draft when no version_id is given", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := UpdateCanvasReadme(ctx, r.Organization.ID.String(), canvas.ID.String(), "", "# draft readme")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "# draft readme", resp.Content)

		draft, err := models.FindCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User)
		require.NoError(t, err)
		assert.Equal(t, resp.VersionId, draft.ID.String())
		assert.Equal(t, "# draft readme", draft.Readme)
	})

	t.Run("updates an existing draft when version_id matches", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draft, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := UpdateCanvasReadme(
			ctx, r.Organization.ID.String(), canvas.ID.String(), draft.ID.String(), "updated",
		)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, draft.ID.String(), resp.VersionId)
		assert.Equal(t, "updated", resp.Content)
	})

	t.Run("rejects published version", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err = UpdateCanvasReadme(
			ctx, r.Organization.ID.String(), canvas.ID.String(), liveVersion.ID.String(), "nope",
		)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
	})

	t.Run("rejects invalid canvas id", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasReadme(ctx, r.Organization.ID.String(), "not-a-uuid", "", "x")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})
}
