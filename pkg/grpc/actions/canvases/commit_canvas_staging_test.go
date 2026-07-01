package canvases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type createBranchFailGitProvider struct {
	*inmemory.Provider
}

func (p *createBranchFailGitProvider) CreateBranch(_ context.Context, _, _, _ string) error {
	return errors.New("git branch create failed")
}

func TestCommitCanvasStagingRollsBackDatabaseWhenGitBranchCreateFails(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	created, err := CreateCanvasVersion(ctx, orgID, canvas.ID.String(), "")
	require.NoError(t, err)

	canvasID := canvas.ID.String()
	versionID := created.GetVersion().GetMetadata().GetId()

	_, err = StageRepositorySpecFileOperations(ctx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: "README.md", Content: []byte("pending")},
	})
	require.NoError(t, err)

	failingGit := &createBranchFailGitProvider{Provider: inmemory.NewProvider()}
	var _ gitprovider.Provider = failingGit

	_, err = CommitCanvasStaging(
		ctx,
		failingGit,
		nil,
		r.Encryptor,
		r.Registry,
		orgID,
		canvasID,
		versionID,
		models.CanvasGitBranchMain,
		"Try new branch",
		"feat/rollback-test",
		testWebhookBaseURL,
		r.AuthService,
	)
	require.Error(t, err)

	_, err = models.FindWorkflowBranch(database.Conn(), canvas.ID, "feat/rollback-test")
	require.Error(t, err)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}
