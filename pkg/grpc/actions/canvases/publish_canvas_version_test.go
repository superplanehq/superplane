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
			r.Encryptor, r.Registry, r.GitProvider,
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
			r.Encryptor, r.Registry, r.GitProvider,
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
			r.Encryptor, r.Registry, r.GitProvider,
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
			r.Encryptor, r.Registry, r.GitProvider,
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
			r.Encryptor, r.Registry, r.GitProvider,
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
		createResp, err := CreateCanvasVersion(otherCtx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)
		draftVersionID := createResp.GetVersion().GetMetadata().GetId()

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("version not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-missing-version")

		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, uuid.New().String(),
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "version not found")
	})

	t.Run("draft version -> publishes and deletes draft", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-draft")
		draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Published Name")

		resp, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)
		require.NotNil(t, resp.Version)
		assert.Equal(t, pb.CanvasVersion_STATE_PUBLISHED, resp.Version.Metadata.State)
		assert.NotEmpty(t, resp.Version.Metadata.Id)
		assert.NotNil(t, resp.Version.Metadata.PublishedAt)

		// Publishing materializes a fresh live version from the merge commit and
		// reconciles (deletes) the draft branch version row.
		_, err = models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(draftVersionID))
		require.Error(t, err)

		// The canvas live version should point to the newly published version.
		canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)
		assert.Equal(t, resp.Version.Metadata.Id, canvas.LiveVersionID.String())
	})

	t.Run("draft version -> preserves canvas folder assignment", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-draft-in-folder")

		draftVersionID := createDraftVersionID(ctx, t, r, canvasID, "")

		commitDraftMetadataOnly(ctx, t, r, canvasID, draftVersionID, "publish-draft-in-folder-renamed", canvasID)

		folder, err := models.CreateCanvasFolder(r.Organization.ID, "Publish Folder", models.CanvasFolderColorBlue)
		require.NoError(t, err)

		_, err = models.UpdateCanvasFolderMembership(r.Organization.ID, uuid.MustParse(canvasID), &folder.ID)
		require.NoError(t, err)

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)

		canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)
		require.NotNil(t, canvas.CanvasFolderID)
		assert.Equal(t, folder.ID, *canvas.CanvasFolderID)
	})

	t.Run("metadata-only draft version -> publishes without graph changes", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-metadata-only")

		draftVersionID := createDraftVersionID(ctx, t, r, canvasID, "")

		commitDraftMetadataOnly(
			ctx, t, r, canvasID, draftVersionID,
			"publish-metadata-only-renamed",
			"updated through metadata-only publish",
		)

		resp, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)
		require.NotNil(t, resp.Version)
		assert.Equal(t, pb.CanvasVersion_STATE_PUBLISHED, resp.Version.Metadata.State)
		assert.Equal(t, "publish-metadata-only-renamed", resp.Version.Metadata.Name)
		assert.Equal(t, "updated through metadata-only publish", resp.Version.Metadata.Description)

		canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)
		assert.Equal(t, "publish-metadata-only-renamed", canvas.Name)
		assert.Equal(t, "updated through metadata-only publish", canvas.Description)
		assert.Equal(t, resp.Version.Metadata.Id, canvas.LiveVersionID.String())
	})

	t.Run("console-only draft changes -> publishes console to live", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-console-only")

		draftVersionID := createDraftVersionID(ctx, t, r, canvasID, "")

		commitDraftConsoleOnly(ctx, t, r, canvasID, draftVersionID, "published console")

		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)

		liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), uuid.MustParse(canvasID))
		require.NoError(t, err)
		panels := liveVersion.ConsolePanels.Data()
		require.Len(t, panels, 1)
		assert.Equal(t, "published console", panels[0].Content["body"])
	})

	t.Run("draft version with duplicate name -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_ = createGitCanvas(ctx, t, r, "publish-duplicate-live", nil)
		canvasID := createGitCanvas(ctx, t, r, "other-canvas", nil)

		draftVersionID := createDraftVersionID(ctx, t, r, canvasID, "")
		commitDraftMetadataOnly(ctx, t, r, canvasID, draftVersionID, "publish-duplicate-live", "")

		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
	})

}
