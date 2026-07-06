package canvases

import (
	"context"
	"io"
	"strings"
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

func Test__CommitCanvasStaging__AppliesStagedCanvas(t *testing.T) {
	r, ctx, canvasID, versionID := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	canvasUUID := uuid.MustParse(canvasID)
	versionUUID := uuid.MustParse(versionID)

	canvas, err := models.FindCanvas(r.Organization.ID, canvasUUID)
	require.NoError(t, err)

	baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	staged := baseline + "\n# staged edit\n"

	_, err = PutCanvasStaging(ctx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(staged)},
	})
	require.NoError(t, err)

	resp, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "Update canvas", "", r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())
	require.NotNil(t, resp.GetVersion().GetMetadata())

	updatedCanvas, err := models.FindCanvas(r.Organization.ID, canvasUUID)
	require.NoError(t, err)
	assert.Equal(t, canvas.Name, updatedCanvas.Name)
	assert.NotEqual(t, versionUUID, updatedCanvas.LiveVersionID)

	hasStaging, err := models.HasStagedFilesForUser(database.DB(ctx), canvasUUID, r.User)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func Test__CommitCanvasStaging__IgnoresRenamedCanvasInYAML(t *testing.T) {
	r, ctx, canvasID, versionID := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	canvasUUID := uuid.MustParse(canvasID)
	canvas, err := models.FindCanvas(r.Organization.ID, canvasUUID)
	require.NoError(t, err)

	baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	renamed := strings.Replace(baseline, "name: "+canvas.Name, "name: "+canvas.Name+"-staged", 1)
	require.NotEqual(t, baseline, renamed)

	_, err = PutCanvasStaging(ctx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(renamed)},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "Rename attempt", "", r.AuthService)
	require.NoError(t, err)

	updatedCanvas, err := models.FindCanvas(r.Organization.ID, canvasUUID)
	require.NoError(t, err)
	assert.Equal(t, canvas.Name, updatedCanvas.Name)
}

func Test__CommitCanvasStaging__RequiresStagedChanges(t *testing.T) {
	r, ctx, canvasID, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	_, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "Nothing to commit", "", r.AuthService)
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

	content, found, deleted, err := ReadStagedRepositoryFile(ctx, database.DB(ctx), orgID, canvasID, "README.md")
	require.NoError(t, err)
	assert.True(t, found)
	assert.False(t, deleted)
	assert.Equal(t, "staged readme", content)

	resp, err := CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvasID, "Add readme", "", r.AuthService)
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
	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	baseline, err := ReadRepositorySpecFile(ownerCtx, orgID, canvasID, liveVersion.ID.String(), CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# owner staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	_, err = PutCanvasStaging(otherCtx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# other commit\n")},
	})
	require.NoError(t, err)

	_, err = CommitCanvasStaging(otherCtx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "Promote live", "", r.AuthService)
	require.NoError(t, err)

	_, err = CommitCanvasStaging(ownerCtx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, "Stale commit", "", r.AuthService)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, msg, "stale staging")
}
