package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DeleteCanvasVersion(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), "invalid-id", missingCommitSHA)
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
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), uuid.New().String(), missingCommitSHA)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("version not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-version-missing")

		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, missingCommitSHA)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("published version -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-published-version")

		canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)
		require.NotNil(t, canvas.LiveVersionID)

		_, err = DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, *canvas.LiveVersionID)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "only draft versions can be discarded")
	})

	t.Run("draft owned by another user -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-other-user-draft")

		otherUser := support.CreateUser(t, r, r.Organization.ID)
		otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
		draftVersionID := createDraftVersion(otherUserCtx, t, r, canvasID, "Other User Draft")

		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("draft owned by user -> deletes version", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "delete-own-draft")
		draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft To Delete")

		resp, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
		require.NoError(t, err)
		require.NotNil(t, resp)

		_, err = models.FindCanvasVersion(uuid.MustParse(canvasID), draftVersionID)
		assert.Error(t, err)
	})

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := DeleteCanvasVersion(context.Background(), r.Organization.ID.String(), uuid.New().String(), missingCommitSHA)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})
}
