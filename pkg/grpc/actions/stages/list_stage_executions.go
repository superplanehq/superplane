package stages

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	MinLimit     = 50
	MaxLimit     = 100
	DefaultLimit = 50
)

func ListStageExecutions(ctx context.Context, canvasID string, stageIdOrName string, pbStates []pb.Execution_State, pbResults []pb.Execution_Result, limit uint32, before *timestamppb.Timestamp) (*pb.ListStageExecutionsResponse, error) {
	err := actions.ValidateUUIDs(stageIdOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = models.FindStageByName(canvasID, stageIdOrName)
	} else {
		stage, err = models.FindStageByID(canvasID, stageIdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}
		return nil, err
	}

	states, err := validateExecutionStates(pbStates)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	results, err := validateExecutionResults(pbResults)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	limit = getLimit(limit)
	result := listAndCountExecutionsInParallel(stage, states, results, limit, getBefore(before))
	if result.listErr != nil {
		return nil, result.listErr
	}

	if result.countErr != nil {
		return nil, result.countErr
	}

	serialized, err := serializeExecutions(result.executions)
	if err != nil {
		return nil, err
	}

	response := &pb.ListStageExecutionsResponse{
		Executions:    serialized,
		TotalCount:    uint32(result.totalCount),
		HasNextPage:   result.HasNextPage(limit),
		LastTimestamp: result.LastTimestamp(),
	}

	return response, nil
}

type listAndCountResult struct {
	executions []models.StageExecution
	totalCount int64
	listErr    error
	countErr   error
}

func (r *listAndCountResult) HasNextPage(limit uint32) bool {
	return len(r.executions) == int(limit) && r.totalCount > int64(limit)
}

func (r *listAndCountResult) LastTimestamp() *timestamppb.Timestamp {
	if len(r.executions) > 0 {
		lastExecution := r.executions[len(r.executions)-1]
		return timestamppb.New(*lastExecution.CreatedAt)
	}
	return nil
}

func listAndCountExecutionsInParallel(stage *models.Stage, states, results []string, limit uint32, beforeTime *time.Time) *listAndCountResult {
	result := &listAndCountResult{}
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		result.executions, result.listErr = stage.FilterExecutions(states, results, int(limit), beforeTime)
	}()

	go func() {
		defer wg.Done()
		result.totalCount, result.countErr = stage.CountExecutions(states, results)
	}()

	wg.Wait()
	return result
}

func validateExecutionStates(in []pb.Execution_State) ([]string, error) {
	if len(in) == 0 {
		return []string{
			models.ExecutionPending,
			models.ExecutionStarted,
			models.ExecutionFinished,
		}, nil
	}

	states := []string{}
	for _, s := range in {
		state, err := ProtoToExecutionState(s)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

func validateExecutionResults(in []pb.Execution_Result) ([]string, error) {
	if len(in) == 0 {
		return []string{}, nil
	}

	results := []string{}
	for _, r := range in {
		result, err := ProtoToExecutionResult(r)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func getLimit(limit uint32) uint32 {
	if limit == 0 {
		return DefaultLimit
	}

	if limit > MaxLimit {
		return MaxLimit
	}

	return limit
}

func getBefore(before *timestamppb.Timestamp) *time.Time {
	if before != nil && before.IsValid() {
		t := before.AsTime()
		return &t
	}

	return nil
}

func ProtoToExecutionState(state pb.Execution_State) (string, error) {
	switch state {
	case pb.Execution_STATE_PENDING:
		return models.ExecutionPending, nil
	case pb.Execution_STATE_STARTED:
		return models.ExecutionStarted, nil
	case pb.Execution_STATE_FINISHED:
		return models.ExecutionFinished, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution state: %v", state)
	}
}

func ProtoToExecutionResult(result pb.Execution_Result) (string, error) {
	switch result {
	case pb.Execution_RESULT_PASSED:
		return models.ResultPassed, nil
	case pb.Execution_RESULT_FAILED:
		return models.ResultFailed, nil
	case pb.Execution_RESULT_CANCELLED:
		return models.ResultCancelled, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution result: %v", result)
	}
}

func ExecutionStateToProto(state string) pb.Execution_State {
	switch state {
	case models.ExecutionPending:
		return pb.Execution_STATE_PENDING
	case models.ExecutionStarted:
		return pb.Execution_STATE_STARTED
	case models.ExecutionFinished:
		return pb.Execution_STATE_FINISHED
	default:
		return pb.Execution_STATE_UNKNOWN
	}
}

func serializeExecutions(executions []models.StageExecution) ([]*pb.Execution, error) {
	result := []*pb.Execution{}
	for _, execution := range executions {
		serialized, err := serializeExecution(execution)
		if err != nil {
			return nil, err
		}
		result = append(result, serialized)
	}
	return result, nil
}

func serializeExecution(execution models.StageExecution) (*pb.Execution, error) {
	e := &pb.Execution{
		Id:            execution.ID.String(),
		State:         ExecutionStateToProto(execution.State),
		Result:        actions.ExecutionResultToProto(execution.Result),
		ResultReason:  actions.ExecutionResultReasonToProto(execution.ResultReason),
		ResultMessage: execution.ResultMessage,
		CreatedAt:     timestamppb.New(*execution.CreatedAt),
		Outputs:       []*pb.OutputValue{},
		Resources:     []*pb.ExecutionResource{},
	}

	if execution.StartedAt != nil {
		e.StartedAt = timestamppb.New(*execution.StartedAt)
	}

	if execution.FinishedAt != nil {
		e.FinishedAt = timestamppb.New(*execution.FinishedAt)
	}

	if execution.CancelledAt != nil {
		e.CancelledAt = timestamppb.New(*execution.CancelledAt)
	}

	for k, v := range execution.Outputs.Data() {
		e.Outputs = append(e.Outputs, &pb.OutputValue{Name: k, Value: v.(string)})
	}

	resources, err := execution.Resources()
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		e.Resources = append(e.Resources, &pb.ExecutionResource{
			Id:     r.ExternalID,
			Type:   r.ResourceType,
			State:  r.State,
			Result: r.Result,
			Url:    r.ResourceURL,
		})
	}

	if execution.StageEvent != nil {
		serializedStageEvent, err := actions.SerializeStageEvent(*execution.StageEvent)
		if err != nil {
			return nil, err
		}
		e.StageEvent = serializedStageEvent
	}

	return e, nil
}
