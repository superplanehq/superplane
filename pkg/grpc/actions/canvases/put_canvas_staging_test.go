package canvases

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/services/files"
)

func Test__PutCanvasStaging__StagesCanvasYAML(t *testing.T) {
	r, ctx, canvas, liveVersion := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	fileReader := files.NewAppFileReader(database.DB(ctx), canvas, r.User)
	reader, err := fileReader.ReadFromVersion(ctx, files.CanvasYAMLPath, liveVersion.ID.String())
	require.NoError(t, err)
	baseline, err := io.ReadAll(reader)
	require.NoError(t, err)

	staged := string(baseline) + "\n# staged edit\n"
	state, err := PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(staged)},
	})
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Equal(t, []string{files.CanvasYAMLPath}, state.GetStagedPaths())
	assert.Equal(t, liveVersion.ID.String(), state.GetBaseVersionId())

	effective, err := fileReader.ReadFromStaging(ctx, files.CanvasYAMLPath)
	require.NoError(t, err)
	assert.Equal(t, staged, effective)

	committed, err := fileReader.ReadFromVersion(ctx, files.CanvasYAMLPath, liveVersion.ID.String())
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
	require.ErrorContains(t, err, "reserved path")
}
