package events

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListEvents(ctx context.Context, canvasID string, sourceType pb.EventSourceType, sourceID string) (*pb.ListEventsResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	_, err = models.FindCanvasByID(canvasID, orgUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "canvas not found")
		}
		return nil, err
	}

	sourceTypeStr := EventSourceTypeToString(sourceType)
	events, err := models.ListEventsByCanvasID(canvasUUID, sourceTypeStr, sourceID)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeEvents(events)
	if err != nil {
		return nil, err
	}

	response := &pb.ListEventsResponse{
		Events: serialized,
	}

	return response, nil
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
	rawStruct, err := structpb.NewStruct(map[string]interface{}{})
	if err == nil && len(in.Raw) > 0 {
		data, dataErr := in.GetData()
		if dataErr == nil {
			rawStruct, _ = structpb.NewStruct(data)
		}
	}

	headersStruct, err := structpb.NewStruct(map[string]interface{}{})
	if err == nil && len(in.Headers) > 0 {
		headers, headersErr := in.GetHeaders()
		if headersErr == nil {
			headersStruct, _ = structpb.NewStruct(headers)
		}
	}

	event := &pb.Event{
		Id:         in.ID.String(),
		SourceId:   in.SourceID.String(),
		SourceName: in.SourceName,
		SourceType: StringToEventSourceType(in.SourceType),
		Type:       in.Type,
		State:      in.State,
		ReceivedAt: timestamppb.New(*in.ReceivedAt),
		Raw:        rawStruct,
		Headers:    headersStruct,
	}

	return event, nil
}
