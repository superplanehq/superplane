package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

	run, err := models.FindCanvasRunInTransaction(db, canvasID, runUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "run not found")
		}

		log.WithError(err).Error("failed to find run")
		return nil, status.Errorf(codes.Internal, "failed to find run")
	}

	rootEventsByRunID, err := listRootEventsForRuns(ctx, canvasID, []uuid.UUID{run.ID})
	if err != nil {
		return nil, err
	}

	executions, err := models.ListExecutionsForRunsInTransaction(db, canvasID, []uuid.UUID{run.ID})
	if err != nil {
		return nil, err
	}

	serializedRun, err := SerializeCanvasRun(*run, rootEventsByRunID[run.ID.String()], executions)
	if err != nil {
		return nil, err
	}

	childRuns, err := models.ListChildCanvasRunsInTransaction(db, canvasID, run.ID)
	if err != nil {
		return nil, err
	}

	childRunIDs := make([]uuid.UUID, 0, len(childRuns))
	for _, childRun := range childRuns {
		childRunIDs = append(childRunIDs, childRun.ID)
	}

	childRootEventsByRunID, err := listRootEventsForRuns(ctx, canvasID, childRunIDs)
	if err != nil {
		return nil, err
	}

	childExecutions, err := models.ListExecutionsForRunsInTransaction(db, canvasID, childRunIDs)
	if err != nil {
		return nil, err
	}

	childExecutionsByRunID := make(map[string][]models.CanvasNodeExecution, len(childRunIDs))
	for _, execution := range childExecutions {
		childExecutionsByRunID[execution.RunID.String()] = append(childExecutionsByRunID[execution.RunID.String()], execution)
	}

	serializedChildRuns, err := SerializeCanvasRuns(childRuns, childRootEventsByRunID, childExecutionsByRunID)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeRunResponse{
		Run:       serializedRun,
		ChildRuns: serializedChildRuns,
	}, nil
}
