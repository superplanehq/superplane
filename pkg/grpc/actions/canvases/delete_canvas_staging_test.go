package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func Test__DeleteCanvasStaging(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	defer r.Close()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# pending\n")},
	})
	require.NoError(t, err)

	state, err := DeleteCanvasStaging(ctx, database.DB(t.Context()), canvas, nil)
	require.NoError(t, err)
	assert.False(t, state.GetHasStaging())

	effective, err := ReadRepositorySpecFileStaged(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, effective)
}
