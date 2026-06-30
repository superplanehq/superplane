package canvases

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"gorm.io/gorm"
)

func CommitCanvasStaging(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	branchName string,
	commitMessage string,
	newBranchName string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.CommitCanvasStagingResponse, error) {
	canvas, branch, headVersion, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, branchName, versionID)
	if err != nil {
		return nil, err
	}

	rows, err := models.ListWorkflowStaging(branch.ID, userUUID)
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

		message := resolvedStagingCommitMessage(commitMessage)
		if err := commitStagedGitFiles(ctx, gitProvider, canvas, organizationID, userUUID.String(), message, gitOps); err != nil {
			return nil, err
		}
	}

	targetBranchName := branch.Name
	if strings.TrimSpace(newBranchName) != "" {
		targetBranchName = strings.TrimSpace(newBranchName)
	}

	stagingBranchID := branch.ID

	var committed *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		message := resolvedStagingCommitMessage(commitMessage)
		if targetBranchName != branch.Name {
			createdBranch, createErr := models.CreateWorkflowBranch(tx, canvas.ID, targetBranchName, nil)
			if createErr != nil {
				return createErr
			}
			branch = createdBranch
		}

		var createErr error
		committed, createErr = models.CreateCommitOnBranch(tx, models.CreateCommitInput{
			WorkflowID:    canvas.ID,
			BranchName:    targetBranchName,
			OwnerID:       userUUID,
			CommitMessage: message,
			Nodes:         append([]models.Node(nil), headVersion.Nodes...),
			Edges:         append([]models.Edge(nil), headVersion.Edges...),
			ConsolePanels: headVersion.ConsolePanels.Data(),
			ConsoleLayout: headVersion.ConsoleLayout.Data(),
		})
		if createErr != nil {
			return createErr
		}

		return nil
	})
	if err != nil {
		if len(gitRevertOps) > 0 {
			if revertErr := revertGitFileCommit(ctx, gitProvider, canvas, organizationID, userUUID.String(), gitRevertOps); revertErr != nil {
				log.Errorf("failed to revert git commit after commit failure for canvas %s: %v", canvasID, revertErr)
			}
		}
		return nil, grpcerrors.Internal(err, "failed to create commit")
	}

	if len(specOps) > 0 {
		if err := ApplyRepositorySpecFileOperations(
			ctx,
			usageService,
			encryptor,
			registry,
			organizationID,
			canvasID,
			committed.ID.String(),
			webhookBaseURL,
			authService,
			nil,
			false,
			specOps,
		); err != nil {
			return nil, err
		}

		updated, reloadErr := models.FindCanvasVersion(canvas.ID, committed.ID)
		if reloadErr != nil {
			return nil, grpcerrors.Internal(reloadErr, "failed to load committed version")
		}
		committed = updated
	}

	if err := models.DiscardWorkflowStaging(stagingBranchID, userUUID, nil); err != nil {
		return nil, grpcerrors.Internal(err, "failed to clear staging")
	}

	state, _, err := stagingSummaryForBranch(stagingBranchID, userUUID)
	if err != nil {
		return nil, err
	}

	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersionMetadata(committed, organizationID, nil),
		StagingSummary: state,
	}, nil
}

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
	return "Update canvas"
}
