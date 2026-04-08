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

func DescribeRun(ctx context.Context, reg *registry.Registry, canvasID, eventID string) (*pb.DescribeRunResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id: %v", err)
	}

	rootEvent, err := models.FindCanvasEvent(eventUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "run not found")
		}
		return nil, err
	}

	if rootEvent.WorkflowID != canvasUUID {
		return nil, status.Errorf(codes.NotFound, "run not found")
	}

	allExecutions, err := models.ListAllExecutionsForRootEvent(eventUUID)
	if err != nil {
		return nil, err
	}

	parentExecutions := filterParentExecutions(allExecutions)
	childExecutions := filterChildExecutions(allExecutions)

	executionsByEventID := map[string][]models.CanvasNodeExecution{
		rootEvent.ID.String(): parentExecutions,
	}

	runEvent, err := SerializeCanvasEventWithExecutions(*rootEvent, executionsByEventID[rootEvent.ID.String()])
	if err != nil {
		return nil, err
	}

	serializedExecutions, err := SerializeNodeExecutions(parentExecutions, childExecutions)
	if err != nil {
		return nil, err
	}

	snapshotVersion, err := resolveSnapshotVersion(canvasUUID, allExecutions)
	if err != nil {
		return nil, err
	}

	var protoVersion *pb.CanvasVersion
	if snapshotVersion != nil {
		canvas, err := models.FindCanvasWithoutOrgScope(canvasUUID)
		if err != nil {
			return nil, err
		}
		protoVersion = SerializeCanvasVersion(snapshotVersion, canvas.OrganizationID.String())
	}

	return &pb.DescribeRunResponse{
		Run:             runEvent,
		SnapshotVersion: protoVersion,
		Executions:      serializedExecutions,
	}, nil
}

func resolveSnapshotVersion(canvasID uuid.UUID, executions []models.CanvasNodeExecution) (*models.CanvasVersion, error) {
	for _, exec := range executions {
		if exec.CanvasVersionID != nil {
			version, err := models.FindCanvasVersion(canvasID, *exec.CanvasVersionID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return nil, err
			}
			return version, nil
		}
	}

	// Fall back to the current live version for runs that predate the migration.
	version, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return version, nil
}

func filterParentExecutions(executions []models.CanvasNodeExecution) []models.CanvasNodeExecution {
	result := make([]models.CanvasNodeExecution, 0, len(executions))
	for _, exec := range executions {
		if exec.ParentExecutionID == nil {
			result = append(result, exec)
		}
	}
	return result
}

func filterChildExecutions(executions []models.CanvasNodeExecution) []models.CanvasNodeExecution {
	result := make([]models.CanvasNodeExecution, 0)
	for _, exec := range executions {
		if exec.ParentExecutionID != nil {
			result = append(result, exec)
		}
	}
	return result
}
