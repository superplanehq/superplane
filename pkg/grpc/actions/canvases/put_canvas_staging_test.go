package canvases

import (
	"context"
	"testing"

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

func Test__PutCanvasStaging__StagesCanvasYAML(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	defer r.Close()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	staged := baseline + "\n# staged edit\n"
	state, err := PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(staged)},
	})
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Equal(t, []string{CanvasYAMLRepositoryPath}, state.GetStagedPaths())
	assert.Equal(t, version.ID.String(), state.GetBaseVersionId())

	effective, err := ReadRepositorySpecFileStaged(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, staged, effective)

	committed, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, committed)
	assert.NotContains(t, committed, "# staged edit")
}

func Test__PutCanvasStaging__RejectsNonSpecFile(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	defer r.Close()

	_, err := PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: "README.md", Content: []byte("staged readme")},
	})
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Contains(t, msg, "only canvas.yaml and console.yaml")
}

func Test__PutCanvasStaging__RejectsSpecFileDelete(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	defer r.Close()

	_, err := PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Delete: true},
	})
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Contains(t, msg, "cannot be deleted")
}

func Test__PutCanvasStaging__RejectsMixedBatchWithoutWriting(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	defer r.Close()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# staged edit\n")},
		{Path: "README.md", Content: []byte("staged readme")},
	})
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Contains(t, msg, "only canvas.yaml and console.yaml")

	hasStaging, err := models.HasStagedFilesForUser(database.DB(ctx), canvas.ID, r.User)
	require.NoError(t, err)
	assert.False(t, hasStaging)

	effective, err := ReadRepositorySpecFileStaged(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, effective)
}

func Test__PutCanvasStagingReplacingStale__ReplacesStaleDraft(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	ownerCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	baseline, err := ReadRepositorySpecFile(ownerCtx, canvas, liveVersion, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# owner staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
	_, err = PutCanvasStaging(otherCtx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# other commit\n")},
	})
	require.NoError(t, err)
	_, err = CommitCanvasStaging(otherCtx, database.DB(t.Context()), r.Encryptor, r.Registry, canvas, "Promote live", "", r.AuthService)
	require.NoError(t, err)

	replacement := baseline + "\n# agent edit\n"
	_, err = PutCanvasStagingReplacingStale(ownerCtx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(replacement)},
	})
	require.NoError(t, err)

	liveVersion, err = models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)
	rows, err := models.ListStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, liveVersion.ID, rows[0].BaseVersionID)
	assert.Equal(t, replacement, rows[0].Content)
}

func Test__PutCanvasStagingReplacingStale__KeepsCurrentDraft(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	defer r.Close()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	current := baseline + "\n# current draft\n"
	_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(current)},
	})
	require.NoError(t, err)

	_, err = PutCanvasStagingReplacingStale(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# agent edit\n")},
	})
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Equal(t, CurrentStagingCannotBeDiscardedMessage, msg)

	rows, err := models.ListStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, current, rows[0].Content)
}

func Test__PutCanvasStagingMatchingCanvas__UpdatesWhenContentMatches(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	defer r.Close()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	current := baseline + "\n# current draft\n"
	_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(current)},
	})
	require.NoError(t, err)

	updated := current + "model: opus\n"
	_, err = PutCanvasStagingMatchingCanvas(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(updated)},
	}, current)
	require.NoError(t, err)

	rows, err := models.ListStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, updated, rows[0].Content)
}

func Test__PutCanvasStagingMatchingCanvas__RejectsWhenContentChanges(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	defer r.Close()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	current := baseline + "\n# current draft\n"
	_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(current)},
	})
	require.NoError(t, err)

	_, err = PutCanvasStagingMatchingCanvas(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(current + "model: opus\n")},
	}, baseline)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Equal(t, StagedCanvasChangedMessage, msg)

	rows, err := models.ListStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, current, rows[0].Content)

	_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(current + "model: opus\n")},
	})
	require.NoError(t, err)

	rows, err = models.ListStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, current+"model: opus\n", rows[0].Content)
}
