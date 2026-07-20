package canvases

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/services/files"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__CommitCanvasStaging__AppliesStagedCanvas(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	staged := baseline + "\n# staged edit\n"

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(staged)},
	})
	require.NoError(t, err)

	resp, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Update canvas", "", r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())
	require.NotNil(t, resp.GetVersion().GetMetadata())

	updatedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, canvas.Name, updatedCanvas.Name)
	assert.NotEqual(t, version.ID, updatedCanvas.LiveVersionID)

	hasStaging, err := models.HasStagedFilesForUser(database.DB(ctx), canvas.ID, r.User)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func Test__CommitCanvasStaging__IgnoresRenamedCanvasInYAML(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	renamed := strings.Replace(baseline, "name: "+canvas.Name, "name: "+canvas.Name+"-staged", 1)
	require.NotEqual(t, baseline, renamed)

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(renamed)},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Rename attempt", "", r.AuthService)
	require.NoError(t, err)

	updatedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, canvas.Name, updatedCanvas.Name)
}

func Test__CommitCanvasStaging__RejectsInvalidConsoleYAML(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	_, err := PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: ConsoleYAMLRepositoryPath, Content: []byte("just a scalar, not an object")},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Bad console", "", r.AuthService)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Contains(t, msg, "invalid console yaml")
}

func Test__CommitCanvasStaging__RequiresStagedChanges(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	_, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Nothing to commit", "", r.AuthService)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, msg, "no staged changes")
}

func Test__CommitCanvasStaging__StageArbitraryRepositoryFileCommitsToGit(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	canvasID := canvas.ID.String()

	_, err := PutCanvasStaging(ctx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: "README.md", Content: []byte("staged readme")},
	})
	require.NoError(t, err)

	fileReader := files.NewAppFileReader(database.DB(ctx), r.GitProvider, canvas, r.User)
	reader, err := fileReader.ReadFromStaging(ctx, "README.md")
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "staged readme", string(content))

	resp, err := CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvasID, "Add readme", "", r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())

	reader, err = r.GitProvider.GetFile(ctx, repository.RepoID, "README.md", "")
	require.NoError(t, err)
	committed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "staged readme", string(committed))

	hasStaging, err := models.HasStagedFilesForUser(database.DB(ctx), canvas.ID, r.User)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func Test__CommitCanvasStaging__RejectsStaleStaging(t *testing.T) {
	r := support.Setup(t)
	ownerCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	baseline, err := ReadRepositorySpecFile(ownerCtx, canvas, liveVersion, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# owner staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	_, err = PutCanvasStaging(otherCtx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# other commit\n")},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(otherCtx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Promote live", "", r.AuthService)
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ownerCtx, nil, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Stale commit", "", r.AuthService)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, msg, "stale staging")
}
