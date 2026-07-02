package canvases

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
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
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
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
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
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
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
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

		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, code)
		assert.Contains(t, msg, "failed to commit repository files")
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
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("commits files with authenticated user as author", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		headSHA, err := r.GitProvider.Head(ctx, repository.RepoID, "")
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

		reader, err := r.GitProvider.GetFile(ctx, repository.RepoID, "README.md", "")
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		assert.Equal(t, "hello world", string(content))

		files, err := r.GitProvider.ListFiles(ctx, repository.RepoID, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"README.md"}, files)
	})
}

func Test__CommitCanvasRepositoryFiles_RejectsDirectSpecCommitAndPreservesStaging(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	_, err = models.UpsertWorkflowStagingPath(
		nil,
		canvas.ID,
		r.User,
		liveVersion.ID,
		r.Organization.ID,
		ConsoleYAMLRepositoryPath,
		"stale staged content",
		&r.User,
	)
	require.NoError(t, err)

	hasStaging, err := models.HasWorkflowStagingForUser(nil, canvas.ID, r.User)
	require.NoError(t, err)
	require.True(t, hasStaging)

	yamlText := `apiVersion: v1
kind: Canvas
metadata:
  name: ` + canvas.Name + `
spec:
  nodes:
    - id: s
      name: Start
      type: TYPE_TRIGGER
      component: start
  edges: []
`

	_, err = commitCanvasRepositoryFilesForTest(
		ctx,
		r,
		r.Organization.ID.String(),
		canvas.ID.String(),
		liveVersion.ID.String(),
		"",
		"Update canvas.yaml",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(yamlText)},
		},
	)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, msg, "direct version updates are not supported")

	hasStaging, err = models.HasWorkflowStagingForUser(nil, canvas.ID, r.User)
	require.NoError(t, err)
	assert.True(t, hasStaging, "direct spec commit should leave staged changes intact")
}
