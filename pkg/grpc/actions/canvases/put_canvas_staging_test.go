package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func Test__PutCanvasStaging__StagesCanvasYAML(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	staged := baseline + "\n# staged edit\n"
	state, err := PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
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

func Test__PutCanvasStaging__RejectsReservedPath(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	_, err := PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: ".superplane/config", Content: []byte("nope")},
	})
	require.Error(t, err)
}
