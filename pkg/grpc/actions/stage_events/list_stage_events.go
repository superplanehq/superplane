package stageevents

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	MinLimit     = 50
	MaxLimit     = 100
	DefaultLimit = 50
)

func ListStageEvents(ctx context.Context, canvasID string, stageIdOrName string, pbStates []pb.StageEvent_State, pbStateReasons []pb.StageEvent_StateReason, limit uint32, before *timestamppb.Timestamp) (*pb.ListStageEventsResponse, error) {
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

	limit = getLimit(limit)
	result := listAndCountEventsInParallel(stage, states, stateReasons, limit, getBefore(before))
	if result.listErr != nil {
		return nil, result.listErr
	}

	if result.countErr != nil {
		return nil, result.countErr
	}

	serialized, err := serializeStageEvents(result.events)
	if err != nil {
		return nil, err
	}

	response := &pb.ListStageEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(result.totalCount),
		HasNextPage:   result.hasNextPage(limit),
		LastTimestamp: result.lastTimestamp(),
	}

	return response, nil
}

type listAndCountEventsResult struct {
	events     []models.StageEvent
	totalCount int64
	listErr    error
	countErr   error
}

func (r *listAndCountEventsResult) hasNextPage(limit uint32) bool {
	return len(r.events) == int(limit) && r.totalCount > int64(limit)
}

func (r *listAndCountEventsResult) lastTimestamp() *timestamppb.Timestamp {
	if len(r.events) > 0 {
		lastEvent := r.events[len(r.events)-1]
		return timestamppb.New(*lastEvent.CreatedAt)
	}

	return nil
}

func listAndCountEventsInParallel(stage *models.Stage, states, stateReasons []string, limit uint32, beforeTime *time.Time) *listAndCountEventsResult {
	result := &listAndCountEventsResult{}
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		result.events, result.listErr = stage.ListEventsWithLimitAndBefore(states, stateReasons, int(limit), beforeTime)
	}()

	go func() {
		defer wg.Done()
		result.totalCount, result.countErr = stage.CountEvents(states, stateReasons)
	}()

	wg.Wait()
	return result
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
			models.StageEventStateDiscarded,
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
	case pb.StageEvent_STATE_DISCARDED:
		return models.StageEventStateDiscarded, nil
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
	}

	if in.DiscardedBy != nil {
		e.DiscardedBy = in.DiscardedBy.String()
	}
	if in.DiscardedAt != nil {
		e.DiscardedAt = timestamppb.New(*in.DiscardedAt)
	}

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

	if in.Event != nil {
		serializedTriggerEvent, err := serializeEvent(*in.Event)
		if err != nil {
			return nil, err
		}
		e.TriggerEvent = serializedTriggerEvent
	}

	return &e, nil
}

func stateToProto(state string) pb.StageEvent_State {
	switch state {
	case models.StageEventStatePending:
		return pb.StageEvent_STATE_PENDING
	case models.StageEventStateWaiting:
		return pb.StageEvent_STATE_WAITING
	case models.StageEventStateProcessed:
		return pb.StageEvent_STATE_PROCESSED
	case models.StageEventStateDiscarded:
		return pb.StageEvent_STATE_DISCARDED
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
	case models.StageEventStateReasonStuck:
		return pb.StageEvent_STATE_REASON_STUCK
	case models.StageEventStateReasonTimeout:
		return pb.StageEvent_STATE_REASON_TIMEOUT
	default:
		return pb.StageEvent_STATE_REASON_UNKNOWN
	}
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
