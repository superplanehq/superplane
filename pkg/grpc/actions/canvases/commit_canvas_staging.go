package canvases

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// CommitCanvasStaging commits all staged repository files to the draft branch in
// git, materializes the resulting commit into workflow_versions, and clears staging.
func CommitCanvasStaging(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.CommitCanvasStagingResponse, error) {
	_ = usageService
	_ = encryptor
	_ = webhookBaseURL
	_ = authService

	canvas, version, userUUID, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	if version.BranchName == nil || strings.TrimSpace(*version.BranchName) == "" {
		return nil, status.Error(codes.FailedPrecondition, "draft branch is required")
	}

	rows, err := models.ListWorkflowStaging(version.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load staging: %v", err)
	}

	// Committing with nothing staged is a no-op: callers such as the publish
	// flow flush staging unconditionally before promoting a draft, so an empty
	// commit must succeed instead of failing the surrounding operation.
	if len(rows) == 0 {
		summary, _, err := stagingSummaryForVersion(version.ID)
		if err != nil {
			return nil, err
		}

		return &pb.CommitCanvasStagingResponse{
			Version:        SerializeCanvasVersionMetadata(version, organizationID),
			StagingSummary: summary,
		}, nil
	}

	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	draftBranch := strings.TrimSpace(*version.BranchName)
	expectedHeadSHA := resolveStagingExpectedHeadSHA(ctx, gitProvider, repository.RepoID, draftBranch, version, rows)

	gitOps := stagedGitOperations(rows)
	operations := make([]gitprovider.FileOperation, 0, len(gitOps))
	for _, operation := range gitOps {
		content := operation.GetContent()
		var reader io.Reader
		if !operation.GetDelete() {
			reader = bytes.NewReader(content)
		}

		operations = append(operations, gitprovider.FileOperation{
			Path:      operation.GetPath(),
			Content:   reader,
			SizeBytes: int64(len(content)),
			Delete:    operation.GetDelete(),
		})
	}

	commitSHA, err := gitProvider.Commit(ctx, repository.RepoID, gitprovider.CommitOptions{
		Branch:          draftBranch,
		BaseBranch:      draftBranch,
		ExpectedHeadSHA: expectedHeadSHA,
		Message:         "Commit staged changes",
		Operations:      operations,
		Author: gitprovider.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit staged files: %v", err)
	}

	var pending *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		registered, registerErr := materialize.RegisterPendingDraftVersion(
			tx,
			canvas.ID,
			draftBranch,
			commitSHA,
			materialize.RegisterPendingDraftOptions{CreatedBy: version.OwnerID},
		)
		if registerErr != nil {
			return registerErr
		}
		pending = registered

		// A commit re-materializes the draft from git, so any staged DB edits
		// for this version are now stale and must be cleared.
		return models.DiscardWorkflowStagingInTransaction(tx, version.ID, nil)
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit staged changes: %v", err)
	}

	// Worker-authoritative materialization: the commit and the pending row are
	// persisted, so hand the snapshot load to the worker (inline in tests).
	if err := materialize.RequestBranchMaterialization(ctx, canvas.ID, draftBranch, commitSHA, &userUUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to request draft materialization: %v", err)
	}

	summary, _, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	// Optimistic response: returns the pending version metadata at the new commit;
	// nodes/edges are materialized asynchronously by the worker.
	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersionMetadata(pending, organizationID),
		StagingSummary: summary,
	}, nil
}

func resolveStagingExpectedHeadSHA(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	repoID string,
	draftBranch string,
	version *models.CanvasVersion,
	rows []models.WorkflowStaging,
) string {
	for _, row := range rows {
		if strings.TrimSpace(row.BaseHeadSHA) != "" {
			return strings.TrimSpace(row.BaseHeadSHA)
		}
	}

	if version != nil && strings.TrimSpace(version.CommitSHA) != "" {
		return strings.TrimSpace(version.CommitSHA)
	}

	headSHA, err := gitProvider.Head(ctx, repoID, draftBranch)
	if err != nil {
		return ""
	}

	return headSHA
}

func stagedGitOperations(rows []models.WorkflowStaging) []*pb.CanvasRepositoryFileOperation {
	ops := make([]*pb.CanvasRepositoryFileOperation, 0, len(rows))
	for _, row := range rows {
		ops = append(ops, &pb.CanvasRepositoryFileOperation{
			Path:    row.Path,
			Content: []byte(row.Content),
			Delete:  row.Deleted,
		})
	}
	return ops
}
