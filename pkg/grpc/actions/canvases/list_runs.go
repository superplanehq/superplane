package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListRuns(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, limit uint32, before *timestamppb.Timestamp) (*pb.ListRunsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)
	runs, err := models.ListCanvasRuns(canvasID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	count, err := models.CountCanvasRuns(canvasID)
	if err != nil {
		return nil, err
	}

	runIDs := make([]uuid.UUID, len(runs))
	for i, run := range runs {
		runIDs[i] = run.ID
	}

	rootEventsByRunID, err := listRootEventsForRuns(canvasID, runIDs)
	if err != nil {
		return nil, err
	}

	executions, err := models.ListParentExecutionsForRunsInTransaction(database.Conn(), canvasID, runIDs)
	if err != nil {
		return nil, err
	}

	executionsByRunID := make(map[string][]models.CanvasNodeExecution, len(runIDs))
	for _, execution := range executions {
		executionsByRunID[execution.RunID.String()] = append(executionsByRunID[execution.RunID.String()], execution)
	}

	serialized, err := SerializeCanvasRuns(runs, rootEventsByRunID, executionsByRunID)
	if err != nil {
		return nil, err
	}

	return &pb.ListRunsResponse{
		Runs:          serialized,
		TotalCount:    uint32(count),
		HasNextPage:   hasNextPage(len(runs), int(limit), count),
		LastTimestamp: getLastRunTimestamp(runs),
	}, nil
}

func listRootEventsForRuns(canvasID uuid.UUID, runIDs []uuid.UUID) (map[string]models.CanvasEvent, error) {
	eventsByRunID := map[string]models.CanvasEvent{}
	if len(runIDs) == 0 {
		return eventsByRunID, nil
	}

	var events []models.CanvasEvent
	err := database.Conn().
		Where("workflow_id = ?", canvasID).
		Where("run_id IN ?", runIDs).
		Where("execution_id IS NULL").
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		eventsByRunID[event.RunID.String()] = event
	}

	return eventsByRunID, nil
}

func SerializeCanvasRuns(runs []models.CanvasRun, rootEventsByRunID map[string]models.CanvasEvent, executionsByRunID map[string][]models.CanvasNodeExecution) ([]*pb.CanvasRun, error) {
	result := make([]*pb.CanvasRun, 0, len(runs))

	for _, run := range runs {
		serializedRun, err := SerializeCanvasRun(run, rootEventsByRunID[run.ID.String()], executionsByRunID[run.ID.String()])
		if err != nil {
			return nil, err
		}

		result = append(result, serializedRun)
	}

	return result, nil
}

func SerializeCanvasRun(run models.CanvasRun, rootEvent models.CanvasEvent, executions []models.CanvasNodeExecution) (*pb.CanvasRun, error) {
	if rootEvent.ID == uuid.Nil {
		return nil, status.Error(codes.NotFound, "root event not found")
	}

	serializedRootEvent, err := SerializeCanvasEvent(rootEvent)
	if err != nil {
		return nil, err
	}

	executionRefs := make([]*pb.CanvasNodeExecutionRef, 0, len(executions))
	for _, execution := range executions {
		executionRefs = append(executionRefs, SerializeNodeExecutionRef(execution))
	}

	serialized := &pb.CanvasRun{
		Id:         run.ID.String(),
		CanvasId:   run.WorkflowID.String(),
		RootEvent:  serializedRootEvent,
		State:      RunStateToProto(run.State),
		Result:     RunResultToProto(run.Result),
		Executions: executionRefs,
		CreatedAt:  timestamppb.New(*run.CreatedAt),
		UpdatedAt:  timestamppb.New(*run.UpdatedAt),
	}

	if run.FinishedAt != nil {
		serialized.FinishedAt = timestamppb.New(*run.FinishedAt)
	}

	return serialized, nil
}

func RunStateToProto(state string) pb.CanvasRun_State {
	switch state {
	case models.CanvasRunStateStarted:
		return pb.CanvasRun_STATE_STARTED
	case models.CanvasRunStateFinished:
		return pb.CanvasRun_STATE_FINISHED
	default:
		return pb.CanvasRun_STATE_UNKNOWN
	}
}

func RunResultToProto(result string) pb.CanvasRun_Result {
	switch result {
	case models.CanvasRunResultPassed:
		return pb.CanvasRun_RESULT_PASSED
	case models.CanvasRunResultFailed:
		return pb.CanvasRun_RESULT_FAILED
	case models.CanvasRunResultCancelled:
		return pb.CanvasRun_RESULT_CANCELLED
	default:
		return pb.CanvasRun_RESULT_UNKNOWN
	}
}

func getLastRunTimestamp(runs []models.CanvasRun) *timestamppb.Timestamp {
	if len(runs) > 0 {
		return timestamppb.New(*runs[len(runs)-1].CreatedAt)
	}

	return nil
}
