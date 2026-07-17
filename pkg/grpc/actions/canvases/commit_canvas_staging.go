package canvases

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
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
	db := database.DB(ctx)

	user, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	userID := uuid.MustParse(user)
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvasInTransaction(db, uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	stagedFiles, err := models.ListStagedFilesForUser(db, canvas.ID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	if len(stagedFiles) == 0 {
		return nil, grpcerrors.FailedPrecondition(nil, "no staged changes to commit")
	}

	//
	// Verify if staged files are for the live version.
	// Staged files for stale versions cannot be committed.
	//
	if err := ensureNotStaleStaging(db, canvas, stagedFiles); err != nil {
		return nil, err
	}

	//
	// Commit to git first.
	// If something goes wrong, we will revert the git commit.
	//
	specOps, gitOps := stagedCommitOperations(stagedFiles)
	var gitRevertOps []gitprovider.FileOperation
	if len(gitOps) > 0 {
		var snapshotErr error
		gitRevertOps, snapshotErr = snapshotGitFilesBeforeCommit(ctx, gitProvider, canvas, gitOps)
		if snapshotErr != nil {
			return nil, snapshotErr
		}

		if err := commitStagedGitFiles(
			ctx,
			gitProvider,
			canvas,
			organizationID,
			userID.String(),
			resolvedStagingCommitMessage(commitMessage),
			gitOps,
		); err != nil {
			return nil, err
		}
	}

	var newLiveVersion *models.CanvasVersion
	var publishResult changesets.CanvasPublishResult
	err = db.Transaction(func(tx *gorm.DB) error {
		liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
		if err != nil {
			return grpcerrors.Internal(err, "failed to load live version")
		}

		if err := ensureNotStaleStaging(tx, canvas, stagedFiles); err != nil {
			return err
		}

		//
		// Create the new version, starting from the specs from the live version,
		// and applying the spec operations on it.
		//
		nextVersion, err := createNewCanvasVersionFromLive(
			ctx,
			tx,
			usageService,
			registry,
			organizationID,
			canvas,
			liveVersion,
			specOps,
			userID,
			commitMessage,
		)

		if err != nil {
			log.Errorf("failed to create new canvas version from live: %v", err)
			return err
		}

		//
		// Make the new version live.
		//
		publishResult, err = publishCanvasVersionInTransaction(
			ctx,
			tx,
			canvas,
			liveVersion,
			nextVersion,
			changesets.CanvasPublisherOptions{
				Registry:       registry,
				OrgID:          canvas.OrganizationID,
				Encryptor:      encryptor,
				AuthService:    authService,
				WebhookBaseURL: webhookBaseURL,
				GitProvider:    gitProvider,
			},
		)

		if err != nil {
			return err
		}

		//
		// Remove staged files for user.
		//
		err = models.DiscardStagedFilesForUser(tx, canvas.ID, userID, nil)
		if err != nil {
			return err
		}

		newLiveVersion = nextVersion
		return nil
	})

	//
	// If anything goes wrong here, we might need to revert the git commit.
	//
	if err != nil {
		if len(gitRevertOps) > 0 {
			if revertErr := revertGitFileCommit(ctx, gitProvider, canvas, organizationID, userID.String(), gitRevertOps); revertErr != nil {
				log.Errorf("failed to revert git commit after spec apply failure for canvas %s: %v", canvasID, revertErr)
			}
		}

		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}

		log.Errorf("failed to commit staging: %v", err)
		return nil, grpcerrors.Internal(err, "failed to commit staging")
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), organizationID).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}

	if err := messages.NewCanvasStagingMessage(canvas.ID.String(), userID.String()).Publish(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}

	publishDeletedNodeCleanupMessages(canvas.ID, publishResult)

	ownersByID, _ := ownersByIDForCanvasVersions(ctx, organizationID, []models.CanvasVersion{*newLiveVersion})

	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersionMetadata(newLiveVersion, organizationID, ownersByID),
		StagingSummary: buildStagingSummary(canvas, []models.WorkflowStagedFile{}),
	}, nil
}

func publishDeletedNodeCleanupMessages(canvasID uuid.UUID, result changesets.CanvasPublishResult) {
	for _, executionID := range result.CancelledExecutionIDs {
		if err := messages.PublishCanvasExecutionByID(canvasID, executionID); err != nil {
			log.Errorf("failed to publish cancelled execution RabbitMQ message: %v", err)
		}
	}

	for _, queueItem := range result.DeletedQueueItems {
		if queueItem.RunID == uuid.Nil {
			continue
		}

		if err := messages.NewCanvasQueueItemMessage(queueItem).PublishDeleted(); err != nil {
			log.Errorf("failed to publish deleted queue item RabbitMQ message: %v", err)
		}
	}
}

