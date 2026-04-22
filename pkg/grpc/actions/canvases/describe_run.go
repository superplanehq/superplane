package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// DescribeRun returns the full picture of a single canvas run:
//   - the root event (with execution refs),
//   - the parent executions,
//   - the child executions (for blueprint nodes),
//   - the snapshot canvas version active when the run started.
//
// It is the backend for the Run View in the UI and intentionally includes
// everything needed to render the run without additional round trips.
func DescribeRun(ctx context.Context, organizationID, canvasID, eventID string) (*pb.DescribeRunResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id: %v", err)
	}

	rootEvent, err := models.FindCanvasEventForCanvas(canvasUUID, eventUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "run not found")
		}
		return nil, err
	}

	allExecutions, err := models.ListAllExecutionsForRootEvent(rootEvent.ID)
	if err != nil {
		return nil, err
	}

	parents, children := splitParentAndChildExecutions(allExecutions)

	executionsByEventID := map[string][]models.CanvasNodeExecution{
		rootEvent.ID.String(): parents,
	}

	serializedRun, err := SerializeCanvasEventWithExecutions(*rootEvent, executionsByEventID[rootEvent.ID.String()])
	if err != nil {
		return nil, err
	}

	serializedExecutions, err := SerializeNodeExecutions(parents, nil)
	if err != nil {
		return nil, err
	}

	serializedChildren, err := SerializeNodeExecutions(children, nil)
	if err != nil {
		return nil, err
	}

	snapshotVersion, err := resolveSnapshotVersion(canvasUUID, allExecutions, organizationID)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeRunResponse{
		Run:              serializedRun,
		Executions:       serializedExecutions,
		ChildExecutions:  serializedChildren,
		SnapshotVersion:  snapshotVersion,
	}, nil
}

func splitParentAndChildExecutions(executions []models.CanvasNodeExecution) ([]models.CanvasNodeExecution, []models.CanvasNodeExecution) {
	parents := []models.CanvasNodeExecution{}
	children := []models.CanvasNodeExecution{}
	for _, execution := range executions {
		if execution.ParentExecutionID == nil {
			parents = append(parents, execution)
		} else {
			children = append(children, execution)
		}
	}
	return parents, children
}

// resolveSnapshotVersion picks the canvas version that was live when the run
// started. We prefer the CanvasVersionID stamped on any execution in the run
// (all executions for the same run share the same version). If none of the
// executions have one stamped (e.g. the run is from before this feature
// shipped), we fall back to the canvas's current live version.
func resolveSnapshotVersion(canvasID uuid.UUID, executions []models.CanvasNodeExecution, organizationID string) (*pb.CanvasVersion, error) {
	var versionID *uuid.UUID
	for i := range executions {
		if executions[i].CanvasVersionID != nil {
			id := *executions[i].CanvasVersionID
			versionID = &id
			break
		}
	}

	if versionID != nil {
		version, err := models.FindCanvasVersion(canvasID, *versionID)
		if err == nil {
			return SerializeCanvasVersion(version, organizationID), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return SerializeCanvasVersion(liveVersion, organizationID), nil
}
