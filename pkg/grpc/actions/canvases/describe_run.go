package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func DescribeRun(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, runID string) (*pb.DescribeRunResponse, error) {
	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid run id")
	}

	db := database.DB(ctx)

	run, err := models.FindCanvasRunInTransaction(db, canvasID, runUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "run not found")
		}

		log.WithError(err).Error("failed to find run")
		return nil, grpcerrors.Internal(err, "failed to find run")
	}

	runDetails, err := loadRunDetailsForRuns(ctx, canvasID, []uuid.UUID{run.ID})
	if err != nil {
		return nil, err
	}

	serializedRun, err := SerializeCanvasRun(
		db,
		*run,
		runDetails.rootEventsByRunID[run.ID.String()],
		runDetails.executionsByRunID[run.ID.String()],
		runDetails.queueItemsByRunID[run.ID.String()],
	)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeRunResponse{
		Run: serializedRun,
	}, nil
}
