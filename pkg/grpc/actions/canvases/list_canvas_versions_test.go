package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
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
		canvasID := canvas.ID.String()
		orgID := r.Organization.ID.String()

		liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
		require.NoError(t, err)

		baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, liveVersion.ID.String(), CanvasYAMLRepositoryPath)
		require.NoError(t, err)

		_, err = PutCanvasStaging(ctx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# first commit\n")},
		})
		require.NoError(t, err)

		firstCommit, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "First", "", r.AuthService)
		require.NoError(t, err)

		_, err = PutCanvasStaging(ctx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# second commit\n")},
		})
		require.NoError(t, err)

		secondCommit, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "Second", "", r.AuthService)
		require.NoError(t, err)

		response, err := ListCanvasVersionsPaginated(ctx, orgID, canvasID, 0, nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(response.GetVersions()), 2)
		assert.Equal(t, secondCommit.GetVersion().GetMetadata().GetId(), response.GetVersions()[0].GetMetadata().GetId())
		assert.Equal(t, firstCommit.GetVersion().GetMetadata().GetId(), response.GetVersions()[1].GetMetadata().GetId())
		assert.GreaterOrEqual(t, response.GetTotalCount(), uint32(2))
	})
}
