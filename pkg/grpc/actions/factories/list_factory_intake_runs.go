package factories

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	defaultIntakeRunsLimit = 50
	maxIntakeRunsLimit     = 200
)

func ListFactoryIntakeRuns(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryIntakeRunsRequest,
) (*pb.ListFactoryIntakeRunsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	intake, err := factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultIntakeRunsLimit
	}
	if limit > maxIntakeRunsLimit {
		limit = maxIntakeRunsLimit
	}

	var before *time.Time
	if req.Before != nil {
		beforeTime := req.GetBefore().AsTime()
		before = &beforeTime
	}

	runs, err := models.ListCanvasRunsInTransaction(db, intake.CanvasID, limit, before, models.CanvasRunFilters{})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	serialized, err := serializeIntakeRuns(db, factory, intake, runs)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intake runs")
	}

	response := &pb.ListFactoryIntakeRunsResponse{Runs: serialized}
	if len(runs) > 0 {
		if last := runs[len(runs)-1].CreatedAt; last != nil {
			response.LastTimestamp = timestamppb.New(*last)
		}
	}

	return response, nil
}

// intakeRunContext is everything the page of runs needs, loaded in batch. Every
// map is keyed by run id.
type intakeRunContext struct {
	graph      intakeGraph
	rootEvents map[uuid.UUID]models.CanvasEvent
	analyses   map[uuid.UUID]models.CanvasNodeExecution
	creations  map[uuid.UUID]models.CanvasNodeExecution
	scores     map[uuid.UUID]int
	orders     map[uuid.UUID]models.FactoryWorkOrder
	stages     map[uuid.UUID]string
}

func serializeIntakeRuns(
	db *gorm.DB,
	factory *models.Factory,
	intake *models.FactoryIntake,
	runs []models.CanvasRun,
) ([]*pb.FactoryIntakeRun, error) {
	context, err := loadIntakeRunContext(db, factory, intake, runs)
	if err != nil {
		return nil, err
	}

	serialized := make([]*pb.FactoryIntakeRun, 0, len(runs))
	for _, run := range runs {
		title := intakeRunTitle(intake.Source, context.rootEvents[run.ID])
		if title == "" {
			continue
		}

		serializedRun := &pb.FactoryIntakeRun{
			Id:        run.ID.String(),
			Title:     title,
			Placement: intakeRunPlacement(run, context),
		}

		if score, ok := context.scores[run.ID]; ok {
			confidence := int32(score)
			serializedRun.ConfidencePct = &confidence
		}

		if order, ok := context.orders[run.ID]; ok {
			serializedRun.WorkOrderId = order.ID.String()
			serializedRun.Stage = context.stages[run.ID]
		}

		if run.CreatedAt != nil {
			serializedRun.CreatedAt = timestamppb.New(*run.CreatedAt)
		}
		if analysis, ok := context.analyses[run.ID]; ok && analysis.UpdatedAt != nil {
			serializedRun.AnalyzedAt = timestamppb.New(*analysis.UpdatedAt)
		}

		serialized = append(serialized, serializedRun)
	}

	return serialized, nil
}

func loadIntakeRunContext(
	db *gorm.DB,
	factory *models.Factory,
	intake *models.FactoryIntake,
	runs []models.CanvasRun,
) (intakeRunContext, error) {
	context := intakeRunContext{
		rootEvents: map[uuid.UUID]models.CanvasEvent{},
		analyses:   map[uuid.UUID]models.CanvasNodeExecution{},
		creations:  map[uuid.UUID]models.CanvasNodeExecution{},
		scores:     map[uuid.UUID]int{},
		orders:     map[uuid.UUID]models.FactoryWorkOrder{},
		stages:     map[uuid.UUID]string{},
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{intake.CanvasID})
	if err != nil {
		return context, err
	}
	context.graph = resolveIntakeGraph(intake.Source, specs[intake.CanvasID])

	runIDs := make([]uuid.UUID, len(runs))
	for i := range runs {
		runIDs[i] = runs[i].ID
	}

	context.rootEvents, err = models.ListRootEventsForRuns(db, intake.CanvasID, runIDs)
	if err != nil {
		return context, err
	}

	executions, err := models.ListExecutionsForRunsInTransaction(db, intake.CanvasID, runIDs)
	if err != nil {
		return context, err
	}

	analysisIDs := make([]uuid.UUID, 0, len(runs))
	for _, execution := range executions {
		switch execution.NodeID {
		case context.graph.AnalysisNodeID:
			context.analyses[execution.RunID] = execution
			analysisIDs = append(analysisIDs, execution.ID)
		case context.graph.CreateNodeID:
			context.creations[execution.RunID] = execution
		}
	}

	if err := loadIntakeScores(db, &context, analysisIDs); err != nil {
		return context, err
	}

	context.orders, err = models.ListWorkOrdersBySourceRunIDs(db, factory.ID, runIDs)
	if err != nil {
		return context, err
	}

	if err := loadIntakeStages(db, &context); err != nil {
		return context, err
	}

	return context, nil
}

