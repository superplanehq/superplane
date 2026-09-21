package canvases

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
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
	db *gorm.DB,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	canvas *models.Canvas,
	commitMessage string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.CommitCanvasStagingResponse, error) {
	user, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	userID := uuid.MustParse(user)
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

	specOps := stagedSpecOperations(stagedFiles)
	if len(specOps) == 0 {
		return nil, grpcerrors.FailedPrecondition(nil, "no staged changes to commit")
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
			canvas.OrganizationID.String(),
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

	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}

		log.Errorf("failed to commit staging: %v", err)
		return nil, grpcerrors.Internal(err, "failed to commit staging")
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}

	if err := messages.NewCanvasStagingMessage(canvas.ID.String(), userID.String()).Publish(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}

	publishDeletedNodeCleanupMessages(canvas.ID, publishResult)

	ownersByID, _ := ownersByIDForCanvasVersions(ctx, canvas.OrganizationID.String(), []models.CanvasVersion{*newLiveVersion})

	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersion(newLiveVersion, canvas.OrganizationID.String(), ownersByID),
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

func stagedSpecOperations(rows []models.WorkflowStagedFile) []*pb.CanvasRepositoryFileOperation {
	specContentByPath := map[string]string{}
	for _, row := range rows {
		if !IsRepositorySpecFilePath(row.Path) || row.Deleted {
			continue
		}
		specContentByPath[row.Path] = row.Content
	}

	var specOps []*pb.CanvasRepositoryFileOperation
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

	return specOps
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
