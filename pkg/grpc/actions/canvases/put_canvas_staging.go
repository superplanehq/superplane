package canvases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func PutCanvasStaging(ctx context.Context, db *gorm.DB, canvas *models.Canvas, operations []*pb.CanvasRepositoryFileOperation) (*pb.StagingSummary, error) {
	user, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	userID := uuid.MustParse(user)

	validated, err := validatedStagedSpecOperations(operations)
	if err != nil {
		return nil, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		baseVersionID, err := findBaseVersionIDForStagingUpdate(tx, canvas, userID)
		if err != nil {
			return err
		}

		for _, operation := range validated {
			if _, err := models.UpsertStagedFile(
				tx,
				canvas.ID,
				userID,
				*baseVersionID,
				canvas.OrganizationID,
				operation.path,
				operation.content,
			); err != nil {
				return grpcerrors.Internal(err, "failed to stage")
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
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

type stagedSpecOperation struct {
	path    string
	content string
}

func validatedStagedSpecOperations(operations []*pb.CanvasRepositoryFileOperation) ([]stagedSpecOperation, error) {
	validated := make([]stagedSpecOperation, 0, len(operations))
	for _, operation := range operations {
		if operation == nil {
			continue
		}

		normalized, err := requireStagedSpecFilePath(operation.GetPath())
		if err != nil {
			return nil, err
		}

		if operation.GetDelete() {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("%q cannot be deleted", operation.GetPath()))
		}

		validated = append(validated, stagedSpecOperation{
			path:    normalized,
			content: string(operation.GetContent()),
		})
	}

	return validated, nil
}
