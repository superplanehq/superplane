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
		canvasID := createCanvasWithNoopNode(ctx, t, r, "commit-files-author")
		created, err := CreateCanvasVersion(ctx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, "")
		require.NoError(t, err)
		versionID := created.GetVersion().GetMetadata().GetId()

		version, err := models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(versionID))
		require.NoError(t, err)
		require.NotNil(t, version.BranchName)
		draftBranch := *version.BranchName

		repository, err := models.FindRepository(r.Organization.ID, uuid.MustParse(canvasID))
		require.NoError(t, err)
		headSHA, err := r.GitProvider.Head(ctx, repository.RepoID, draftBranch)
		require.NoError(t, err)

		// Commits target the draft branch; arbitrary repository files land there.
		response, err := commitCanvasRepositoryFilesForTest(
			ctx,
			r,
			r.Organization.ID.String(),
			canvasID,
			versionID,
			headSHA,
			"add readme",
			[]*pb.CanvasRepositoryFileOperation{
				{Path: "README.md", Content: []byte("hello world")},
			},
		)
		require.NoError(t, err)
		assert.NotEmpty(t, response.CommitSha)

		reader, err := r.GitProvider.GetFile(ctx, repository.RepoID, "README.md", draftBranch)
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		assert.Equal(t, "hello world", string(content))

		files, err := r.GitProvider.ListFiles(ctx, repository.RepoID, draftBranch)
		require.NoError(t, err)
		assert.Contains(t, files, "README.md")
	})
}

func Test__CommitCanvasRepositoryFiles_DiscardsStaging(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, draftVersionID := createGitCanvasWithDraft(ctx, t, r, "commit-discards-staging")
	draftVersionUUID := uuid.MustParse(draftVersionID)

	_, err := models.UpsertWorkflowStagingPath(
		draftVersionUUID,
		r.Organization.ID,
		ConsoleYAMLRepositoryPath,
		"stale staged content",
		"",
		&r.User,
	)
	require.NoError(t, err)

	hasStaging, err := models.HasWorkflowStaging(draftVersionUUID)
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
		draftVersionID,
		"",
		"Update canvas.yaml",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(yamlText)},
		},
	)
	require.NoError(t, err)

	hasStaging, err = models.HasWorkflowStaging(draftVersionUUID)
	require.NoError(t, err)
	assert.False(t, hasStaging, "direct commit should discard existing staged changes")
}
