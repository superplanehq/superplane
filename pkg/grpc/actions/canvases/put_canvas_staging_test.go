package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
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
