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

const CurrentStagingCannotBeDiscardedMessage = "current staging cannot be discarded"

func PutCanvasStaging(ctx context.Context, db *gorm.DB, canvas *models.Canvas, operations []*pb.CanvasRepositoryFileOperation) (*pb.StagingSummary, error) {
	return putCanvasStaging(ctx, db, canvas, operations, false)
}

func PutCanvasStagingReplacingStale(ctx context.Context, db *gorm.DB, canvas *models.Canvas, operations []*pb.CanvasRepositoryFileOperation) (*pb.StagingSummary, error) {
	return putCanvasStaging(ctx, db, canvas, operations, true)
}

func putCanvasStaging(ctx context.Context, db *gorm.DB, canvas *models.Canvas, operations []*pb.CanvasRepositoryFileOperation, replaceIfStale bool) (*pb.StagingSummary, error) {
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
		if err := models.LockStagedFilesForUser(tx, canvas.ID, userID); err != nil {
			return grpcerrors.Internal(err, "failed to stage")
		}

		baseVersionID, err := stagingBaseVersionID(tx, canvas, userID, replaceIfStale)
		if err != nil {
			return err
		}

		for _, operation := range validated {
			if _, err := models.UpsertStagedFile(
				tx,
				canvas.ID,
				userID,
				baseVersionID,
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

func stagingBaseVersionID(db *gorm.DB, canvas *models.Canvas, userID uuid.UUID, replaceIfStale bool) (uuid.UUID, error) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	if err != nil {
		return uuid.Nil, grpcerrors.Internal(err, "failed to load live version")
	}

	stagedFiles, err := models.ListStagedFilesForUser(db, canvas.ID, userID)
	if err != nil {
		return uuid.Nil, grpcerrors.Internal(err, "failed to load staging")
	}

	if len(stagedFiles) == 0 {
		return liveVersion.ID, nil
	}

	baseVersionID := stagedFiles[0].BaseVersionID
	if baseVersionID == liveVersion.ID {
		if replaceIfStale {
			return uuid.Nil, grpcerrors.FailedPrecondition(nil, CurrentStagingCannotBeDiscardedMessage)
		}
		return baseVersionID, nil
	}

	if !replaceIfStale {
		return uuid.Nil, grpcerrors.FailedPrecondition(nil, "stale staging cannot be updated")
	}

	if err := models.DiscardStagedFilesForUser(db, canvas.ID, userID, nil); err != nil {
		return uuid.Nil, grpcerrors.Internal(err, "failed to discard staging")
	}

	return liveVersion.ID, nil
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
