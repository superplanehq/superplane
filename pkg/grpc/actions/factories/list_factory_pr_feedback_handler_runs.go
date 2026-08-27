package factories

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	defaultPRFeedbackRunsLimit = 50
	maxPRFeedbackRunsLimit     = 200

	prFeedbackEventComment = "github.prComment"
	prFeedbackEventReview  = "github.prReview"
	prFeedbackEventReply   = "github.prReviewComment"
)

func ListFactoryPRFeedbackHandlerRuns(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryPRFeedbackHandlerRunsRequest,
) (*pb.ListFactoryPRFeedbackHandlerRunsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	handlerID, err := parsePRFeedbackHandlerID(req.GetHandlerId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	handler, err := factory.FindPRFeedbackHandler(db, handlerID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultPRFeedbackRunsLimit
	}
	if limit > maxPRFeedbackRunsLimit {
		limit = maxPRFeedbackRunsLimit
	}

	var before *time.Time
	if req.Before != nil {
		beforeTime := req.GetBefore().AsTime()
		before = &beforeTime
	}

	runs, err := models.ListCanvasRunsInTransaction(db, handler.CanvasID, limit, before, models.CanvasRunFilters{})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	serialized, err := serializePRFeedbackRuns(db, factory, handler, runs)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handler runs")
	}

	response := &pb.ListFactoryPRFeedbackHandlerRunsResponse{Runs: serialized}
	if len(runs) > 0 {
		if last := runs[len(runs)-1].CreatedAt; last != nil {
			response.LastTimestamp = timestamppb.New(*last)
		}
	}

	return response, nil
}

type prFeedbackRunContext struct {
	graph      prFeedbackGraph
	rootEvents map[uuid.UUID]models.CanvasEvent
	runners    map[uuid.UUID]models.CanvasNodeExecution
	orders     map[uuid.UUID]models.FactoryWorkOrder
}

func serializePRFeedbackRuns(
	db *gorm.DB,
	factory *models.Factory,
	handler *models.FactoryPRFeedbackHandler,
	runs []models.CanvasRun,
) ([]*pb.FactoryPRFeedbackHandlerRun, error) {
	context, err := loadPRFeedbackRunContext(db, factory, handler, runs)
	if err != nil {
		return nil, err
	}

	serialized := make([]*pb.FactoryPRFeedbackHandlerRun, 0, len(runs))
	for _, run := range runs {
		payload := prFeedbackRootPayload(context.rootEvents[run.ID])
		number := prFeedbackPullRequestNumber(payload)
		serializedRun := &pb.FactoryPRFeedbackHandlerRun{
			Id:                run.ID.String(),
			Title:             prFeedbackRunTitle(number),
			Repository:        nestedString(payload, "repository", "full_name"),
			PullRequestNumber: number,
			PullRequestUrl:    prFeedbackPullRequestURL(payload),
			Trigger:           prFeedbackRunTrigger(context.rootEvents[run.ID]),
			TriggerAuthor:     prFeedbackTriggerAuthor(context.rootEvents[run.ID], payload),
			TriggerUrl:        prFeedbackTriggerURL(context.rootEvents[run.ID], payload),
			Status:            prFeedbackRunStatus(run, context.runners[run.ID]),
		}

		if order, ok := context.orders[run.ID]; ok {
			serializedRun.WorkOrderId = order.ID.String()
		}
		if run.CreatedAt != nil {
			serializedRun.CreatedAt = timestamppb.New(*run.CreatedAt)
		}
		if startedAt := prFeedbackRunStartedAt(run, context.runners[run.ID]); startedAt != nil {
			serializedRun.StartedAt = startedAt
		}
		if run.FinishedAt != nil {
			serializedRun.FinishedAt = timestamppb.New(*run.FinishedAt)
		}

		serialized = append(serialized, serializedRun)
	}

	return serialized, nil
}

func loadPRFeedbackRunContext(
	db *gorm.DB,
	factory *models.Factory,
	handler *models.FactoryPRFeedbackHandler,
	runs []models.CanvasRun,
) (prFeedbackRunContext, error) {
	context := prFeedbackRunContext{
		rootEvents: map[uuid.UUID]models.CanvasEvent{},
		runners:    map[uuid.UUID]models.CanvasNodeExecution{},
		orders:     map[uuid.UUID]models.FactoryWorkOrder{},
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{handler.CanvasID})
	if err != nil {
		return context, err
	}
	context.graph = resolvePRFeedbackGraph(specs[handler.CanvasID])

	runIDs := make([]uuid.UUID, len(runs))
	for i := range runs {
		runIDs[i] = runs[i].ID
	}

	context.rootEvents, err = models.ListRootEventsForRuns(db, handler.CanvasID, runIDs)
	if err != nil {
		return context, err
	}

	executions, err := models.ListExecutionsForRunsInTransaction(db, handler.CanvasID, runIDs)
	if err != nil {
		return context, err
	}
	for _, execution := range executions {
		if context.graph.RunnerNodeID != "" && execution.NodeID == context.graph.RunnerNodeID {
			context.runners[execution.RunID] = execution
		}
	}

	keys := make([]string, 0, len(runs))
	keyByRunID := map[uuid.UUID]string{}
	for _, run := range runs {
		payload := prFeedbackRootPayload(context.rootEvents[run.ID])
		key := prFeedbackPullRequestURL(payload)
		if key == "" {
			continue
		}
		keys = append(keys, key)
		keyByRunID[run.ID] = key
	}

	ordersByKey, err := factory.ListWorkOrdersByArtifactKeys(db, keys)
	if err != nil {
		return context, err
	}
	for runID, key := range keyByRunID {
		if order, ok := ordersByKey[key]; ok {
			context.orders[runID] = order
		}
	}

	return context, nil
}

