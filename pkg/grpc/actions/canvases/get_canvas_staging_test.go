package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__GetCanvasStaging(t *testing.T) {
	r, ctx, canvasID, versionID := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ctx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# pending\n")},
	})
	require.NoError(t, err)

	state, err := GetCanvasStaging(ctx, orgID, canvasID)
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Contains(t, state.GetStagedPaths(), CanvasYAMLRepositoryPath)
}

func Test__GetCanvasStaging__StagedReadIsPerUser(t *testing.T) {
	r, ownerCtx, canvasID, versionID := setupLiveCanvasStaging(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ownerCtx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PutCanvasStaging(ownerCtx, orgID, canvasID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	liveVersion, err := models.FindLiveCanvasVersion(uuid.MustParse(canvasID))
	require.NoError(t, err)

	otherRead, err := ReadStagedRepositorySpecFile(otherCtx, database.DB(otherCtx), orgID, canvasID, liveVersion, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, otherRead)
}
