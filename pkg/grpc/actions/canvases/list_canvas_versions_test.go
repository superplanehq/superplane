package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__ListCanvasVersionsPaginated(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), "invalid-id", 0, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, "invalid-id", uuid.New().String(), 0, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), uuid.New().String(), 0, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("lists committed versions newest first", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		orgID := r.Organization.ID.String()

		liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
		require.NoError(t, err)

		baseline, err := ReadRepositorySpecFile(ctx, canvas, liveVersion, CanvasYAMLRepositoryPath)
		require.NoError(t, err)

		_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# first commit\n")},
		})
		require.NoError(t, err)

		firstCommit, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "First", "", r.AuthService)
		require.NoError(t, err)

		_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# second commit\n")},
		})
		require.NoError(t, err)

		secondCommit, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Second", "", r.AuthService)
		require.NoError(t, err)

		response, err := ListCanvasVersionsPaginated(ctx, orgID, canvas.ID.String(), 0, nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(response.GetVersions()), 2)
		assert.Equal(t, secondCommit.GetVersion().GetMetadata().GetId(), response.GetVersions()[0].GetMetadata().GetId())
		assert.Equal(t, firstCommit.GetVersion().GetMetadata().GetId(), response.GetVersions()[1].GetMetadata().GetId())
		assert.GreaterOrEqual(t, response.GetTotalCount(), uint32(2))
	})

	// Regression guard for #5851: the versions endpoint must never return
	// codes.Internal (HTTP 500) because a canvas holds a stale reference to a
	// version row that no longer exists. History listing does not depend on
	// LiveVersionID, so a dangling pointer must not break the response.
	t.Run("stale live version reference -> no internal error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		orgID := r.Organization.ID.String()

		staleVersionID := uuid.New()
		err := database.Conn().
			Model(&models.Canvas{}).
			Where("id = ?", canvas.ID).
			Update("live_version_id", staleVersionID).
			Error
		require.NoError(t, err)

		_, err = ListCanvasVersionsPaginated(ctx, orgID, canvas.ID.String(), 0, nil)
		require.NoError(t, err)
	})
}
