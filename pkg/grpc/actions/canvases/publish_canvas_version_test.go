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
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
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

	t.Run("version not found -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-missing-version")

		_, err := PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
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

	//
	// Regression for the user-reported flow:
	//   1. Update readme   (UpdateCanvasReadme)
	//   2. Publish canvas  (readme should now be live)
	//   3. Click Edit      (CreateCanvasVersion, draft inherits readme)
	//   4. Edit a node     (UpdateCanvasVersion -- spec carries no readme)
	//   5. Publish canvas  (readme MUST still be live after this publish)
	//
	// Before the fix in UpdateCanvasVersion, step 4 silently overwrote the
	// draft's readme with "" because the frontend mutation only ships
	// nodes/edges. Step 5 then promoted that empty draft to live, which
	// the user observed as "Readme disappears".
	//
	t.Run("readme survives full update-readme -> publish -> edit-node -> publish flow", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		// Disable change-management for this canvas: the user reported the
		// bug on the direct-publish path, not the change-request path.
		require.NoError(
			t,
			database.Conn().
				Model(&models.Organization{}).
				Where("id = ?", r.Organization.ID).
				Update("change_management_enabled", false).
				Error,
		)
		t.Cleanup(func() {
			_ = database.Conn().
				Model(&models.Organization{}).
				Where("id = ?", r.Organization.ID).
				Update("change_management_enabled", true).
				Error
		})

		canvasID := createCanvasWithNoopNode(ctx, t, r, "readme-survives-publish-cycle")

		// Step 1: write a readme. UpdateCanvasReadme creates the draft when
		// none exists and sets the readme on it.
		readmeContent := "# my saved readme"
		readmeResp, err := UpdateCanvasReadme(
			ctx,
			r.Organization.ID.String(),
			canvasID,
			"",
			readmeContent,
		)
		require.NoError(t, err)
		require.NotNil(t, readmeResp)
		firstDraftVersionID := readmeResp.VersionId

		// Step 2: publish the readme draft -> live readme is now set.
		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), canvasID, firstDraftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)

		liveAfterReadmePublish, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), uuid.MustParse(canvasID))
		require.NoError(t, err)
		require.Equal(t, readmeContent, liveAfterReadmePublish.Readme, "live readme should be set after publish")

		// Step 3: click Edit -> CreateCanvasVersion clones a fresh draft
		// from live, carrying the readme along.
		createResp, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
		require.NoError(t, err)
		secondDraftVersionID := createResp.Version.Metadata.Id

		secondDraft, err := models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(secondDraftVersionID))
		require.NoError(t, err)
		require.Equal(t, readmeContent, secondDraft.Readme, "fresh draft should inherit live readme")

		// Step 4: edit a node via UpdateCanvasVersion -- mimicking the
		// frontend's autosave that only ships nodes/edges, never readme.
		_, err = UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			secondDraftVersionID,
			&pb.Canvas{
				Metadata: &pb.Canvas_Metadata{Name: "readme-survives-publish-cycle"},
				Spec: &pb.Canvas_Spec{
					Nodes: []*componentpb.Node{
						{
							Id:   "node-1",
							Name: "Renamed Node",
							Type: componentpb.Node_TYPE_COMPONENT,
							Component: &componentpb.Node_ComponentRef{
								Name: "noop",
							},
						},
					},
					Edges: []*componentpb.Edge{},
					// Readme intentionally left empty -- the frontend mutation
					// does not include it.
				},
			},
			nil,
			testWebhookBaseURL,
			r.AuthService,
		)
		require.NoError(t, err)

		afterAutosave, err := models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(secondDraftVersionID))
		require.NoError(t, err)
		assert.Equal(t, readmeContent, afterAutosave.Readme,
			"node-only update must not clobber readme on the draft")

		// Step 5: publish the edited draft -> readme MUST still be live.
		_, err = PublishCanvasVersion(
			ctx,
			r.Encryptor, r.Registry,
			r.Organization.ID.String(), canvasID, secondDraftVersionID,
			testWebhookBaseURL, r.AuthService,
		)
		require.NoError(t, err)

		liveAfterNodeEdit, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), uuid.MustParse(canvasID))
		require.NoError(t, err)
		assert.Equal(t, readmeContent, liveAfterNodeEdit.Readme,
			"live readme must still be set after a node edit + publish cycle")
	})

}
