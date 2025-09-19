package events

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateEvent(ctx context.Context, canvasID string, protoSourceType pb.EventSourceType, sourceID string, eventType string, raw map[string]any) (*pb.CreateEventResponse, error) {
	sourceType := actions.ProtoToEventSourceType(protoSourceType)
	if sourceType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid source type")
	}

	parsedCanvasID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	parsedSourceID, err := uuid.Parse(sourceID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid source ID")
	}

	if eventType == "" {
		return nil, status.Error(codes.InvalidArgument, "event type is required")
	}

	sourceName, err := findSourceName(parsedCanvasID, sourceType, parsedSourceID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "source not found: %v", err)
	}

	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid raw data: %v", err)
	}

	event, err := models.CreateEvent(parsedSourceID, parsedCanvasID, sourceName, sourceType, eventType, rawBytes, []byte(`{}`))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}

	serialized, err := actions.SerializeEvent(*event)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize event: %v", err)
	}

	return &pb.CreateEventResponse{
		Event: serialized,
	}, nil
}

func findSourceName(canvasID uuid.UUID, sourceType string, sourceID uuid.UUID) (string, error) {
	switch sourceType {
	case models.SourceTypeEventSource:
		source, err := models.FindExternalEventSourceByID(canvasID.String(), sourceID.String())
		if err != nil {
			return "", fmt.Errorf("event source %s not found: %w", sourceID.String(), err)
		}
		return source.Name, nil
	case models.SourceTypeStage:
		stage, err := models.FindStageByID(canvasID.String(), sourceID.String())
		if err != nil {
			return "", fmt.Errorf("stage %s not found: %w", sourceID.String(), err)
		}
		return stage.Name, nil
	case models.SourceTypeConnectionGroup:
		group, err := models.FindConnectionGroupByID(canvasID.String(), sourceID.String())
		if err != nil {
			return "", fmt.Errorf("connection group %s not found: %w", sourceID.String(), err)
		}
		return group.Name, nil
	default:
		return "", fmt.Errorf("unsupported source type: %s", sourceType)
	}
}