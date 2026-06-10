package canvases

import (
	"bytes"
	"context"
	"io"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CommitCanvasStaging durably persists staged repository files for a draft
// version. Spec files (canvas.yaml/console.yaml) are parsed into the
// workflow_versions row (the same validated path as the interim repository-files
// commit); all other staged files are committed to the canvas git repository.
// Staging rows are cleared afterwards.
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
	canvas, version, userUUID, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	rows, err := models.ListWorkflowStaging(version.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load staging: %v", err)
	}

	specOps, gitOps := stagedCommitOperations(rows)

	// Git files and spec rows live in separate stores, so they cannot share a
	// transaction. Commit git files first: if the git commit fails (for example
	// on a missing repository), the request returns before any spec change is
	// written, keeping the version row consistent with the failed commit.
	if len(gitOps) > 0 {
		if err := commitStagedGitFiles(ctx, gitProvider, canvas, organizationID, userUUID.String(), gitOps); err != nil {
			return nil, err
		}
	}

	if len(specOps) > 0 {
		if err := ApplyRepositorySpecFileOperations(
			ctx,
			usageService,
			encryptor,
			registry,
			organizationID,
			canvasID,
			versionID,
			webhookBaseURL,
			authService,
			specOps,
		); err != nil {
			return nil, err
		}
	}

	if err := models.DiscardWorkflowStaging(version.ID, nil); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clear staging: %v", err)
	}

	committed, err := models.FindCanvasVersion(version.WorkflowID, version.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reload version: %v", err)
	}

	state, _, err := stagingStateForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	return &pb.CommitCanvasStagingResponse{
		Version:      SerializeCanvasVersionMetadata(committed, organizationID),
		StagingState: state,
	}, nil
}

// stagedCommitOperations splits staging rows into spec operations (canvas.yaml /
// console.yaml, applied to the version row) and git operations (every other
// path, committed to the repository). Deleted spec rows are skipped — a draft
// keeps its committed spec when a staged spec file is removed — while deleted
// git rows become repository delete operations.
func stagedCommitOperations(rows []models.WorkflowStaging) (specOps, gitOps []*pb.CanvasRepositoryFileOperation) {
	specContentByPath := map[string]string{}
	for _, row := range rows {
		if IsRepositorySpecFilePath(row.Path) {
			if row.Deleted {
				continue
			}
			specContentByPath[row.Path] = row.Content
			continue
		}

		gitOps = append(gitOps, &pb.CanvasRepositoryFileOperation{
			Path:    row.Path,
			Content: []byte(row.Content),
			Delete:  row.Deleted,
		})
	}

	for _, path := range []string{CanvasYAMLRepositoryPath, ConsoleYAMLRepositoryPath} {
		content, ok := specContentByPath[path]
		if !ok {
			continue
		}
		specOps = append(specOps, &pb.CanvasRepositoryFileOperation{
			Path:    path,
			Content: []byte(content),
		})
	}

	return specOps, gitOps
}

// commitStagedGitFiles commits the non-spec staged files to the canvas git
// repository, authored by the committing user.
func commitStagedGitFiles(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	canvas *models.Canvas,
	organizationID string,
	userID string,
	gitOps []*pb.CanvasRepositoryFileOperation,
) error {
	if gitProvider == nil {
		return status.Error(codes.FailedPrecondition, "git provider is not configured")
	}

	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err != nil {
		return status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	// Commit on top of the current branch head. Staging does not track a head
	// SHA per file, so resolve it just before committing.
	headSHA, err := gitProvider.Head(ctx, repository.RepoID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to resolve repository head: %v", err)
	}

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

	_, err = gitProvider.Commit(ctx, repository.RepoID, gitprovider.CommitOptions{
		Branch:          "main",
		BaseBranch:      "main",
		ExpectedHeadSHA: headSHA,
		Message:         "Update files",
		Operations:      operations,
		Author: gitprovider.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to commit repository files: %v", err)
	}

	return nil
}
