package canvases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func PutCanvasStaging(
	ctx context.Context,
	db *gorm.DB,
	registry *registry.Registry,
	canvas *models.Canvas,
	operations []*pb.CanvasRepositoryFileOperation,
) (*CanvasStagingState, error) {
	user, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	userID := uuid.MustParse(user)

	baseVersionID, err := findBaseVersionIDForStagingUpdate(db, canvas, userID)
	if err != nil {
		return nil, err
	}

	for _, operation := range operations {
		if operation == nil {
			continue
		}

		normalized := normalizeRepositoryFilePath(operation.GetPath())
		if normalized == "" {
			return nil, grpcerrors.InvalidArgument(nil, "file path is required")
		}
		if normalized == gitprovider.ReservedSuperPlanePath ||
			strings.HasPrefix(normalized, gitprovider.ReservedSuperPlanePath+"/") {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("path %q is reserved for SuperPlane", operation.GetPath()))
		}

		if operation.GetDelete() {
			if IsRepositorySpecFilePath(normalized) {
				return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("%q cannot be deleted", operation.GetPath()))
			}

			if err := models.MarkStagedFilePathDeleted(
				db,
				canvas.ID,
				userID,
				*baseVersionID,
				canvas.OrganizationID,
				normalized,
			); err != nil {
				return nil, grpcerrors.Internal(err, "failed to stage deletion")
			}
			continue
		}

		if IsRepositorySpecFilePath(normalized) {
			if err := validateStagedSpecFileContent(registry, canvas.OrganizationID.String(), normalized, operation.GetContent()); err != nil {
				return nil, err
			}
		}

		if _, err := models.UpsertStagedFile(
			db,
			canvas.ID,
			userID,
			*baseVersionID,
			canvas.OrganizationID,
			normalized,
			string(operation.GetContent()),
		); err != nil {
			return nil, grpcerrors.Internal(err, "failed to stage")
		}
	}

	if err := messages.NewCanvasStagingMessage(canvas.ID.String(), userID.String()).Publish(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}

	return BuildCanvasStagingState(ctx, db, registry, canvas, userID)
}

func findBaseVersionIDForStagingUpdate(db *gorm.DB, canvas *models.Canvas, userID uuid.UUID) (*uuid.UUID, error) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load live version")
	}

	stagedFiles, err := models.ListStagedFilesForUser(db, canvas.ID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	if len(stagedFiles) > 0 {
		baseVersionID := stagedFiles[0].BaseVersionID
		if baseVersionID != liveVersion.ID {
			return nil, grpcerrors.FailedPrecondition(nil, "stale staging cannot be updated")
		}

		return &baseVersionID, nil
	}

	return &liveVersion.ID, nil
}
