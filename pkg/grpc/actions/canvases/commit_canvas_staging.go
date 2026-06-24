package canvases

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"io"
	"strings"
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
	commitMessages ...string,
) (*pb.CommitCanvasStagingResponse, error) {
	canvas, version, userUUID, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	rows, err := models.ListWorkflowStaging(version.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	specOps, gitOps := stagedCommitOperations(rows)

	var gitRevertOps []gitprovider.FileOperation
	if len(gitOps) > 0 {
		var snapshotErr error
		gitRevertOps, snapshotErr = snapshotGitFilesBeforeCommit(ctx, gitProvider, canvas, gitOps)
		if snapshotErr != nil {
			return nil, snapshotErr
		}

		if err := commitStagedGitFiles(ctx, gitProvider, canvas, organizationID, userUUID.String(), resolvedStagingCommitMessage(commitMessages...), gitOps); err != nil {
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
			nil,
			false,
			specOps,
		); err != nil {
			if len(gitRevertOps) > 0 {
				if revertErr := revertGitFileCommit(ctx, gitProvider, canvas, organizationID, userUUID.String(), gitRevertOps); revertErr != nil {
					log.Errorf(
						"failed to revert git commit after spec apply failure for canvas %s version %s: %v",
						canvasID,
						versionID,
						revertErr,
					)
				}
			}
			return nil, err
		}
	}

	if err := models.DiscardWorkflowStaging(version.ID, nil); err != nil {
		return nil, grpcerrors.Internal(err, "failed to clear staging")
	}

	committed, err := models.FindCanvasVersion(version.WorkflowID, version.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to reload version")
	}

	state, _, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersionMetadata(committed, organizationID, nil),
		StagingSummary: state,
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
	message string,
	gitOps []*pb.CanvasRepositoryFileOperation,
) error {
	if gitProvider == nil {
		return grpcerrors.FailedPrecondition(nil, "git provider is not configured")
	}

	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err != nil {
		return grpcerrors.NotFound(err, "repository not found")
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return grpcerrors.Internal(err, "failed to find user")
	}

	// Commit on top of the current branch head. Staging does not track a head
	// SHA per file, so resolve it just before committing.
	headSHA, err := gitProvider.Head(ctx, repository.RepoID, "")
	if err != nil {
		return grpcerrors.Internal(err, "failed to resolve repository head")
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
		Message:         message,
		Operations:      operations,
		Author: gitprovider.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})
	if err != nil {
		return grpcerrors.Internal(err, "failed to commit repository files")
	}

	return nil
}

func snapshotGitFilesBeforeCommit(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	canvas *models.Canvas,
	gitOps []*pb.CanvasRepositoryFileOperation,
) ([]gitprovider.FileOperation, error) {
	if gitProvider == nil {
		return nil, grpcerrors.FailedPrecondition(nil, "git provider is not configured")
	}

	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "repository not found")
	}

	revertOps := make([]gitprovider.FileOperation, 0, len(gitOps))
	for _, operation := range gitOps {
		path := operation.GetPath()
		if operation.GetDelete() {
			reader, readErr := gitProvider.GetFile(ctx, repository.RepoID, path, "")
			if readErr != nil {
				return nil, grpcerrors.FailedPrecondition(nil, fmt.Sprintf("cannot snapshot %q before staged delete: %v", path, readErr))
			}

			content, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil {
				return nil, grpcerrors.Internal(readErr, "failed to read before commit")
			}

			revertOps = append(revertOps, gitprovider.FileOperation{
				Path:      path,
				Content:   bytes.NewReader(content),
				SizeBytes: int64(len(content)),
			})
			continue
		}

		reader, readErr := gitProvider.GetFile(ctx, repository.RepoID, path, "")
		if readErr != nil {
			revertOps = append(revertOps, gitprovider.FileOperation{
				Path:   path,
				Delete: true,
			})
			continue
		}

		content, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			return nil, grpcerrors.Internal(readErr, "failed to read before commit")
		}

		revertOps = append(revertOps, gitprovider.FileOperation{
			Path:      path,
			Content:   bytes.NewReader(content),
			SizeBytes: int64(len(content)),
		})
	}

	return revertOps, nil
}

func revertGitFileCommit(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	canvas *models.Canvas,
	organizationID string,
	userID string,
	revertOps []gitprovider.FileOperation,
) error {
	if len(revertOps) == 0 {
		return nil
	}

	pbOps := make([]*pb.CanvasRepositoryFileOperation, 0, len(revertOps))
	for _, operation := range revertOps {
		pbOp := &pb.CanvasRepositoryFileOperation{
			Path:   operation.Path,
			Delete: operation.Delete,
		}
		if operation.Content != nil && !operation.Delete {
			content, err := io.ReadAll(operation.Content)
			if err != nil {
				return grpcerrors.Internal(err, "failed to read revert content")
			}
			pbOp.Content = content
		}
		pbOps = append(pbOps, pbOp)
	}

	return commitStagedGitFiles(ctx, gitProvider, canvas, organizationID, userID, "Revert staged file commit", pbOps)
}

func resolvedStagingCommitMessage(messages ...string) string {
	for _, message := range messages {
		if trimmed := strings.TrimSpace(message); trimmed != "" {
			return trimmed
		}
	}
	return "Update files"
}
