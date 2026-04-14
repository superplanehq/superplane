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

func Test__DeleteCanvasVersion(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), "invalid-id", uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid version id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), uuid.New().String(), "invalid-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), uuid.New().String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("version not found -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("published version -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), canvas.LiveVersionID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "only draft versions can be discarded")
	})

	t.Run("draft owned by another user -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		otherUser := support.CreateUser(t, r, r.Organization.ID)
		draft, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, otherUser.ID, nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err = DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), draft.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("draft owned by user -> deletes version", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draft, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), draft.ID.String())
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify the version no longer exists
		_, err = models.FindCanvasVersion(canvas.ID, draft.ID)
		assert.Error(t, err)
	})

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := DeleteCanvasVersion(context.Background(), r.Organization.ID.String(), uuid.New().String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})
}
