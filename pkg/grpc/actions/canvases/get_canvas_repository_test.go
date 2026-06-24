package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__RepositoryStateToProto(t *testing.T) {
	assert.Equal(t, pb.CanvasRepository_STATE_PENDING, repositoryStateToProto(models.RepositoryStatusPending))
	assert.Equal(t, pb.CanvasRepository_STATE_READY, repositoryStateToProto(models.RepositoryStatusReady))
	assert.Equal(t, pb.CanvasRepository_STATE_ERROR, repositoryStateToProto(models.RepositoryStatusError))
	assert.Equal(t, pb.CanvasRepository_STATE_UNSPECIFIED, repositoryStateToProto("unknown"))
}

func Test__GetCanvasRepository(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), "invalid-id")
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), uuid.New().String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("canvas exists, but repository does not -> creates pending repository", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		response, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Repository)
		assert.Equal(t, canvas.ID.String(), response.Repository.Metadata.CanvasId)
		assert.Equal(t, pb.CanvasRepository_STATE_PENDING, response.Repository.Status.State)
		assert.NotNil(t, response.Repository.Metadata.UpdatedAt)
	})

	t.Run("head lookup fails -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)
		_, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), canvas.ID.String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, code)
	})

	t.Run("repository in pending state -> returns metadata without head sha", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusPending, false)
		response, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Repository)
		assert.Equal(t, canvas.ID.String(), response.Repository.Metadata.CanvasId)
		assert.Equal(t, pb.CanvasRepository_STATE_PENDING, response.Repository.Status.State)
		assert.Empty(t, response.Repository.Status.HeadSha)
	})

	t.Run("repository in error state -> returns metadata without head sha", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusError, false)
		response, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Repository)
		assert.Equal(t, canvas.ID.String(), response.Repository.Metadata.CanvasId)
		assert.Equal(t, pb.CanvasRepository_STATE_ERROR, response.Repository.Status.State)
		assert.Empty(t, response.Repository.Status.HeadSha)
	})

	t.Run("returns repository metadata and status", func(t *testing.T) {
		canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		response, err := GetCanvasRepository(context.Background(), r.GitProvider, r.Organization.ID.String(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Repository)
		assert.Equal(t, canvas.ID.String(), response.Repository.Metadata.CanvasId)
		assert.Equal(t, pb.CanvasRepository_STATE_READY, response.Repository.Status.State)
		assert.NotEmpty(t, response.Repository.Status.HeadSha)
		assert.NotNil(t, response.Repository.Metadata.UpdatedAt)
		assert.Equal(t, repository.UpdatedAt.Unix(), response.Repository.Metadata.UpdatedAt.AsTime().Unix())
	})
}
