package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DescribeRun(ctx context.Context, db *gorm.DB, canvas *models.Canvas, runID string) (*pb.DescribeRunResponse, error) {
	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid run id")
	}

	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, runUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "run not found")
		}

		log.WithError(err).Error("failed to find run")
		return nil, grpcerrors.Internal(err, "failed to find run")
	}

	runDetails, err := loadRunDetailsForRuns(ctx, db, canvas.ID, []models.CanvasRun{*run})
	if err != nil {
		return nil, err
	}

	serializedRun, err := SerializeCanvasRun(
		db,
		*run,
		runDetails.rootEventsByRunID[run.ID.String()],
		runDetails.executionsByRunID[run.ID.String()],
		runDetails.queueItemsByRunID[run.ID.String()],
		parentRunForDescribe(runDetails.parentRunsByRunID, run.ID.String()),
		runDetails.childRunsByExecutionID,
	)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeRunResponse{
		Run: serializedRun,
	}, nil
}

func parentRunForDescribe(parentRunsByRunID map[string]models.CanvasRun, runID string) *models.CanvasRun {
	parent, ok := parentRunsByRunID[runID]
	if !ok {
		return nil
	}

	return &parent
}
