package canvases

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// commitCanvasRepositoryFilesForTest wraps CommitCanvasRepositoryFiles with the
// shared services from the test ResourceRegistry so each case only has to supply
// the request-specific arguments.
func commitCanvasRepositoryFilesForTest(
	ctx context.Context,
	r *support.ResourceRegistry,
	organizationID string,
	canvasID string,
	versionID string,
	expectedHeadSha string,
	message string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.CommitCanvasRepositoryFilesResponse, error) {
	return CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		nil,
		r.Encryptor,
		r.Registry,
		organizationID,
		canvasID,
		versionID,
		expectedHeadSha,
		message,
		operations,
		nil,
		"",
		r.AuthService,
	)
}

func Test__CommitCanvasRepositoryFiles(t *testing.T) {
	r := support.Setup(t)

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := commitCanvasRepositoryFilesForTest(
			context.Background(),
			r,
			r.Organization.ID.String(),
			uuid.New().String(),
			"",
			"abc123",
			"commit message",
			nil,
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := commitCanvasRepositoryFilesForTest(
			ctx,
			r,
			r.Organization.ID.String(),
			"invalid-id",
			"",
			"abc123",
			"commit message",
			nil,
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("repository missing -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := commitCanvasRepositoryFilesForTest(
			ctx,
			r,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			"abc123",
			"commit message",
			[]*pb.CanvasRepositoryFileOperation{
				{Path: "README.md", Content: []byte("hello")},
			},
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("commit fails -> propagates error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)

		_, err := commitCanvasRepositoryFilesForTest(
			ctx,
			r,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			"stale-head",
			"commit message",
			[]*pb.CanvasRepositoryFileOperation{
				{Path: "README.md", Content: []byte("hello")},
			},
		)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, s.Code())
		assert.Contains(t, s.Message(), "failed to commit repository files")
	})

	t.Run("canvas from different organization -> not found", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		otherOrg := support.CreateOrganization(t, r, r.User)

		_, err := commitCanvasRepositoryFilesForTest(
			ctx,
			r,
			otherOrg.ID.String(),
			canvas.ID.String(),
			"",
			"abc123",
			"commit message",
			nil,
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("commits files with authenticated user as author", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		headSHA, err := r.GitProvider.Head(ctx, repository.RepoID)
		require.NoError(t, err)

		response, err := commitCanvasRepositoryFilesForTest(
			ctx,
			r,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			headSHA,
			"add readme",
			[]*pb.CanvasRepositoryFileOperation{
				{Path: "README.md", Content: []byte("hello world")},
				{Path: "old.txt", Delete: true},
			},
		)
		require.NoError(t, err)
		assert.NotEmpty(t, response.CommitSha)

		reader, err := r.GitProvider.GetFile(ctx, repository.RepoID, "README.md")
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		assert.Equal(t, "hello world", string(content))

		files, err := r.GitProvider.ListFiles(ctx, repository.RepoID)
		require.NoError(t, err)
		assert.Equal(t, []string{"README.md"}, files)
	})
}
