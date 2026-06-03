package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__PublishCanvas(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := PublishCanvas(
			context.Background(),
			r.GitProvider,
			r.Registry,
			r.Encryptor,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID.String(),
			uuid.New().String(),
			"drafts/user",
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("publish removes draft branch record and git branch", func(t *testing.T) {
		canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-removes-draft")
		canvasUUID := uuid.MustParse(canvasID)

		branchResp, err := CreateDraftBranch(ctx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, "Publish Draft")
		require.NoError(t, err)
		branchName := branchResp.GetBranch().GetBranchName()

		liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvasUUID)
		require.NoError(t, err)

		nodes := append([]models.Node(nil), liveVersion.Nodes...)
		for i := range nodes {
			if nodes[i].ID == "node-1" {
				nodes[i].Name = "Published Noop"
			}
		}

		canvasYAML, err := materialize.BuildCanvasYAML(
			liveVersion.Name,
			liveVersion.Description,
			nodes,
			liveVersion.Edges,
			liveVersion.ChangeManagementEnabled,
			liveVersion.EffectiveChangeRequestApprovers(),
		)
		require.NoError(t, err)

		consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(liveVersion)
		require.NoError(t, err)

		_, err = CommitCanvasRepositoryFiles(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Organization.ID.String(),
			canvasID,
			branchName,
			branchResp.GetBranch().GetTipSha(),
			"Update draft",
			[]*pb.CanvasRepositoryFileOperation{
				{Path: materialize.CanvasFileName, Content: canvasYAML},
				{Path: materialize.ConsoleFileName, Content: consoleYAML},
			},
		)
		require.NoError(t, err)

		_, err = models.FindDraftBranch(canvasUUID, branchName)
		require.NoError(t, err)

		_, err = PublishCanvas(
			ctx,
			r.GitProvider,
			r.Registry,
			r.Encryptor,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID.String(),
			canvasID,
			branchName,
		)
		require.NoError(t, err)

		branches, err := models.ListDraftBranchesForCanvas(canvasUUID)
		require.NoError(t, err)
		for _, branch := range branches {
			assert.NotEqual(t, branchName, branch.BranchName)
		}

		repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
		require.NoError(t, err)

		gitBranches, err := r.GitProvider.ListBranches(ctx, repository.RepoID, branchName)
		require.NoError(t, err)
		assert.NotContains(t, gitBranches, branchName)
	})
}
