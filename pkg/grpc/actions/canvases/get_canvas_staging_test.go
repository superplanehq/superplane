package canvases

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/services/files"
	"github.com/superplanehq/superplane/test/support"
)

func Test__GetCanvasStaging(t *testing.T) {
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

	state, err := GetCanvasStaging(ctx, orgID, canvas.ID.String())
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Contains(t, state.GetStagedPaths(), files.CanvasYAMLPath)
}

func Test__GetCanvasStaging__StagedReadIsPerUser(t *testing.T) {
	r, ownerCtx, canvas, _ := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	fileReader := files.NewAppFileReader(database.DB(ownerCtx), canvas, r.User)
	baselineReader, err := fileReader.ReadFromVersion(ownerCtx, files.CanvasYAMLPath, canvas.LiveVersionID.String())
	require.NoError(t, err)
	baseline, err := io.ReadAll(baselineReader)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: files.CanvasYAMLPath, Content: []byte(string(baseline) + "\n# staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	_, err = fileReader.ReadFromStaging(otherCtx, files.CanvasYAMLPath)
	require.ErrorContains(t, err, "no staged file")
}
