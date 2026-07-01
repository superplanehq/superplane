package canvases

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func setupStagingDraft(t *testing.T) (*support.ResourceRegistry, context.Context, string, string) {
	t.Helper()
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	created, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), "")
	require.NoError(t, err)

	return r, ctx, canvas.ID.String(), created.GetVersion().GetMetadata().GetId()
}

func TestStageRepositorySpecFileOperations(t *testing.T) {
	r, ctx, canvasID, versionID := setupStagingDraft(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	staged := baseline + "\n# staged edit\n"
	state, err := StageRepositorySpecFileOperations(ctx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(staged)},
	})
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Equal(t, []string{CanvasYAMLRepositoryPath}, state.GetStagedPaths())
	assert.Equal(t, versionID, state.GetBaseVersionId())

	// Effective staged read returns the staged content.
	effective, err := ReadRepositorySpecFileStaged(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, staged, effective)

	// Non-staged read still returns the materialized version row.
	committed, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, committed)
	assert.NotContains(t, committed, "# staged edit")
}

func TestDiscardCanvasStaging(t *testing.T) {
	r, ctx, canvasID, versionID := setupStagingDraft(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = StageRepositorySpecFileOperations(ctx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# pending\n")},
	})
	require.NoError(t, err)

	resp, err := DiscardCanvasStaging(ctx, orgID, canvasID, versionID, nil)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())

	// After discard the effective read falls back to the materialized version.
	effective, err := ReadRepositorySpecFileStaged(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, baseline, effective)
}

func TestApplyCanvasAutoLayout(t *testing.T) {
	r, ctx, canvasID, versionID := setupStagingDraft(t)
	orgID := r.Organization.ID.String()

	t.Run("nil layout -> invalid argument", func(t *testing.T) {
		_, err := ApplyCanvasAutoLayout(ctx, orgID, canvasID, versionID, nil)
		require.Error(t, err)
	})

	t.Run("lays out staged canvas and re-stages", func(t *testing.T) {
		resp, err := ApplyCanvasAutoLayout(ctx, orgID, canvasID, versionID, &pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
			Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
		})
		require.NoError(t, err)
		assert.True(t, resp.GetStagingSummary().GetHasStaging())
		assert.Contains(t, resp.GetStagingSummary().GetStagedPaths(), CanvasYAMLRepositoryPath)

		staged, err := ReadRepositorySpecFileStaged(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
		require.NoError(t, err)
		assert.Contains(t, staged, "nodes:")
	})
}

func TestCommitCanvasStagingAppliesStagedCanvas(t *testing.T) {
	r, ctx, canvasID, versionID := setupStagingDraft(t)
	orgID := r.Organization.ID.String()

	canvasUUID := uuid.MustParse(canvasID)
	versionUUID := uuid.MustParse(versionID)

	original, err := models.FindCanvasVersion(canvasUUID, versionUUID)
	require.NoError(t, err)

	baseline, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	renamed := strings.Replace(baseline, "name: "+original.Name, "name: "+original.Name+"-staged", 1)
	require.NotEqual(t, baseline, renamed, "expected canvas name to appear in materialized yaml")

	_, err = StageRepositorySpecFileOperations(ctx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(renamed)},
	})
	require.NoError(t, err)

	resp, err := CommitCanvasStaging(ctx, nil, nil, r.Encryptor, r.Registry, orgID, canvasID, versionID, "", "", "", testWebhookBaseURL, r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())
	assert.Equal(t, original.Name+"-staged", resp.GetVersion().GetMetadata().GetName())

	// Version row is updated and staging is cleared.
	updated, err := models.FindCanvasVersion(canvasUUID, versionUUID)
	require.NoError(t, err)
	assert.Equal(t, original.Name+"-staged", updated.Name)

	hasStaging, err := models.HasWorkflowStaging(updated.ID)
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func TestStageArbitraryRepositoryFile(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	created, err := CreateCanvasVersion(ctx, orgID, canvas.ID.String(), "")
	require.NoError(t, err)
	canvasID := canvas.ID.String()
	versionID := created.GetVersion().GetMetadata().GetId()

	// Staging an arbitrary (non-spec) repository file flips the staging state and
	// reports the path so the UI switches to Reset/Commit.
	state, err := StageRepositorySpecFileOperations(ctx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: "README.md", Content: []byte("staged readme")},
	})
	require.NoError(t, err)
	assert.True(t, state.GetHasStaging())
	assert.Contains(t, state.GetStagedPaths(), "README.md")

	// The staged content is readable before commit.
	content, found, deleted, err := ReadStagedRepositoryFile(ctx, orgID, canvasID, versionID, "README.md")
	require.NoError(t, err)
	assert.True(t, found)
	assert.False(t, deleted)
	assert.Equal(t, "staged readme", content)

	// Reserved SuperPlane paths cannot be staged.
	_, err = StageRepositorySpecFileOperations(ctx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: ".superplane/config", Content: []byte("nope")},
	})
	require.Error(t, err)

	// Commit durably writes the arbitrary file to git and clears staging.
	resp, err := CommitCanvasStaging(ctx, r.GitProvider, nil, r.Encryptor, r.Registry, orgID, canvasID, versionID, "", "", "", testWebhookBaseURL, r.AuthService)
	require.NoError(t, err)
	assert.False(t, resp.GetStagingSummary().GetHasStaging())
	assert.NotEmpty(t, resp.GetVersion().GetMetadata().GetCommitSha())

	reader, err := r.GitProvider.GetFile(ctx, repository.RepoID, "README.md", "")
	require.NoError(t, err)
	committed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "staged readme", string(committed))

	hasStaging, err := models.HasWorkflowStaging(uuid.MustParse(versionID))
	require.NoError(t, err)
	assert.False(t, hasStaging)
}

func TestStagedReadRequiresDraftOwner(t *testing.T) {
	r, ownerCtx, canvasID, versionID := setupStagingDraft(t)
	orgID := r.Organization.ID.String()

	baseline, err := ReadRepositorySpecFile(ownerCtx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = StageRepositorySpecFileOperations(ownerCtx, orgID, canvasID, versionID, []*pb.CanvasRepositoryFileOperation{
		{Path: CanvasYAMLRepositoryPath, Content: []byte(baseline + "\n# staged\n")},
	})
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	_, err = ReadRepositorySpecFileStaged(otherCtx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.Error(t, err)
}

func TestReadRepositorySpecFileStagedPublishedVersionReturnsCommitted(t *testing.T) {
	r, ctx, canvasID, versionID := setupStagingDraft(t)
	orgID := r.Organization.ID.String()

	committed, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)

	_, err = PublishCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.GitProvider,
		orgID,
		canvasID,
		versionID,
		testWebhookBaseURL,
		r.AuthService,
	)
	require.NoError(t, err)

	stagedRead, err := ReadRepositorySpecFileStaged(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	assert.Equal(t, committed, stagedRead)
}
