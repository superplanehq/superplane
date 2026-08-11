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

func Test__ListCanvasVersionsPaginated(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("lists committed versions newest first", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

		liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
		require.NoError(t, err)

		baseline, err := ReadRepositorySpecFile(ctx, canvas, liveVersion, CanvasYAMLRepositoryPath)
		require.NoError(t, err)

		_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# first commit\n")},
		})
		require.NoError(t, err)

		firstCommit, err := CommitCanvasStaging(ctx, database.DB(t.Context()), r.GitProvider, nil, r.Encryptor, r.Registry, canvas, "First", "", r.AuthService)
		require.NoError(t, err)

		_, err = PutCanvasStaging(ctx, database.DB(t.Context()), canvas, []*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# second commit\n")},
		})
		require.NoError(t, err)

		secondCommit, err := CommitCanvasStaging(ctx, database.DB(t.Context()), r.GitProvider, nil, r.Encryptor, r.Registry, canvas, "Second", "", r.AuthService)
		require.NoError(t, err)

		response, err := ListCanvasVersionsPaginated(ctx, database.DB(t.Context()), canvas, 0, nil, "")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(response.GetVersions()), 2)
		assert.Equal(t, secondCommit.GetVersion().GetMetadata().GetId(), response.GetVersions()[0].GetId())
		assert.Equal(t, firstCommit.GetVersion().GetMetadata().GetId(), response.GetVersions()[1].GetId())
		assert.GreaterOrEqual(t, response.GetTotalCount(), uint32(2))
	})
}
