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
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__PublishCanvasVersion(t *testing.T) {
	r := support.Setup(t)

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := PublishCanvasVersion(
			context.Background(),
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), uuid.New().String(), uuid.New().String(),
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), "invalid-id", uuid.New().String(),
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid version id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), uuid.New().String(), "invalid-id",
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), uuid.New().String(), uuid.New().String(),
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("published version -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-published")

		canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), canvasID, canvas.LiveVersionID.String(),
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "only draft versions can be published")
	})

	t.Run("draft owned by another user -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-other-user")

		otherUser := support.CreateUser(t, r, r.Organization.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
		createResp, err := CreateCanvasVersion(otherCtx, r.Organization.ID.String(), canvasID)
		require.NoError(t, err)
		draftVersionID := createResp.Version.Metadata.Id

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("draft version -> publishes and deletes draft", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-draft")
		draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Published Name")

		resp, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)
		require.NotNil(t, resp.Version)
		assert.Equal(t, pb.CanvasVersion_STATE_PUBLISHED, resp.Version.Metadata.State)
		assert.Equal(t, draftVersionID, resp.Version.Metadata.Id)
		assert.NotNil(t, resp.Version.Metadata.PublishedAt)

		// The same version should now be published (not deleted)
		version, err := models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(draftVersionID))
		require.NoError(t, err)
		assert.Equal(t, models.CanvasVersionStatePublished, version.State)

		// The canvas live version should point to it
		canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)
		assert.Equal(t, draftVersionID, canvas.LiveVersionID.String())
	})

	t.Run("versioning disabled -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("versioning_enabled", false).Error,
		)
		require.NoError(t,
			database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("versioning_enabled", false).Error,
		)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), canvas.ID.String(), uuid.New().String(),
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "versioning is disabled")
	})
}
