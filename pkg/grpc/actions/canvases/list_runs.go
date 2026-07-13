package canvases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListRuns(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, limit uint32, before *timestamppb.Timestamp, states []pb.CanvasRun_State, results []pb.CanvasRun_Result) (*pb.ListRunsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)
	filters, err := buildCanvasRunFilters(states, results)
	if err != nil {
		return nil, err
	}

	runs, err := listCanvasRuns(ctx, canvasID, int(limit), beforeTime, filters)
	if err != nil {
		return nil, err
	}

	count, err := countCanvasRuns(ctx, canvasID, filters)
	if err != nil {
		return nil, err
	}

	runIDs := make([]uuid.UUID, len(runs))
	for i, run := range runs {
		runIDs[i] = run.ID
	}

	runDetails, err := loadRunDetailsForRuns(ctx, canvasID, runIDs)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeCanvasRuns(ctx, runs, runDetails.rootEventsByRunID, runDetails.executionsByRunID, runDetails.queueItemsByRunID)
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

func SerializeCanvasRuns(
	db *gorm.DB,
	runs []models.CanvasRun,
	rootEventsByRunID map[string]models.CanvasEvent,
	executionsByRunID map[string][]models.CanvasNodeExecution,
	queueItemsByRunID map[string][]models.CanvasNodeQueueItem,
) ([]*pb.CanvasRun, error) {
	inputEvents, err := loadInputEventsForQueueItems(db, queueItemsForRuns(runs, queueItemsByRunID))
	if err != nil {
		return nil, err
	}

	result := make([]*pb.CanvasRun, 0, len(runs))

	for _, run := range runs {
		serializedRun, err := serializeCanvasRunWithQueueItemInputs(
			run,
			rootEventsByRunID[run.ID.String()],
			executionsByRunID[run.ID.String()],
			queueItemsByRunID[run.ID.String()],
			inputEvents,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, serializedRun)
	}

	return result, nil
}

func SerializeCanvasRun(
	db *gorm.DB,
	run models.CanvasRun,
	rootEvent models.CanvasEvent,
	executions []models.CanvasNodeExecution,
	queueItems []models.CanvasNodeQueueItem,
) (*pb.CanvasRun, error) {
	inputEvents, err := loadInputEventsForQueueItems(db, queueItems)
	if err != nil {
		return nil, err
	}

	return serializeCanvasRunWithQueueItemInputs(run, rootEvent, executions, queueItems, inputEvents)
}

func serializeCanvasRunWithQueueItemInputs(
	run models.CanvasRun,
	rootEvent models.CanvasEvent,
	executions []models.CanvasNodeExecution,
	queueItems []models.CanvasNodeQueueItem,
	inputEvents []models.CanvasEvent,
) (*pb.CanvasRun, error) {
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

	serializedQueueItems, err := serializeNodeQueueItemsWithInputEvents(queueItems, inputEvents)
	if err != nil {
		return nil, err
	}

	serialized := &pb.CanvasRun{
		Id:         run.ID.String(),
		CanvasId:   run.WorkflowID.String(),
		VersionId:  run.VersionID.String(),
		RootEvent:  serializedRootEvent,
		State:      RunStateToProto(run.State),
		Result:     RunResultToProto(run.Result),
		Executions: executionRefs,
		QueueItems: serializedQueueItems,
		CreatedAt:  timestamppb.New(*run.CreatedAt),
		UpdatedAt:  timestamppb.New(*run.UpdatedAt),
	}

	if run.FinishedAt != nil {
		serialized.FinishedAt = timestamppb.New(*run.FinishedAt)
	}

	return serialized, nil
}

