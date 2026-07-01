package canvases

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

type commitWorkflowVersionInput struct {
	Canvas           *models.Canvas
	Version          *models.CanvasVersion
	OrganizationID   string
	UserID           uuid.UUID
	Message          string
	BranchName       string
	ParentBranchName string
	ExtraGitOps      []*pb.CanvasRepositoryFileOperation
}

func commitWorkflowVersionToGit(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	input commitWorkflowVersionInput,
) (string, error) {
	if gitProvider == nil {
		return "", grpcerrors.FailedPrecondition(nil, "git provider is not configured")
	}

	repository, err := models.FindRepository(input.Canvas.OrganizationID, input.Canvas.ID)
	if err != nil {
		return "", grpcerrors.NotFound(err, "repository not found")
	}

	author, err := resolveGitCommitAuthor(input.OrganizationID, input.UserID)
	if err != nil {
		return "", err
	}

	branchName := gitprovider.DefaultBranch(strings.TrimSpace(input.BranchName))
	parentBranchName := gitprovider.DefaultBranch(strings.TrimSpace(input.ParentBranchName))
	if parentBranchName == "" {
		parentBranchName = models.CanvasGitBranchMain
	}

	if err := ensureGitBranchExists(ctx, gitProvider, repository.RepoID, branchName, parentBranchName); err != nil {
		log.WithError(err).WithFields(workflowGitSyncLogFields(input, repository.RepoID, branchName, parentBranchName)).Error("commitWorkflowVersionToGit: failed to prepare git branch")
		return "", grpcerrors.Internal(err, "failed to prepare git branch")
	}

	operations, err := buildWorkflowVersionGitOperations(input.Canvas, input.Version, input.OrganizationID, input.ExtraGitOps)
	if err != nil {
		return "", err
	}

	headSHA, err := gitProvider.Head(ctx, repository.RepoID, branchName)
	if err != nil {
		log.WithError(err).WithFields(workflowGitSyncLogFields(input, repository.RepoID, branchName, parentBranchName)).Error("commitWorkflowVersionToGit: failed to resolve repository head")
		return "", grpcerrors.Internal(err, "failed to resolve repository head")
	}

	commitSHA, err := gitProvider.Commit(ctx, repository.RepoID, gitprovider.CommitOptions{
		Branch:          branchName,
		BaseBranch:      branchName,
		ExpectedHeadSHA: headSHA,
		Message:         strings.TrimSpace(input.Message),
		Operations:      operations,
		Author:          author,
	})
	if err != nil {
		log.WithError(err).WithFields(workflowGitSyncLogFields(input, repository.RepoID, branchName, parentBranchName)).WithField("expected_head_sha", headSHA).Error("commitWorkflowVersionToGit: failed to commit workflow version to git")
		return "", grpcerrors.Internal(err, "failed to commit workflow version to git")
	}

	return commitSHA, nil
}

func mergeWorkflowBranchInGit(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	canvas *models.Canvas,
	organizationID string,
	userID uuid.UUID,
	sourceBranch string,
	targetBranch string,
	message string,
) (string, error) {
	if gitProvider == nil {
		return "", grpcerrors.FailedPrecondition(nil, "git provider is not configured")
	}

	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err != nil {
		return "", grpcerrors.NotFound(err, "repository not found")
	}

	author, err := resolveGitCommitAuthor(organizationID, userID)
	if err != nil {
		return "", err
	}

	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = gitprovider.DefaultBranch(strings.TrimSpace(targetBranch))
	if sourceBranch == "" || sourceBranch == targetBranch {
		headSHA, headErr := gitProvider.Head(ctx, repository.RepoID, targetBranch)
		if headErr != nil {
			log.WithError(headErr).WithFields(workflowGitMergeLogFields(canvas, organizationID, repository.RepoID, sourceBranch, targetBranch)).Error("mergeWorkflowBranchInGit: failed to resolve repository head")
			return "", grpcerrors.Internal(headErr, "failed to resolve repository head")
		}
		return headSHA, nil
	}

	if err := ensureGitBranchExists(ctx, gitProvider, repository.RepoID, sourceBranch, targetBranch); err != nil {
		log.WithError(err).WithFields(workflowGitMergeLogFields(canvas, organizationID, repository.RepoID, sourceBranch, targetBranch)).Error("mergeWorkflowBranchInGit: failed to prepare source git branch")
		return "", grpcerrors.Internal(err, "failed to prepare source git branch")
	}

	mergeSHA, err := gitProvider.MergeBranch(ctx, repository.RepoID, sourceBranch, targetBranch, message, author)
	if err != nil {
		log.WithError(err).WithFields(workflowGitMergeLogFields(canvas, organizationID, repository.RepoID, sourceBranch, targetBranch)).Error("mergeWorkflowBranchInGit: failed to merge git branch")
		return "", grpcerrors.Internal(err, "failed to merge git branch")
	}

	return mergeSHA, nil
}

