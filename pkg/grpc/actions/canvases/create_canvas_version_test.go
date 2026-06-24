package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestCreateCanvasVersion(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := CreateCanvasVersion(context.Background(), r.Organization.ID.String(), uuid.New().String(), "")
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
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
