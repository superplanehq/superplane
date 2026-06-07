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

func TestGetCanvasVersionLimit(t *testing.T) {
	require.Equal(t, uint32(DefaultLimit), getCanvasVersionLimit(0))
	require.Equal(t, uint32(20), getCanvasVersionLimit(20))
	require.Equal(t, uint32(MaxCanvasVersionLimit), getCanvasVersionLimit(MaxCanvasVersionLimit+1))
}

func TestCreateCanvasVersion(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := CreateCanvasVersion(context.Background(), r.Organization.ID.String(), uuid.New().String(), "")
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("creates draft branch and version", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-create")
		canvasUUID := uuid.MustParse(canvasID)

		response, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)

		version := response.GetVersion()
		require.NotNil(t, version)
		require.NotNil(t, version.GetMetadata())
		assert.NotEmpty(t, version.GetMetadata().GetBranchName())
		assert.Equal(t, "Draft #1", version.GetMetadata().GetDisplayName())
		assert.NotEmpty(t, version.GetMetadata().GetId())
		require.NotNil(t, version.GetMetadata().GetOwner())
		assert.Equal(t, r.User.String(), version.GetMetadata().GetOwner().GetId())

		stored := findRegisteredDraftBranch(t, canvasUUID, version.GetMetadata().GetBranchName())
		assert.Equal(t, version.GetMetadata().GetId(), stored.ID.String())

		versionUUID, err := uuid.Parse(version.GetMetadata().GetId())
		require.NoError(t, err)
		storedVersion, err := models.FindCanvasVersion(canvasUUID, versionUUID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasVersionStateDraft, storedVersion.State)
	})

	t.Run("uses display name override when provided", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-named")
		canvasUUID := uuid.MustParse(canvasID)

		response, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "Release prep")
		require.NoError(t, err)

		version := response.GetVersion()
		require.NotNil(t, version)
		require.NotNil(t, version.GetMetadata())
		assert.Equal(t, "Release prep", version.GetMetadata().GetDisplayName())

		stored := findRegisteredDraftBranch(t, canvasUUID, version.GetMetadata().GetBranchName())
		assert.Equal(t, "Release prep", stored.DisplayName)
	})
}

func TestDeleteCanvasVersion(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := DeleteCanvasVersion(context.Background(), r.Organization.ID.String(), uuid.New().String(), uuid.New().String())
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), "invalid-id", uuid.New().String())
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := DeleteCanvasVersion(ctx, "invalid-id", uuid.New().String(), uuid.New().String())
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid version id -> error", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-delete-invalid-version")
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "invalid-id")
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), uuid.New().String(), uuid.New().String())
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("template canvas -> error", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-delete-template")
		canvasUUID := uuid.MustParse(canvasID)

		createResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)
		versionID := createResponse.GetVersion().GetMetadata().GetId()

		require.NoError(t, database.Conn().
			Model(&models.Canvas{}).
			Where("id = ?", canvasUUID).
			Update("is_template", true).
			Error)

		_, err = DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, versionID)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "templates are read-only")
	})

	t.Run("missing version -> not found", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-delete-missing")
		_, err := DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, uuid.New().String())
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("draft owned by another user -> error", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-delete-other-owner")

		createResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)
		versionID := createResponse.GetVersion().GetMetadata().GetId()

		otherUser := support.CreateUser(t, r, r.Organization.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

		_, err = DeleteCanvasVersion(otherCtx, r.Organization.ID.String(), canvasID, versionID)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("deletes draft branch and version", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-delete")
		canvasUUID := uuid.MustParse(canvasID)

		createResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)
		branchName := createResponse.GetVersion().GetMetadata().GetBranchName()
		versionID := createResponse.GetVersion().GetMetadata().GetId()

		_, err = DeleteCanvasVersion(ctx, r.Organization.ID.String(), canvasID, versionID)
		require.NoError(t, err)

		err = findRegisteredDraftBranchErr(canvasUUID, branchName)
		require.Error(t, err)

		versionUUID, err := uuid.Parse(versionID)
		require.NoError(t, err)
		_, err = models.FindCanvasVersion(canvasUUID, versionUUID)
		require.Error(t, err)
	})
}

func TestListDraftCanvasVersions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), "invalid-id", 0, nil, pb.CanvasVersion_STATE_DRAFT)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, "invalid-id", uuid.New().String(), 0, nil, pb.CanvasVersion_STATE_DRAFT)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), uuid.New().String(), 0, nil, pb.CanvasVersion_STATE_DRAFT)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("returns empty list when no drafts exist", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-list-empty")

		response, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), canvasID, 0, nil, pb.CanvasVersion_STATE_DRAFT)
		require.NoError(t, err)
		assert.Empty(t, response.GetVersions())
	})

	t.Run("lists created draft branches", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-list")

		first, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "First draft")
		require.NoError(t, err)
		second, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "Second draft")
		require.NoError(t, err)

		response, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), canvasID, 0, nil, pb.CanvasVersion_STATE_DRAFT)
		require.NoError(t, err)
		require.Len(t, response.GetVersions(), 2)

		displayNames := []string{
			response.GetVersions()[0].GetMetadata().GetDisplayName(),
			response.GetVersions()[1].GetMetadata().GetDisplayName(),
		}
		assert.Contains(t, displayNames, first.GetVersion().GetMetadata().GetDisplayName())
		assert.Contains(t, displayNames, second.GetVersion().GetMetadata().GetDisplayName())
	})
}
