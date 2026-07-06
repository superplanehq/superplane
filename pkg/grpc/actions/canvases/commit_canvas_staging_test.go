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
	r, ctx, canvas, liveVersion := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	fileReader := files.NewAppFileReader(database.DB(ctx), canvas, r.User)
	baselineReader, err := fileReader.ReadFromVersion(ctx, files.CanvasYAMLPath, liveVersion.ID.String())
	require.NoError(t, err)
	baseline, err := io.ReadAll(baselineReader)
	require.NoError(t, err)

	staged := string(baseline) + "\n# staged edit\n"

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(staged)},
	})
	require.NoError(t, err)

	resp, err := CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Update canvas", "", r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())
	require.NotNil(t, resp.GetVersion().GetMetadata())

	updatedCanvas, err := models.FindCanvasInTransaction(database.DB(ctx), r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, canvas.Name, updatedCanvas.Name)
	assert.NotEqual(t, liveVersion.ID, updatedCanvas.LiveVersionID)

	hasStaging, err := models.HasStagedFilesForUser(database.DB(ctx), canvas.ID, r.User)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func Test__CommitCanvasStaging__IgnoresRenamedCanvasInYAML(t *testing.T) {
	r, ctx, canvas, liveVersion := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	fileReader := files.NewAppFileReader(database.DB(ctx), canvas, r.User)
	baselineReader, err := fileReader.ReadFromVersion(ctx, files.CanvasYAMLPath, liveVersion.ID.String())
	require.NoError(t, err)
	baseline, err := io.ReadAll(baselineReader)
	require.NoError(t, err)

	renamed := strings.Replace(string(baseline), "name: "+canvas.Name, "name: "+canvas.Name+"-staged", 1)
	require.NotEqual(t, string(baseline), renamed)

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(renamed)},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Rename attempt", "", r.AuthService)
	require.NoError(t, err)

	updatedCanvas, err := models.FindCanvasInTransaction(database.DB(ctx), r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, canvas.Name, updatedCanvas.Name)
}

func Test__CommitCanvasStaging__RequiresStagedChanges(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	_, err := CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Nothing to commit", "", r.AuthService)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, msg, "no staged changes")
}

func Test__CommitCanvasStaging__ArbitraryFileCommitsToGit(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	_, err := PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: "README.md", Content: []byte("staged readme")},
	})
	require.NoError(t, err)

	fileReader := files.NewAppFileReader(database.DB(ctx), canvas, r.User)
	contentReader, err := fileReader.ReadFromStaging(ctx, "README.md")
	require.NoError(t, err)
	content, err := io.ReadAll(contentReader)
	require.NoError(t, err)
	assert.Equal(t, "staged readme", string(content))

	resp, err := CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvas.ID.String(), "Add readme", "", r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())

	reader, err := r.GitProvider.GetFile(ctx, repository.RepoID, "README.md", "")
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
	canvasID := canvas.ID.String()

	fileReader := files.NewAppFileReader(database.DB(ownerCtx), canvas, r.User)
	baselineReader, err := fileReader.ReadFromVersion(ownerCtx, files.CanvasYAMLPath, canvas.LiveVersionID.String())
	require.NoError(t, err)
	baseline, err := io.ReadAll(baselineReader)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(string(baseline) + "\n# owner staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	_, err = PutCanvasStaging(otherCtx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(string(baseline) + "\n# other commit\n")},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(otherCtx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvasID, "Promote live", "", r.AuthService)
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ownerCtx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvasID, "Stale commit", "", r.AuthService)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, msg, "stale staging")
}