func queueItemsForRuns(runs []models.CanvasRun, queueItemsByRunID map[string][]models.CanvasNodeQueueItem) []models.CanvasNodeQueueItem {
	var count int
	for _, run := range runs {
		count += len(queueItemsByRunID[run.ID.String()])
	}

	queueItems := make([]models.CanvasNodeQueueItem, 0, count)
	for _, run := range runs {
		queueItems = append(queueItems, queueItemsByRunID[run.ID.String()]...)
	}

	return queueItems
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

type runDetailsForRuns struct {
	rootEventsByRunID map[string]models.CanvasEvent
	executionsByRunID map[string][]models.CanvasNodeExecution
	queueItemsByRunID map[string][]models.CanvasNodeQueueItem
}

func loadRunDetailsForRuns(ctx context.Context, canvasID uuid.UUID, runIDs []uuid.UUID) (*runDetailsForRuns, error) {
	if len(runIDs) == 0 {
		return &runDetailsForRuns{
			rootEventsByRunID: map[string]models.CanvasEvent{},
			executionsByRunID: map[string][]models.CanvasNodeExecution{},
			queueItemsByRunID: map[string][]models.CanvasNodeQueueItem{},
		}, nil
	}

	var rootEventsByRunID map[string]models.CanvasEvent
	var executionsByRunID map[string][]models.CanvasNodeExecution
	var queueItemsByRunID map[string][]models.CanvasNodeQueueItem
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rootEventsByRunID, err = loadRootEventsForRunsSpan(gctx, canvasID, runIDs)
		if err != nil {
			return err
		}

		return nil
	})

	g.Go(func() error {
		executions, err := listExecutionsForRuns(gctx, canvasID, runIDs)
		if err != nil {
			return err
		}

		executionsByRunID = groupExecutionsByRunID(executions, len(runIDs))
		return nil
	})

	g.Go(func() error {
		var err error
		queueItemsByRunID, err = loadQueueItemsForRuns(gctx, canvasID, runIDs)
		if err != nil {
			return err
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &runDetailsForRuns{
		rootEventsByRunID: rootEventsByRunID,
		executionsByRunID: executionsByRunID,
		queueItemsByRunID: queueItemsByRunID,
	}, nil
}

func groupExecutionsByRunID(executions []models.CanvasNodeExecution, runCount int) map[string][]models.CanvasNodeExecution {
	executionsByRunID := make(map[string][]models.CanvasNodeExecution, runCount)
	for _, execution := range executions {
		executionsByRunID[execution.RunID.String()] = append(executionsByRunID[execution.RunID.String()], execution)
	}

	return executionsByRunID
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
	queueItemsByRunID map[string][]models.CanvasNodeQueueItem,
) (serialized []*pb.CanvasRun, err error) {
	ctx, done := telemetry.Span(ctx, "runs.serialize")
	defer done(&err)

	serialized, err = SerializeCanvasRuns(database.DB(ctx), runs, rootEventsByRunID, executionsByRunID, queueItemsByRunID)
	if err != nil {
		return nil, err
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int("runs.count", len(runs)))
	}

	return serialized, nil
}

func listCanvasRuns(ctx context.Context, canvasID uuid.UUID, limit int, beforeTime *time.Time, filters models.CanvasRunFilters) (runs []models.CanvasRun, err error) {
	ctx, done := telemetry.Span(ctx, "runs.list")
	defer done(&err)

	return models.ListCanvasRunsInTransaction(database.DB(ctx), canvasID, limit, beforeTime, filters)
}

func countCanvasRuns(ctx context.Context, canvasID uuid.UUID, filters models.CanvasRunFilters) (count int64, err error) {
	ctx, done := telemetry.Span(ctx, "runs.count")
	defer done(&err)

	return models.CountCanvasRunsInTransaction(database.DB(ctx), canvasID, filters)
}

func loadRootEventsForRunsSpan(ctx context.Context, canvasID uuid.UUID, runIDs []uuid.UUID) (events map[string]models.CanvasEvent, err error) {
	ctx, done := telemetry.Span(ctx, "runs.load_root_events")
	defer done(&err)

	return listRootEventsForRuns(ctx, canvasID, runIDs)
}

func listExecutionsForRuns(ctx context.Context, canvasID uuid.UUID, runIDs []uuid.UUID) (executions []models.CanvasNodeExecution, err error) {
	ctx, done := telemetry.Span(ctx, "runs.load_executions")
	defer done(&err)

	return models.ListExecutionsForRunsInTransaction(database.DB(ctx), canvasID, runIDs)
}

func loadQueueItemsForRuns(ctx context.Context, canvasID uuid.UUID, runIDs []uuid.UUID) (map[string][]models.CanvasNodeQueueItem, error) {
	queueItemsByRunID := make(map[string][]models.CanvasNodeQueueItem, len(runIDs))
	if len(runIDs) == 0 {
		return queueItemsByRunID, nil
	}

	queueItems, err := listQueueItemsForRuns(ctx, canvasID, runIDs)
	if err != nil {
		return nil, err
	}

	for _, queueItem := range queueItems {
		queueItemsByRunID[queueItem.RunID.String()] = append(queueItemsByRunID[queueItem.RunID.String()], queueItem)
	}

	return queueItemsByRunID, nil
}

func listQueueItemsForRuns(ctx context.Context, canvasID uuid.UUID, runIDs []uuid.UUID) (queueItems []models.CanvasNodeQueueItem, err error) {
	ctx, done := telemetry.Span(ctx, "runs.load_queue_items")
	defer done(&err)

	return models.ListNodeQueueItemsForRunsInTransaction(database.DB(ctx), canvasID, runIDs)
}
