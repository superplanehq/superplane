package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func CancelRun(ctx context.Context, organizationID string, workflowID, runID uuid.UUID) (*pb.CancelRunResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.PermissionDenied(nil, "user not authenticated")
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "user not found")
	}

	var run *models.CanvasRun
	var drainResult *models.RunCancellationDrainResult
	var publishCancelled bool

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		run, err = models.LockCanvasRunInTransaction(tx, runID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(err, "run not found")
			}

			return err
		}

		if run.WorkflowID != workflowID {
			return grpcerrors.NotFound(nil, "run not found")
		}

		if run.State == models.CanvasRunStateFinished {
			return nil
		}

		drainResult, err = run.DrainForCancellation(tx, &user.ID)
		if err != nil {
			return err
		}

		if run.State == models.CanvasRunStateCancelling {
			return nil
		}

		publishCancelled = true
		return run.MarkAsCancelling(tx, &user.ID)
	})

	if err != nil {
		return nil, err
	}

	if publishCancelled {
		if err := messages.NewCanvasRunMessage(workflowID.String(), runID.String()).Publish(); err != nil {
			log.Errorf("failed to publish run state RabbitMQ message: %v", err)
		}
	}

	messages.PublishRunCancellationDrain(workflowID, drainResult)

	run, err = models.FindCanvasRunInTransaction(database.DB(ctx), workflowID, runID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load run")
	}

	runDetails, err := loadRunDetailsForRuns(ctx, workflowID, []models.CanvasRun{*run})
	if err != nil {
		return nil, err
	}

	serializedRun, err := SerializeCanvasRun(
		database.DB(ctx),
		*run,
		runDetails.rootEventsByRunID[run.ID.String()],
		runDetails.executionsByRunID[run.ID.String()],
		runDetails.queueItemsByRunID[run.ID.String()],
		parentRunForDescribe(runDetails.parentRunsByRunID, run.ID.String()),
		map[string][]models.CanvasRun{},
	)
	if err != nil {
		return nil, err
	}

	return &pb.CancelRunResponse{
		Run: serializedRun,
	}, nil
}
