package canvases

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
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
	commitMessage string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.CommitCanvasStagingResponse, error) {
	staging, err := loadCanvasStagingContext(ctx, organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	if len(staging.rows) == 0 {
		return nil, grpcerrors.FailedPrecondition(nil, "no staged changes to commit")
	}

	if err := ensureStagingNotStale(staging); err != nil {
		return nil, err
	}

	specOps, gitOps := stagedCommitOperations(staging.rows)

	var gitRevertOps []gitprovider.FileOperation
	if len(gitOps) > 0 {
		var snapshotErr error
		gitRevertOps, snapshotErr = snapshotGitFilesBeforeCommit(ctx, gitProvider, staging.canvas, gitOps)
		if snapshotErr != nil {
			return nil, snapshotErr
		}

		if err := commitStagedGitFiles(
			ctx,
			gitProvider,
			staging.canvas,
			organizationID,
			staging.userID.String(),
			resolvedStagingCommitMessage(commitMessage),
			gitOps,
		); err != nil {
			return nil, err
		}
	}

	var nextVersionID string
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		liveVersion, liveErr := models.FindLiveCanvasVersionInTransaction(tx, staging.canvas.ID)
		if liveErr != nil {
			return liveErr
		}

		nextVersion, createErr := models.CreateCommitVersionFromLiveInTransaction(
			tx,
			liveVersion,
			staging.userID,
			commitMessage,
		)
		if createErr != nil {
			return createErr
		}

		nextVersionID = nextVersion.ID.String()
		return nil
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create commit version")
	}

	if len(specOps) > 0 {
		if err := ApplyRepositorySpecFileOperationsToCommitTarget(
			ctx,
			usageService,
			encryptor,
			registry,
			organizationID,
			canvasID,
			nextVersionID,
			webhookBaseURL,
			authService,
			nil,
			specOps,
		); err != nil {
			if len(gitRevertOps) > 0 {
				if revertErr := revertGitFileCommit(ctx, gitProvider, staging.canvas, organizationID, staging.userID.String(), gitRevertOps); revertErr != nil {
					log.Errorf("failed to revert git commit after spec apply failure for canvas %s: %v", canvasID, revertErr)
				}
			}
			return nil, err
		}
	}

	var committedVersion *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		liveVersion, liveErr := models.FindLiveCanvasVersionInTransaction(tx, staging.canvas.ID)
		if liveErr != nil {
			return liveErr
		}

		nextVersionUUID, parseErr := uuid.Parse(nextVersionID)
		if parseErr != nil {
			return parseErr
		}

		nextVersion, findErr := models.FindCanvasVersionInTransaction(tx, staging.canvas.ID, nextVersionUUID)
		if findErr != nil {
			return findErr
		}

		if publishErr := publishCanvasVersionInTransaction(
			ctx,
			tx,
			liveVersion,
			nextVersion,
			changesets.CanvasPublisherOptions{
				Registry:       registry,
				OrgID:          staging.canvas.OrganizationID,
				Encryptor:      encryptor,
				AuthService:    authService,
				WebhookBaseURL: webhookBaseURL,
				GitProvider:    gitProvider,
			},
		); publishErr != nil {
			if len(gitRevertOps) > 0 {
				if revertErr := revertGitFileCommit(ctx, gitProvider, staging.canvas, organizationID, staging.userID.String(), gitRevertOps); revertErr != nil {
					log.Errorf("failed to revert git commit after publish failure for canvas %s: %v", canvasID, revertErr)
				}
			}
			return publishErr
		}

		if discardErr := models.DiscardWorkflowStagingForUserInTransaction(tx, staging.canvas.ID, staging.userID, nil); discardErr != nil {
			return discardErr
		}

		committedVersion = nextVersion
		return nil
	})
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, grpcerrors.Internal(err, "failed to commit staging")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(staging.canvas.ID.String(), committedVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	publishStagingUpdated(staging.canvas.ID)

	state, _, err := stagingSummaryForCanvas(staging.canvas, staging.userID)
	if err != nil {
		return nil, err
	}

	ownersByID, _ := ownersByIDForCanvasVersions(ctx, organizationID, []models.CanvasVersion{*committedVersion})

	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersionMetadata(committedVersion, organizationID, ownersByID),
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
	return "Update files"
}
