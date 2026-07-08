package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__GetCanvasStaging(t *testing.T) {
	r, ctx, canvas, version := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ctx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# pending\n")},
	})
	require.NoError(t, err)

	state, err := GetCanvasStaging(ctx, orgID, canvas.ID.String())
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Contains(t, state.GetStagedPaths(), CanvasYAMLRepositoryPath)
	require.NotNil(t, state.GetSpec())
}

func Test__GetCanvasStaging__StagedReadIsPerUser(t *testing.T) {
	r, ownerCtx, canvas, version := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ownerCtx, canvas, version, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, orgID, canvas.ID.String(), []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	otherRead, err := ReadStagedRepositorySpecFile(otherCtx, database.DB(otherCtx), orgID, canvas.ID.String(), liveVersion, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, otherRead)
}
