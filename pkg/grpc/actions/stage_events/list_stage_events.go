package stageevents

import (
	"context"
	"errors"
	"fmt"
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
	DefaultLimit = 20
	MaxLimit     = 50
)

func ListStageEvents(ctx context.Context, canvasID string, stageIdOrName string, pbStates []pb.StageEvent_State, pbStateReasons []pb.StageEvent_StateReason, limit int32, before *timestamppb.Timestamp) (*pb.ListStageEventsResponse, error) {
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

	states, err := validateStageEventStates(pbStates)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	stateReasons, err := validateStageEventStateReasons(pbStateReasons)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	validatedLimit := validateLimit(int(limit))

	var beforeTime *time.Time
	if before != nil && before.IsValid() {
		t := before.AsTime()
		beforeTime = &t
	}

	events, err := stage.ListEventsWithLimitAndBefore(states, stateReasons, validatedLimit, beforeTime)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeStageEvents(events)
	if err != nil {
		return nil, err
	}

	response := &pb.ListStageEventsResponse{
		Events: serialized,
	}

	return response, nil
}

func validateStageEventStates(in []pb.StageEvent_State) ([]string, error) {
	//
	// If no states are provided, return all states.
	//
	if len(in) == 0 {
		return []string{
			models.StageEventStatePending,
			models.StageEventStateWaiting,
			models.StageEventStateProcessed,
		}, nil
	}

	states := []string{}
	for _, s := range in {
		state, err := protoToState(s)
		if err != nil {
			return nil, err
		}

		states = append(states, state)
	}

	return states, nil
}

func validateLimit(limit int) int {
	if limit < 1 || limit > MaxLimit {
		return DefaultLimit
	}
	return limit
}

func validateStageEventStateReasons(in []pb.StageEvent_StateReason) ([]string, error) {
	if len(in) == 0 {
		return []string{}, nil
	}

	stateReasons := []string{}
	for _, sr := range in {
		stateReason, err := protoToStateReason(sr)
		if err != nil {
			return nil, err
		}
		stateReasons = append(stateReasons, stateReason)
	}

	return stateReasons, nil
}

func protoToStateReason(stateReason pb.StageEvent_StateReason) (string, error) {
	switch stateReason {
	case pb.StageEvent_STATE_REASON_APPROVAL:
		return models.StageEventStateReasonApproval, nil
	case pb.StageEvent_STATE_REASON_TIME_WINDOW:
		return models.StageEventStateReasonTimeWindow, nil
	case pb.StageEvent_STATE_REASON_EXECUTION:
		return models.StageEventStateReasonExecution, nil
	case pb.StageEvent_STATE_REASON_CONNECTION:
		return models.StageEventStateReasonConnection, nil
	case pb.StageEvent_STATE_REASON_CANCELLED:
		return models.StageEventStateReasonCancelled, nil
	case pb.StageEvent_STATE_REASON_UNHEALTHY:
		return models.StageEventStateReasonUnhealthy, nil
	case pb.StageEvent_STATE_REASON_STUCK:
		return models.StageEventStateReasonStuck, nil
	case pb.StageEvent_STATE_REASON_TIMEOUT:
		return models.StageEventStateReasonTimeout, nil
	default:
		return "", fmt.Errorf("invalid state reason: %v", stateReason)
	}
}

func protoToState(state pb.StageEvent_State) (string, error) {
	switch state {
	case pb.StageEvent_STATE_PENDING:
		return models.StageEventStatePending, nil
	case pb.StageEvent_STATE_WAITING:
		return models.StageEventStateWaiting, nil
	case pb.StageEvent_STATE_PROCESSED:
		return models.StageEventStateProcessed, nil
	default:
		return "", fmt.Errorf("invalid state: %v", state)
	}
}

func serializeStageEvents(in []models.StageEvent) ([]*pb.StageEvent, error) {
	out := []*pb.StageEvent{}
	for _, i := range in {
		e, err := serializeStageEvent(i)
		if err != nil {
			return nil, err
		}

		out = append(out, e)
	}

	return out, nil
}

// TODO: very inefficient way of querying the approvals/execution that we should fix later
func serializeStageEvent(in models.StageEvent) (*pb.StageEvent, error) {
	e := pb.StageEvent{
		Id:          in.ID.String(),
		State:       stateToProto(in.State),
		StateReason: stateReasonToProto(in.StateReason),
		CreatedAt:   timestamppb.New(*in.CreatedAt),
		SourceId:    in.SourceID.String(),
		SourceType:  pb.Connection_TYPE_EVENT_SOURCE,
		Approvals:   []*pb.StageEventApproval{},
		Inputs:      []*pb.KeyValuePair{},
		Name:        in.Name,
		EventId:     in.EventID.String(),
	}

	//
	// Add execution
	//
	execution, err := serializeStageEventExecution(in)
	if err != nil {
		return nil, err
	}

	e.Execution = execution

	//
	// Add inputs
	//
	for k, v := range in.Inputs.Data() {
		e.Inputs = append(e.Inputs, &pb.KeyValuePair{Name: k, Value: v.(string)})
	}

	//
	// Add approvals
	//
	approvals, err := in.FindApprovals()
	if err != nil {
		return nil, err
	}

	for _, approval := range approvals {
		e.Approvals = append(e.Approvals, &pb.StageEventApproval{
			ApprovedBy: approval.ApprovedBy.String(),
			ApprovedAt: timestamppb.New(*approval.ApprovedAt),
		})
	}

	return &e, nil
}

func serializeStageEventExecution(event models.StageEvent) (*pb.Execution, error) {
	execution, err := models.FindExecutionByStageEventID(event.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		return nil, nil
	}

	e := &pb.Execution{
		Id:        execution.ID.String(),
		State:     executionStateToProto(execution.State),
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

	return e, nil
}

func executionStateToProto(state string) pb.Execution_State {
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

func stateToProto(state string) pb.StageEvent_State {
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

func stateReasonToProto(stateReason string) pb.StageEvent_StateReason {
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
