package events

import (
	"context"
	"sync"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type listAndCountEventsResult struct {
	events     []models.Event
	totalCount int64
	listErr    error
	countErr   error
}

func listAndCountEventsInParallel(canvasID uuid.UUID, sourceType, sourceIDStr string, limit int, beforeTime *time.Time) *listAndCountEventsResult {
	result := &listAndCountEventsResult{}
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		result.events, result.listErr = models.ListEventsByCanvasIDWithLimitAndBefore(canvasID, sourceType, sourceIDStr, limit, beforeTime)
	}()

	go func() {
		defer wg.Done()
		result.totalCount, result.countErr = models.CountEventsByCanvasIDAndFilters(canvasID, sourceType, sourceIDStr)
	}()

	wg.Wait()
	return result
}

func ListEvents(ctx context.Context, canvasID string, sourceType pb.EventSourceType, sourceID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListEventsResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	validatedLimit := validateLimit(int(limit))

	var beforeTime *time.Time
	if before != nil && before.IsValid() {
		t := before.AsTime()
		beforeTime = &t
	}

	sourceTypeStr := EventSourceTypeToString(sourceType)
	result := listAndCountEventsInParallel(canvasUUID, sourceTypeStr, sourceID, validatedLimit, beforeTime)

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

	hasNextPage := int64(len(result.events)) == int64(validatedLimit) && result.totalCount > int64(validatedLimit)

	var lastTimestamp *timestamppb.Timestamp
	if len(result.events) > 0 {
		lastEvent := result.events[len(result.events)-1]
		lastTimestamp = timestamppb.New(*lastEvent.ReceivedAt)
	}

	response := &pb.ListEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(result.totalCount),
		HasNextPage:   hasNextPage,
		LastTimestamp: lastTimestamp,
	}

	return response, nil
}

func validateLimit(limit int) int {
	if limit < 1 || limit > MaxLimit {
		return DefaultLimit
	}
	return limit
}

func EventSourceTypeToString(sourceType pb.EventSourceType) string {
	switch sourceType {
	case pb.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE:
		return models.SourceTypeEventSource
	case pb.EventSourceType_EVENT_SOURCE_TYPE_STAGE:
		return models.SourceTypeStage
	case pb.EventSourceType_EVENT_SOURCE_TYPE_CONNECTION_GROUP:
		return models.SourceTypeConnectionGroup
	default:
		return ""
	}
}

func StringToEventSourceType(sourceType string) pb.EventSourceType {
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
		SourceType: StringToEventSourceType(in.SourceType),
		Type:       in.Type,
		State:      StringToEventStateProto(in.State),
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
