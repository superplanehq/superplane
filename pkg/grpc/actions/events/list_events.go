package events

import (
	"context"
	"sync"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	MinLimit     = 50
	MaxLimit     = 100
	DefaultLimit = 50
)

func ListEvents(ctx context.Context, canvasID string, sourceType pb.EventSourceType, sourceID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListEventsResponse, error) {
	limit = getLimit(limit)
	result := listAndCountEventsInParallel(
		uuid.MustParse(canvasID),
		actions.ProtoToEventSourceType(sourceType),
		sourceID,
		int(limit),
		getBefore(before),
	)

	if result.listErr != nil {
		return nil, result.listErr
	}

	if result.countErr != nil {
		return nil, result.countErr
	}

	serialized, err := serializeEvents(result.events)
	if err != nil {
		return nil, err
	}

	response := &pb.ListEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(result.totalCount),
		HasNextPage:   result.hasNextPage(limit),
		LastTimestamp: result.lastTimestamp(),
	}

	return response, nil
}

type listAndCountEventsResult struct {
	events     []models.Event
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
		return timestamppb.New(*lastEvent.ReceivedAt)
	}

	return nil
}

func listAndCountEventsInParallel(canvasID uuid.UUID, sourceType, sourceID string, limit int, beforeTime *time.Time) *listAndCountEventsResult {
	result := &listAndCountEventsResult{}
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		result.events, result.listErr = models.FilterEvents(canvasID, sourceType, sourceID, limit, beforeTime)
	}()

	go func() {
		defer wg.Done()
		result.totalCount, result.countErr = models.CountEvents(canvasID, sourceType, sourceID)
	}()

	wg.Wait()
	return result
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

func EventStateProtoToString(state pb.Event_State) string {
	switch state {
	case pb.Event_STATE_PROCESSED:
		return models.EventStateProcessed
	case pb.Event_STATE_PENDING:
		return models.EventStatePending
	case pb.Event_STATE_DISCARDED:
		return models.EventStateDiscarded
	default:
		return ""
	}
}

func StringToEventStateProto(state string) pb.Event_State {
	switch state {
	case models.EventStateProcessed:
		return pb.Event_STATE_PROCESSED
	case models.EventStatePending:
		return pb.Event_STATE_PENDING
	case models.EventStateDiscarded:
		return pb.Event_STATE_DISCARDED
	default:
		return pb.Event_STATE_UNKNOWN
	}
}

func serializeEvents(in []models.Event) ([]*pb.Event, error) {
	out := []*pb.Event{}
	for _, event := range in {
		serialized, err := serializeEvent(event)
		if err != nil {
			return nil, err
		}
		out = append(out, serialized)
	}
	return out, nil
}

func serializeEvent(in models.Event) (*pb.Event, error) {
	event := &pb.Event{
		Id:         in.ID.String(),
		SourceId:   in.SourceID.String(),
		SourceName: in.SourceName,
		SourceType: actions.EventSourceTypeToProto(in.SourceType),
		Type:       in.Type,
		State:      actions.EventStateToProto(in.State),
		ReceivedAt: timestamppb.New(*in.ReceivedAt),
	}

	if len(in.Raw) > 0 {
		data, err := in.GetData()
		if err != nil {
			return nil, err
		}

		event.Raw, err = structpb.NewStruct(data)

		if err != nil {
			return nil, err
		}
	}

	if len(in.Headers) > 0 {
		headers, err := in.GetHeaders()
		if err != nil {
			return nil, err
		}

		event.Headers, err = structpb.NewStruct(headers)

		if err != nil {
			return nil, err
		}
	}

	return event, nil
}
