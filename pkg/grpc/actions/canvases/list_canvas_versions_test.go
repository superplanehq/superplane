package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestGetCanvasVersionLimit(t *testing.T) {
	require.Equal(t, uint32(DefaultLimit), getCanvasVersionLimit(0))
	require.Equal(t, uint32(20), getCanvasVersionLimit(20))
	require.Equal(t, uint32(MaxCanvasVersionLimit), getCanvasVersionLimit(MaxCanvasVersionLimit+1))
}

func TestListDraftCanvasVersions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), "invalid-id", 0, nil, pb.CanvasVersion_STATE_DRAFT)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, "invalid-id", uuid.New().String(), 0, nil, pb.CanvasVersion_STATE_DRAFT)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), uuid.New().String(), 0, nil, pb.CanvasVersion_STATE_DRAFT)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("returns empty list when no drafts exist", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-list-empty")

		response, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), canvasID, 0, nil, pb.CanvasVersion_STATE_DRAFT)
		require.NoError(t, err)
		assert.Empty(t, response.GetVersions())
	})

	t.Run("lists created draft branches newest first", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-list")

		first, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "First draft")
		require.NoError(t, err)
		second, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "Second draft")
		require.NoError(t, err)

		response, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), canvasID, 0, nil, pb.CanvasVersion_STATE_DRAFT)
		require.NoError(t, err)
		require.Len(t, response.GetVersions(), 2)
		assert.Equal(t, second.GetVersion().GetMetadata().GetDisplayName(), response.GetVersions()[0].GetMetadata().GetDisplayName())
		assert.Equal(t, first.GetVersion().GetMetadata().GetDisplayName(), response.GetVersions()[1].GetMetadata().GetDisplayName())
	})

	t.Run("lists only drafts owned by the current user", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "draft-branch-list-owner-filter")
		otherUser := support.CreateUser(t, r, r.Organization.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

		_, err := CreateCanvasVersion(otherCtx, r.Organization.ID.String(), canvasID, "Other user draft")
		require.NoError(t, err)
		ownDraft, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID, "My draft")
		require.NoError(t, err)

		response, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), canvasID, 0, nil, pb.CanvasVersion_STATE_DRAFT)
		require.NoError(t, err)
		require.Len(t, response.GetVersions(), 1)
		assert.Equal(t, ownDraft.GetVersion().GetMetadata().GetDisplayName(), response.GetVersions()[0].GetMetadata().GetDisplayName())
	})
}
