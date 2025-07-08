package stageevents

import (
	"context"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListStageEvents(ctx context.Context, req *pb.ListStageEventsRequest) (*pb.ListStageEventsResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.StageIdOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = canvas.FindStageByName(req.StageIdOrName)
	} else {
		stage, err = canvas.FindStageByID(req.StageIdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	states, err := validateStageEventStates(req.States)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	events, err := stage.ListEvents(states, []string{})
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
		Message:     in.Message,
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
		Id:          execution.ID.String(),
		ReferenceId: execution.ReferenceID,
		State:       executionStateToProto(execution.State),
		Result:      actions.ExecutionResultToProto(execution.Result),
		CreatedAt:   timestamppb.New(*execution.CreatedAt),
		Outputs:     []*pb.OutputValue{},
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

	return e, nil
}

func executionStateToProto(state string) pb.Execution_State {
	switch state {
	case models.StageExecutionPending:
		return pb.Execution_STATE_PENDING
	case models.StageExecutionStarted:
		return pb.Execution_STATE_STARTED
	case models.StageExecutionFinished:
		return pb.Execution_STATE_FINISHED
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
	default:
		return pb.StageEvent_STATE_REASON_UNKNOWN
	}
}