func deleteGitBranchBestEffort(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	canvas *models.Canvas,
	branchName string,
) {
	if gitProvider == nil {
		return
	}

	branchName = strings.TrimSpace(branchName)
	if branchName == "" || branchName == models.CanvasGitBranchMain {
		return
	}

	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err != nil {
		return
	}

	_ = gitProvider.DeleteBranch(ctx, repository.RepoID, branchName)
}

func ensureGitBranchExists(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	repoID string,
	branchName string,
	fromBranch string,
) error {
	branchName = gitprovider.DefaultBranch(branchName)
	fromBranch = gitprovider.DefaultBranch(fromBranch)

	if gitBranchExists(ctx, gitProvider, repoID, branchName) {
		return nil
	}

	return gitProvider.CreateBranch(ctx, repoID, branchName, fromBranch)
}

func gitBranchExists(ctx context.Context, gitProvider gitprovider.Provider, repoID, branchName string) bool {
	branches, err := gitProvider.ListBranches(ctx, repoID, branchName)
	if err == nil && slices.Contains(branches, branchName) {
		return true
	}

	_, err = gitProvider.Head(ctx, repoID, branchName)
	return err == nil
}

func buildWorkflowVersionGitOperations(
	canvas *models.Canvas,
	version *models.CanvasVersion,
	organizationID string,
	extraOps []*pb.CanvasRepositoryFileOperation,
) ([]gitprovider.FileOperation, error) {
	canvasYAML, err := canvasYAMLFromVersion(canvas, version, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to serialize canvas.yaml")
	}

	consoleYAML, err := consoleYAMLFromVersion(version)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to serialize console.yaml")
	}

	operations := []gitprovider.FileOperation{
		{
			Path:      CanvasYAMLRepositoryPath,
			Content:   strings.NewReader(canvasYAML),
			SizeBytes: int64(len(canvasYAML)),
		},
		{
			Path:      ConsoleYAMLRepositoryPath,
			Content:   strings.NewReader(consoleYAML),
			SizeBytes: int64(len(consoleYAML)),
		},
	}

	for _, operation := range extraOps {
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

	return operations, nil
}

// SyncCanvasVersionsMissingCommitSHA commits workflow versions on main that were
// persisted before the git repository existed (for example the canvas initial commit).
func SyncCanvasVersionsMissingCommitSHA(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	tx *gorm.DB,
	organizationID string,
	canvas *models.Canvas,
) error {
	if gitProvider == nil {
		return nil
	}

	mainBranch, err := models.FindMainWorkflowBranch(tx, canvas.ID)
	if err != nil {
		return err
	}

	var versions []models.CanvasVersion
	if err := tx.
		Where("branch_id = ?", mainBranch.ID).
		Order("created_at ASC, id ASC").
		Find(&versions).Error; err != nil {
		return err
	}

	for i := range versions {
		version := versions[i]
		if strings.TrimSpace(version.CommitSHA) != "" {
			continue
		}

		userID := uuid.Nil
		if version.OwnerID != nil {
			userID = *version.OwnerID
		} else if canvas.CreatedBy != nil {
			userID = *canvas.CreatedBy
		}
		if userID == uuid.Nil {
			continue
		}

		message := strings.TrimSpace(version.CommitMessage)
		if message == "" {
			message = "Initial commit"
		}

		commitSHA, err := commitWorkflowVersionToGit(ctx, gitProvider, commitWorkflowVersionInput{
			Canvas:           canvas,
			Version:          &version,
			OrganizationID:   organizationID,
			UserID:           userID,
			Message:          message,
			BranchName:       version.GitBranch,
			ParentBranchName: models.CanvasGitBranchMain,
		})
		if err != nil {
			return err
		}

		if err := models.UpdateCanvasVersionCommitSHAInTransaction(tx, canvas.ID, version.ID, commitSHA); err != nil {
			return err
		}
	}

	return nil
}

func workflowGitSyncLogFields(input commitWorkflowVersionInput, repoID, branchName, parentBranchName string) log.Fields {
	fields := log.Fields{
		"organization_id": input.OrganizationID,
		"canvas_id":       input.Canvas.ID.String(),
		"version_id":      input.Version.ID.String(),
		"repo_id":         repoID,
		"branch_name":     branchName,
		"parent_branch":   parentBranchName,
	}
	if input.UserID != uuid.Nil {
		fields["user_id"] = input.UserID.String()
	}
	return fields
}

func workflowGitMergeLogFields(canvas *models.Canvas, organizationID, repoID, sourceBranch, targetBranch string) log.Fields {
	return log.Fields{
		"organization_id": organizationID,
		"canvas_id":       canvas.ID.String(),
		"repo_id":         repoID,
		"source_branch":   sourceBranch,
		"target_branch":   targetBranch,
	}
}

func resolveGitCommitAuthor(organizationID string, userID uuid.UUID) (gitprovider.CommitAuthor, error) {
	user, err := models.FindActiveUserByID(organizationID, userID.String())
	if err != nil {
		return gitprovider.CommitAuthor{}, grpcerrors.Internal(err, "failed to find user")
	}

	return gitprovider.CommitAuthor{
		Name:  user.Name,
		Email: user.GetEmail(),
	}, nil
}