func stagedCommitOperations(rows []models.WorkflowStagedFile) (specOps, gitOps []*pb.CanvasRepositoryFileOperation) {
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

func createNewCanvasVersionFromLive(
	ctx context.Context,
	tx *gorm.DB,
	usageService usage.Service,
	registry *registry.Registry,
	organizationID string,
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	operations []*pb.CanvasRepositoryFileOperation,
	userID uuid.UUID,
	commitMessage string,
) (*models.CanvasVersion, error) {

	//
	// Start new version with the live version's nodes, edges, console panels, and console layout.
	//
	now := time.Now()
	newVersion := models.CanvasVersion{
		ID:            uuid.New(),
		WorkflowID:    canvas.ID,
		OwnerID:       &userID,
		CommitMessage: strings.TrimSpace(commitMessage),
		Nodes:         datatypes.NewJSONSlice(slices.Clone(liveVersion.Nodes)),
		Edges:         datatypes.NewJSONSlice(slices.Clone(liveVersion.Edges)),
		ConsolePanels: datatypes.NewJSONType(slices.Clone(liveVersion.ConsolePanels.Data())),
		ConsoleLayout: datatypes.NewJSONType(slices.Clone(liveVersion.ConsoleLayout.Data())),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	//
	// Update it with the operations.
	//
	for _, operation := range operations {
		if operation == nil {
			continue
		}

		if operation.GetDelete() {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("%q cannot be deleted", operation.GetPath()))
		}

		normalized := normalizeRepositoryFilePath(operation.GetPath())
		content := string(operation.GetContent())

		switch normalized {
		case CanvasYAMLRepositoryPath:
			canvas, err := yaml.CanvasFromYAML([]byte(content))
			if err != nil {
				return nil, grpcerrors.InvalidArgument(err, "invalid canvas yaml")
			}

			nodes, edges, err := canvas.Parse(registry, organizationID)
			if err != nil {
				return nil, grpcerrors.InvalidArgument(err, "invalid canvas yaml")
			}

			err = usage.EnsureOrganizationWithinLimits(
				ctx,
				usageService,
				organizationID,
				&usagepb.OrganizationState{},
				&usagepb.CanvasState{
					Nodes: int32(len(nodes)),
				},
			)

			if err != nil {
				return nil, err
			}

			newNodes := injectMetadataIntoNodes(liveVersion.Nodes, nodes)
			newVersion.Nodes = datatypes.NewJSONSlice(slices.Clone(newNodes))
			newVersion.Edges = datatypes.NewJSONSlice(slices.Clone(edges))
		case ConsoleYAMLRepositoryPath:
			console, err := yaml.ConsoleFromYML([]byte(content))
			if err != nil {
				return nil, grpcerrors.InvalidArgument(err, "invalid console yaml")
			}

			newVersion.ConsolePanels = datatypes.NewJSONType(slices.Clone(console.Panels()))
			newVersion.ConsoleLayout = datatypes.NewJSONType(slices.Clone(console.Layout()))
		default:
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("unsupported repository spec file %q", operation.GetPath()))
		}
	}

	err := tx.Create(&newVersion).Error
	if err != nil {
		return nil, err
	}

	return &newVersion, nil
}

func injectMetadataIntoNodes(versionNodes []models.Node, proposedNodes []models.Node) []models.Node {
	result := make([]models.Node, len(proposedNodes))
	copy(result, proposedNodes)

	for i, proposedNode := range result {
		for _, versionNode := range versionNodes {
			if proposedNode.ID == versionNode.ID {
				result[i].Metadata = versionNode.Metadata
			}
		}
	}

	return result
}

func ensureNotStaleStaging(db *gorm.DB, canvas *models.Canvas, stagedFiles []models.WorkflowStagedFile) error {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	if err != nil {
		return grpcerrors.Internal(err, "failed to load live version")
	}

	baseVersionID := findStagingBaseVersionID(stagedFiles)
	if baseVersionID != liveVersion.ID {
		return grpcerrors.FailedPrecondition(nil, "stale staging cannot be committed")
	}

	return nil
}