func prFeedbackRunTitle(number int64) string {
	if number <= 0 {
		return "Address feedback on PR"
	}
	return "Address feedback on PR #" + strconv.FormatInt(number, 10)
}

func prFeedbackRootPayload(event models.CanvasEvent) map[string]any {
	payload, ok := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return payload
}

func prFeedbackEventType(event models.CanvasEvent) string {
	envelope, ok := event.Data.Data().(map[string]any)
	if !ok {
		return ""
	}
	value, _ := envelope["type"].(string)
	return value
}

func prFeedbackRunTrigger(event models.CanvasEvent) pb.FactoryPRFeedbackHandlerRun_Trigger {
	switch prFeedbackEventType(event) {
	case prFeedbackEventComment:
		return pb.FactoryPRFeedbackHandlerRun_TRIGGER_PR_COMMENT
	case prFeedbackEventReview:
		return pb.FactoryPRFeedbackHandlerRun_TRIGGER_PR_REVIEW
	case prFeedbackEventReply:
		return pb.FactoryPRFeedbackHandlerRun_TRIGGER_PR_REVIEW_REPLY
	default:
		return pb.FactoryPRFeedbackHandlerRun_TRIGGER_UNSPECIFIED
	}
}

func prFeedbackTriggerAuthor(event models.CanvasEvent, payload map[string]any) string {
	switch prFeedbackEventType(event) {
	case prFeedbackEventReview:
		return nestedString(payload, "review", "user", "login")
	default:
		if author := nestedString(payload, "comment", "user", "login"); author != "" {
			return author
		}
		return nestedString(payload, "review", "user", "login")
	}
}

func prFeedbackTriggerURL(event models.CanvasEvent, payload map[string]any) string {
	switch prFeedbackEventType(event) {
	case prFeedbackEventReview:
		if url := nestedString(payload, "review", "html_url"); url != "" {
			return url
		}
	}
	if url := nestedString(payload, "comment", "html_url"); url != "" {
		return url
	}
	return prFeedbackPullRequestURL(payload)
}

func prFeedbackPullRequestURL(payload map[string]any) string {
	if url := nestedString(payload, "pull_request", "html_url"); url != "" {
		return url
	}
	return nestedString(payload, "issue", "pull_request", "html_url")
}

func prFeedbackPullRequestNumber(payload map[string]any) int64 {
	if number, ok := jsonInt64(nestedValue(payload, "pull_request", "number")); ok {
		return number
	}
	if number, ok := jsonInt64(nestedValue(payload, "issue", "number")); ok {
		return number
	}
	return 0
}

func prFeedbackRunStartedAt(run models.CanvasRun, runner models.CanvasNodeExecution) *timestamppb.Timestamp {
	if runner.CreatedAt != nil {
		return timestamppb.New(*runner.CreatedAt)
	}
	if run.CreatedAt != nil {
		return timestamppb.New(*run.CreatedAt)
	}
	return nil
}

func prFeedbackRunStatus(run models.CanvasRun, runner models.CanvasNodeExecution) pb.FactoryPRFeedbackHandlerRun_Status {
	if run.State == models.CanvasRunStateCancelling || run.Result == models.CanvasRunResultCancelled {
		return pb.FactoryPRFeedbackHandlerRun_STATUS_CANCELLED
	}
	if run.Result == models.CanvasRunResultFailed || runner.Result == models.CanvasNodeExecutionResultFailed {
		return pb.FactoryPRFeedbackHandlerRun_STATUS_FAILED
	}
	if run.State == models.CanvasRunStateFinished && run.Result == models.CanvasRunResultPassed {
		return pb.FactoryPRFeedbackHandlerRun_STATUS_PASSED
	}
	if runner.State == models.CanvasNodeExecutionStateStarted {
		return pb.FactoryPRFeedbackHandlerRun_STATUS_RUNNING
	}
	return pb.FactoryPRFeedbackHandlerRun_STATUS_QUEUED
}

func nestedValue(payload map[string]any, path ...string) any {
	current := any(payload)
	for _, key := range path {
		record, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = record[key]
	}
	return current
}

func jsonInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
