package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func Test__DeleteCanvasStaging(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# pending\n")},
	})
	require.NoError(t, err)

	state, err := DeleteCanvasStaging(ctx, orgID, canvas.ID.String(), nil)
	require.NoError(t, err)
	assert.False(t, state.GetHasStaging())

	effective, err := ReadRepositorySpecFileStaged(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, effective)
}
