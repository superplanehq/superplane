package stages

import (
	"context"
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	DefaultExecutionLimit = 20
	MaxExecutionLimit     = 50
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

	validatedLimit := validateExecutionLimit(int(limit))

	var beforeTime *time.Time
	if before != nil && before.IsValid() {
		t := before.AsTime()
		beforeTime = &t
	}

	executions, err := stage.ListExecutionsWithLimitAndBefore(states, results, validatedLimit, beforeTime)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeExecutions(executions)
	if err != nil {
		return nil, err
	}

	response := &pb.ListStageExecutionsResponse{
		Executions: serialized,
	}

	return response, nil
}

func validateExecutionStates(in []pb.Execution_State) ([]string, error) {
	if len(in) == 0 {
		return []string{
			models.ExecutionPending,
			models.ExecutionStarted,
			models.ExecutionFinished,
			models.ExecutionCancelled,
		}, nil
	}

	states := []string{}
	for _, s := range in {
		state, err := executionStateProtoToModel(s)
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
		result, err := executionResultProtoToModel(r)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func validateExecutionLimit(limit int) int {
	if limit < 1 || limit > MaxExecutionLimit {
		return DefaultExecutionLimit
	}
	return limit
}

func executionStateProtoToModel(state pb.Execution_State) (string, error) {
	switch state {
	case pb.Execution_STATE_PENDING:
		return models.ExecutionPending, nil
	case pb.Execution_STATE_STARTED:
		return models.ExecutionStarted, nil
	case pb.Execution_STATE_FINISHED:
		return models.ExecutionFinished, nil
	case pb.Execution_STATE_CANCELLED:
		return models.ExecutionCancelled, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution state: %v", state)
	}
}

func executionResultProtoToModel(result pb.Execution_Result) (string, error) {
	switch result {
	case pb.Execution_RESULT_PASSED:
		return models.ResultPassed, nil
	case pb.Execution_RESULT_FAILED:
		return models.ResultFailed, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution result: %v", result)
	}
}

func executionStateModelToProto(state string) pb.Execution_State {
	switch state {
	case models.ExecutionPending:
		return pb.Execution_STATE_PENDING
	case models.ExecutionStarted:
		return pb.Execution_STATE_STARTED
	case models.ExecutionFinished:
		return pb.Execution_STATE_FINISHED
	case models.ExecutionCancelled:
		return pb.Execution_STATE_CANCELLED
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
		Id:        execution.ID.String(),
		State:     executionStateModelToProto(execution.State),
		Result:    actions.ExecutionResultToProto(execution.Result),
		CreatedAt: timestamppb.New(*execution.CreatedAt),
		Outputs:   []*pb.OutputValue{},
		Resources: []*pb.ExecutionResource{},
	}

	if execution.StartedAt != nil {
		e.StartedAt = timestamppb.New(*execution.StartedAt)
	}

	if execution.FinishedAt != nil {
		e.FinishedAt = timestamppb.New(*execution.FinishedAt)
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
			Id: r.ExternalID,
		})
	}

	if execution.EmittedEvent != nil {
		serializedEvent, err := serializeEvent(*execution.EmittedEvent)
		if err != nil {
			return nil, err
		}
		e.EmmitedEvent = serializedEvent
	}

	if execution.StageEvent != nil {
		serializedStageEvent, err := serializeStageEventForExecution(*execution.StageEvent)
		if err != nil {
			return nil, err
		}
		e.StageEvent = serializedStageEvent
	}

	return e, nil
}

func serializeEvent(event models.Event) (*pb.Event, error) {
	e := &pb.Event{
		Id:         event.ID.String(),
		SourceId:   event.SourceID.String(),
		SourceName: event.SourceName,
		SourceType: sourceTypeModelToProto(event.SourceType),
		Type:       event.Type,
		State:      eventStateModelToProto(event.State),
		ReceivedAt: timestamppb.New(*event.ReceivedAt),
	}

	if len(event.Raw) > 0 {
		data, err := event.GetData()
		if err != nil {
			return nil, err
		}

		e.Raw, err = structpb.NewStruct(data)
		if err != nil {
			return nil, err
		}
	}

	if len(event.Headers) > 0 {
		headers, err := event.GetHeaders()
		if err != nil {
			return nil, err
		}

		e.Headers, err = structpb.NewStruct(headers)
		if err != nil {
			return nil, err
		}
	}

	return e, nil
}

