package canvases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/services/files"
	"gorm.io/gorm"
)

func PutCanvasStaging(ctx context.Context, organizationID string, canvasID string, operations []*pb.CanvasRepositoryFileOperation) (*pb.StagingSummary, error) {
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

	//
	// Find the base version id for the staging update.
	//
	baseVersionID, err := findBaseVersionIDForStagingUpdate(db, canvas, userID)
	if err != nil {
		return nil, err
	}

	for _, operation := range operations {
		if operation == nil {
			continue
		}

		normalized := files.NormalizePath(operation.GetPath())
		if normalized == "" {
			return nil, grpcerrors.InvalidArgument(nil, "file path is required")
		}
		if normalized == gitprovider.ReservedSuperPlanePath ||
			strings.HasPrefix(normalized, gitprovider.ReservedSuperPlanePath+"/") {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("path %q is reserved for SuperPlane", operation.GetPath()))
		}

		if operation.GetDelete() {
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

	rows, err := models.ListStagedFilesForUser(db, canvas.ID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	if err := messages.NewCanvasStagingMessage(canvas.ID.String(), userID.String()).Publish(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}

	return buildStagingSummary(canvas, rows), nil
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

	//
	// If we already have staged files, use the base version id of the first staged file.
	//
	if len(stagedFiles) > 0 {
		baseVersionID := stagedFiles[0].BaseVersionID
		if baseVersionID != liveVersion.ID {
			return nil, grpcerrors.FailedPrecondition(nil, "stale staging cannot be updated")
		}

		return &baseVersionID, nil
	}

	//
	// Otherwise, use the live version id.
	//
	return &liveVersion.ID, nil
}
