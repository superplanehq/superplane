package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeRun(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, runID string) (*pb.DescribeRunResponse, error) {
	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid run id: %v", err)
	}

	db := database.DB(ctx)

	run, err := findCanvasRunForDescribe(db, canvasID, runUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "run not found")
		}

		return nil, err
	}

	rootEventsByRunID, err := listRootEventsForRuns(ctx, canvasID, []uuid.UUID{run.ID})
	if err != nil {
		return nil, err
	}

	executions, err := models.ListParentExecutionsForRunsInTransaction(db, canvasID, []uuid.UUID{run.ID})
	if err != nil {
		return nil, err
	}

	serializedRun, err := SerializeCanvasRun(*run, rootEventsByRunID[run.ID.String()], executions)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeRunResponse{
		Run: serializedRun,
	}, nil
}

func findCanvasRunForDescribe(db *gorm.DB, canvasID, runUUID uuid.UUID) (*models.CanvasRun, error) {
	run, err := models.FindCanvasRunInTransaction(db, canvasID, runUUID)
	if err == nil {
		return run, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	run, err = models.FindCanvasRunByRootEventInTransaction(db, runUUID)
	if err != nil {
		return nil, err
	}

	if run.WorkflowID != canvasID {
		return nil, gorm.ErrRecordNotFound
	}

	return run, nil
}