func serializeStageEventForExecution(stageEvent models.StageEvent) (*pb.StageEvent, error) {
	e := &pb.StageEvent{
		Id:          stageEvent.ID.String(),
		State:       stageEventStateModelToProto(stageEvent.State),
		StateReason: stageEventStateReasonModelToProto(stageEvent.StateReason),
		CreatedAt:   timestamppb.New(*stageEvent.CreatedAt),
		SourceId:    stageEvent.SourceID.String(),
		SourceType:  pb.Connection_TYPE_EVENT_SOURCE,
		Approvals:   []*pb.StageEventApproval{},
		Inputs:      []*pb.KeyValuePair{},
		Name:        stageEvent.Name,
	}

	if stageEvent.CancelledBy != nil {
		e.CancelledBy = stageEvent.CancelledBy.String()
	}
	if stageEvent.CancelledAt != nil {
		e.CancelledAt = timestamppb.New(*stageEvent.CancelledAt)
	}

	for k, v := range stageEvent.Inputs.Data() {
		e.Inputs = append(e.Inputs, &pb.KeyValuePair{Name: k, Value: v.(string)})
	}

	approvals, err := stageEvent.FindApprovals()
	if err != nil {
		return nil, err
	}

	for _, approval := range approvals {
		e.Approvals = append(e.Approvals, &pb.StageEventApproval{
			ApprovedBy: approval.ApprovedBy.String(),
			ApprovedAt: timestamppb.New(*approval.ApprovedAt),
		})
	}

	if stageEvent.Event != nil {
		serializedTriggerEvent, err := serializeEvent(*stageEvent.Event)
		if err != nil {
			return nil, err
		}
		e.TriggerEvent = serializedTriggerEvent
	}

	return e, nil
}

func sourceTypeModelToProto(sourceType string) pb.EventSourceType {
	switch sourceType {
	case models.SourceTypeEventSource:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE
	case models.SourceTypeStage:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_STAGE
	case models.SourceTypeConnectionGroup:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_CONNECTION_GROUP
	default:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN
	}
}

func eventStateModelToProto(state string) pb.Event_State {
	switch state {
	case models.EventStatePending:
		return pb.Event_STATE_PENDING
	case models.EventStateDiscarded:
		return pb.Event_STATE_DISCARDED
	case models.EventStateProcessed:
		return pb.Event_STATE_PROCESSED
	default:
		return pb.Event_STATE_UNKNOWN
	}
}

func stageEventStateModelToProto(state string) pb.StageEvent_State {
	switch state {
	case models.StageEventStatePending:
		return pb.StageEvent_STATE_PENDING
	case models.StageEventStateWaiting:
		return pb.StageEvent_STATE_WAITING
	case models.StageEventStateProcessed:
		return pb.StageEvent_STATE_PROCESSED
	default:
		return pb.StageEvent_STATE_UNKNOWN
	}
}

func stageEventStateReasonModelToProto(stateReason string) pb.StageEvent_StateReason {
	switch stateReason {
	case models.StageEventStateReasonApproval:
		return pb.StageEvent_STATE_REASON_APPROVAL
	case models.StageEventStateReasonTimeWindow:
		return pb.StageEvent_STATE_REASON_TIME_WINDOW
	case models.StageEventStateReasonExecution:
		return pb.StageEvent_STATE_REASON_EXECUTION
	case models.StageEventStateReasonConnection:
		return pb.StageEvent_STATE_REASON_CONNECTION
	case models.StageEventStateReasonCancelled:
		return pb.StageEvent_STATE_REASON_CANCELLED
	case models.StageEventStateReasonUnhealthy:
		return pb.StageEvent_STATE_REASON_UNHEALTHY
	case models.StageEventStateReasonStuck:
		return pb.StageEvent_STATE_REASON_STUCK
	case models.StageEventStateReasonTimeout:
		return pb.StageEvent_STATE_REASON_TIMEOUT
	default:
		return pb.StageEvent_STATE_REASON_UNKNOWN
	}
}