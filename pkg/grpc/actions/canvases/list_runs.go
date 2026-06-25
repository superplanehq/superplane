package canvases

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListRuns(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, limit uint32, before *timestamppb.Timestamp, states []pb.CanvasRun_State, results []pb.CanvasRun_Result) (*pb.ListRunsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)
	filters, err := buildCanvasRunFilters(states, results)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)

	var runs []models.CanvasRun
	err = telemetry.RunSpan(ctx, "runs.list", func(ctx context.Context) error {
		var listErr error
		runs, listErr = models.ListCanvasRunsInTransaction(database.DB(ctx), canvasID, int(limit), beforeTime, filters)
		return listErr
	})
	if err != nil {
		return nil, err
	}

	var count int64
	err = telemetry.RunSpan(ctx, "runs.count", func(ctx context.Context) error {
		var countErr error
		count, countErr = models.CountCanvasRunsInTransaction(database.DB(ctx), canvasID, filters)
		return countErr
	})
	if err != nil {
		return nil, err
	}

	runIDs := make([]uuid.UUID, len(runs))
	for i, run := range runs {
		runIDs[i] = run.ID
	}

	var rootEventsByRunID map[string]models.CanvasEvent
	err = telemetry.RunSpan(ctx, "runs.load_root_events", func(ctx context.Context) error {
		var loadErr error
		rootEventsByRunID, loadErr = listRootEventsForRuns(ctx, canvasID, runIDs)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	var executions []models.CanvasNodeExecution
	err = telemetry.RunSpan(ctx, "runs.load_executions", func(ctx context.Context) error {
		var loadErr error
		executions, loadErr = models.ListExecutionsForRunsInTransaction(db, canvasID, runIDs)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	executionsByRunID := make(map[string][]models.CanvasNodeExecution, len(runIDs))
	for _, execution := range executions {
		executionsByRunID[execution.RunID.String()] = append(executionsByRunID[execution.RunID.String()], execution)
	}

	serialized, err := serializeCanvasRuns(ctx, runs, rootEventsByRunID, executionsByRunID)
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

func buildCanvasRunFilters(states []pb.CanvasRun_State, results []pb.CanvasRun_Result) (models.CanvasRunFilters, error) {
	modelStates := make([]string, 0, len(states))
	for _, state := range states {
		modelState, err := ProtoRunStateToModel(state)
		if err != nil {
			return models.CanvasRunFilters{}, err
		}

		modelStates = append(modelStates, modelState)
	}

	modelResults := make([]string, 0, len(results))
	for _, result := range results {
		modelResult, err := ProtoRunResultToModel(result)
		if err != nil {
			return models.CanvasRunFilters{}, err
		}

		modelResults = append(modelResults, modelResult)
	}

	return models.CanvasRunFilters{
		States:  modelStates,
		Results: modelResults,
	}, nil
}

func listRootEventsForRuns(ctx context.Context, canvasID uuid.UUID, runIDs []uuid.UUID) (map[string]models.CanvasEvent, error) {
	eventsByRunID := map[string]models.CanvasEvent{}
	if len(runIDs) == 0 {
		return eventsByRunID, nil
	}

	var events []models.CanvasEvent
	err := database.DB(ctx).
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
		return nil, grpcerrors.NotFound(nil, "root event not found")
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
		VersionId:  run.VersionID.String(),
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

func ProtoRunStateToModel(state pb.CanvasRun_State) (string, error) {
	switch state {
	case pb.CanvasRun_STATE_STARTED:
		return models.CanvasRunStateStarted, nil
	case pb.CanvasRun_STATE_FINISHED:
		return models.CanvasRunStateFinished, nil
	default:
		return "", grpcerrors.InvalidArgument(nil, fmt.Sprintf("invalid run state filter: %s", state.String()))
	}
}

func ProtoRunResultToModel(result pb.CanvasRun_Result) (string, error) {
	switch result {
	case pb.CanvasRun_RESULT_PASSED:
		return models.CanvasRunResultPassed, nil
	case pb.CanvasRun_RESULT_FAILED:
		return models.CanvasRunResultFailed, nil
	case pb.CanvasRun_RESULT_CANCELLED:
		return models.CanvasRunResultCancelled, nil
	default:
		return "", grpcerrors.InvalidArgument(nil, fmt.Sprintf("invalid run result filter: %s", result.String()))
	}
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

func serializeCanvasRuns(
	ctx context.Context,
	runs []models.CanvasRun,
	rootEventsByRunID map[string]models.CanvasEvent,
	executionsByRunID map[string][]models.CanvasNodeExecution,
) ([]*pb.CanvasRun, error) {
	var serialized []*pb.CanvasRun
	err := telemetry.RunSpan(ctx, "runs.serialize", func(ctx context.Context) error {
		var serErr error
		serialized, serErr = SerializeCanvasRuns(runs, rootEventsByRunID, executionsByRunID)

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(attribute.Int("runs.count", len(runs)))
		}

		return serErr
	})
	if err != nil {
		return nil, err
	}

	return serialized, nil
}
