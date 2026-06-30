package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__PublishCanvasVersion(t *testing.T) {
	r := support.Setup(t)

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := PublishCanvasVersion(
			context.Background(),
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), uuid.New().String(), uuid.New().String(),
			"", testWebhookBaseURL, r.AuthService,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), "invalid-id", uuid.New().String(),
			"", testWebhookBaseURL, r.AuthService,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid version id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), uuid.New().String(), "invalid-id",
			"", testWebhookBaseURL, r.AuthService,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), uuid.New().String(), uuid.New().String(),
			"", testWebhookBaseURL, r.AuthService,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
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
			"", testWebhookBaseURL, r.AuthService,
		)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, msg, "only draft versions can be published")
	})

	t.Run("draft owned by another user -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-other-user")

		otherUser := support.CreateUser(t, r, r.Organization.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
		createResp, err := CreateCanvasVersion(otherCtx, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)
		draftVersionID := createResp.GetVersion().GetMetadata().GetId()

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
	})

	t.Run("draft version with staged changes -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-staged")
		draftVersionID := createDraftVersionID(ctx, t, r.Organization.ID.String(), canvasID, "")
		draftVersionUUID := uuid.MustParse(draftVersionID)

		_, err := models.UpsertWorkflowStagingPath(
			draftVersionUUID,
			r.Organization.ID,
			"canvas.yaml",
			"apiVersion: v1\nkind: Canvas\nspec:\n  nodes: []\n  edges: []\n",
			"",
			&r.User,
		)
		require.NoError(t, err)

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
		)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, msg, "staged changes")

		version, err := models.FindCanvasVersion(uuid.MustParse(canvasID), draftVersionUUID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasVersionStateDraft, version.State)
	})

	t.Run("version not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-missing-version")

		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, uuid.New().String(),
			"", testWebhookBaseURL, r.AuthService,
		)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Contains(t, msg, "version not found")
	})

	t.Run("draft version -> publishes and deletes draft", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-draft")
		draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Published Name")

		resp, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
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

	t.Run("draft version -> preserves canvas folder assignment", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-draft-in-folder")

		draftVersionID := createDraftVersionID(ctx, t, r.Organization.ID.String(), canvasID, "")

		require.NoError(t, database.Conn().
			Model(&models.CanvasVersion{}).
			Where("id = ?", uuid.MustParse(draftVersionID)).
			Update("name", "publish-draft-in-folder-renamed").
			Error)

		folder, err := models.CreateCanvasFolder(r.Organization.ID, "Publish Folder", models.CanvasFolderColorBlue)
		require.NoError(t, err)

		_, err = models.UpdateCanvasFolderMembership(r.Organization.ID, uuid.MustParse(canvasID), &folder.ID)
		require.NoError(t, err)

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
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

		draftVersionID := createDraftVersionID(ctx, t, r.Organization.ID.String(), canvasID, "")

		require.NoError(t, database.Conn().
			Model(&models.CanvasVersion{}).
			Where("id = ?", uuid.MustParse(draftVersionID)).
			Updates(map[string]any{
				"name":        "publish-metadata-only-renamed",
				"description": "updated through metadata-only publish",
			}).
			Error)

		resp, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
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
		assert.Equal(t, draftVersionID, canvas.LiveVersionID.String())
	})

	t.Run("console-only draft changes -> publishes console to live", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-console-only")

		draftVersionID := createDraftVersionID(ctx, t, r.Organization.ID.String(), canvasID, "")

		draftVersion, err := models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(draftVersionID))
		require.NoError(t, err)

		_, err = models.UpdateCanvasVersionConsoleInTransaction(
			database.Conn(),
			draftVersion,
			[]models.ConsolePanel{
				{ID: "notes", Type: models.ConsolePanelTypeMarkdown, Content: map[string]any{"body": "published console"}},
			},
			[]models.ConsoleLayoutItem{
				{I: "notes", X: 0, Y: 0, W: 4, H: 2},
			},
		)
		require.NoError(t, err)

		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvasID, draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
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
		existingCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		require.NoError(t, database.Conn().
			Model(&models.CanvasVersion{}).
			Where("id = ?", *existingCanvas.LiveVersionID).
			Update("name", "publish-duplicate-live").
			Error)
		require.NoError(t, database.Conn().
			Model(&models.Canvas{}).
			Where("id = ?", existingCanvas.ID).
			Update("name", "publish-duplicate-live").
			Error)

		draftVersionID := createDraftVersionID(ctx, t, r.Organization.ID.String(), canvas.ID.String(), "")

		require.NoError(t, database.Conn().
			Model(&models.CanvasVersion{}).
			Where("id = ?", uuid.MustParse(draftVersionID)).
			Update("name", "publish-duplicate-live").
			Error)

		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry, r.GitProvider,
			r.Organization.ID.String(), canvas.ID.String(), draftVersionID,
			"", testWebhookBaseURL, r.AuthService,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
	})

}
