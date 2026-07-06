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

func Test__DeleteCanvasStaging(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	fileReader := files.NewAppFileReader(database.DB(ctx), canvas, r.User)
	baselineReader, err := fileReader.ReadFromVersion(ctx, files.CanvasYAMLPath, canvas.LiveVersionID.String())
	require.NoError(t, err)
	baseline, err := io.ReadAll(baselineReader)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(string(baseline) + "\n# pending\n")},
	})
	require.NoError(t, err)

	state, err := DeleteCanvasStaging(ctx, orgID, canvas.ID.String(), nil)
	require.NoError(t, err)
	assert.False(t, state.GetHasStaging())

	_, err = fileReader.ReadFromStaging(ctx, files.CanvasYAMLPath)
	require.ErrorContains(t, err, "no staged file")
}