// loadIntakeScores reads the score the agent returned. An execution's outputs
// are events, not columns, so they load as one batch keyed back to the run.
func loadIntakeScores(db *gorm.DB, context *intakeRunContext, analysisIDs []uuid.UUID) error {
	outputs, err := models.ListCanvasEventsForExecutionsInTransaction(db, analysisIDs)
	if err != nil {
		return err
	}

	runIDByExecutionID := make(map[uuid.UUID]uuid.UUID, len(context.analyses))
	for runID, analysis := range context.analyses {
		runIDByExecutionID[analysis.ID] = runID
	}

	for _, output := range outputs {
		if output.ExecutionID == nil {
			continue
		}

		runID, ok := runIDByExecutionID[*output.ExecutionID]
		if !ok {
			continue
		}
		if _, seen := context.scores[runID]; seen {
			continue
		}
		if score, ok := intakeScoreFromOutput(output.Data.Data()); ok {
			context.scores[runID] = score
		}
	}

	return nil
}

// loadIntakeStages reports how far the created work order travelled, so the
// intake list can distinguish "waiting in the backlog" from "already running".
func loadIntakeStages(db *gorm.DB, context *intakeRunContext) error {
	orderIDs := make([]uuid.UUID, 0, len(context.orders))
	runIDByOrderID := make(map[uuid.UUID]uuid.UUID, len(context.orders))
	for runID, order := range context.orders {
		orderIDs = append(orderIDs, order.ID)
		runIDByOrderID[order.ID] = runID
	}

	dispatches, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, orderIDs)
	if err != nil {
		return err
	}

	for orderID, records := range dispatches {
		if len(records) == 0 {
			continue
		}

		runID, ok := runIDByOrderID[orderID]
		if !ok {
			continue
		}
		context.stages[runID] = latestDispatchStage(records[len(records)-1])
	}

	return nil
}

func latestDispatchStage(record models.FactoryWorkOrderLineDispatchRecord) string {
	if record.QueueItem != nil && record.QueueItem.StepName != "" {
		return record.QueueItem.StepName
	}

	for i := len(record.Executions) - 1; i >= 0; i-- {
		if record.Executions[i].StepName != "" {
			return record.Executions[i].StepName
		}
	}

	return record.LineName
}

func intakeRunPlacement(run models.CanvasRun, context intakeRunContext) pb.FactoryIntakeRun_Placement {
	analysis, ok := context.analyses[run.ID]
	if !ok || analysis.State != models.CanvasNodeExecutionStateFinished {
		return pb.FactoryIntakeRun_PLACEMENT_ANALYZING
	}

	// A failed analysis is reported as a rejection: the intake looked at the
	// item and did not put it in the backlog.
	if analysis.Result == models.CanvasNodeExecutionResultFailed {
		return pb.FactoryIntakeRun_PLACEMENT_REJECTED
	}

	creation, ok := context.creations[run.ID]
	if !ok || creation.Result != models.CanvasNodeExecutionResultPassed {
		return pb.FactoryIntakeRun_PLACEMENT_BELOW_THRESHOLD
	}

	if context.stages[run.ID] != "" {
		return pb.FactoryIntakeRun_PLACEMENT_PROGRESSED
	}

	return pb.FactoryIntakeRun_PLACEMENT_BACKLOG
}

// intakeRunTitle reads the item's title out of the trigger payload. Each source
// nests it differently, and the trigger is the only place it exists.
func intakeRunTitle(source string, event models.CanvasEvent) string {
	payload, ok := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return ""
	}

	switch source {
	case models.FactoryIntakeSourceGitHubIssues:
		return nestedString(payload, "issue", "title")
	case models.FactoryIntakeSourceSentryExceptions:
		return nestedString(payload, "data", "issue", "title")
	case models.FactoryIntakeSourcePagerDutyIncidents:
		return nestedString(payload, "incident", "title")
	default:
		return ""
	}
}

// intakeScoreFromOutput digs the score out of an agent's output. The runner
// wraps its answer differently per harness, so the search prefers a "result"
// key and then falls back to the first number it finds.
func intakeScoreFromOutput(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return clampIntakeConfidence(int(typed)), true
	case int:
		return clampIntakeConfidence(typed), true
	case string:
		score, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return clampIntakeConfidence(int(score)), true
	case []any:
		for _, item := range typed {
			if score, ok := intakeScoreFromOutput(item); ok {
				return score, true
			}
		}
	case map[string]any:
		if result, ok := typed["result"]; ok {
			if score, ok := intakeScoreFromOutput(result); ok {
				return score, true
			}
		}
		for _, item := range typed {
			if score, ok := intakeScoreFromOutput(item); ok {
				return score, true
			}
		}
	}

	return 0, false
}

func nestedString(payload map[string]any, path ...string) string {
	current := any(payload)
	for _, key := range path {
		record, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = record[key]
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}
