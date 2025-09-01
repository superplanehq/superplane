package events

import (
	"context"
	"fmt"
	"strings"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func BulkListEvents(ctx context.Context, canvasID string, sources []*pb.EventSourceItemRequest, limitPerSource int32) (*pb.BulkListEventsResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	sourceFilters := make([]models.SourceFilter, 0, len(sources))
	for _, source := range sources {
		sourceFilters = append(sourceFilters, models.SourceFilter{
			SourceType: EventSourceTypeToString(source.SourceType),
			SourceID:   source.SourceId,
		})
	}

	eventsBySource, err := models.BulkListEventsByCanvasIDAndMultipleSources(canvasUUID, sourceFilters, int(limitPerSource))
	if err != nil {
		return nil, err
	}

	results := make([]*pb.EventSourceItemResult, 0, len(sources))
	for _, source := range sources {
		var sourceEvents []models.Event

		if source.SourceId != "" {
			sourceKey := fmt.Sprintf("%s|%s", EventSourceTypeToString(source.SourceType), source.SourceId)
			sourceEvents = eventsBySource[sourceKey]
		} else {
			sourceTypeStr := EventSourceTypeToString(source.SourceType)
			for key, events := range eventsBySource {
				if strings.HasPrefix(key, sourceTypeStr+"|") {
					sourceEvents = append(sourceEvents, events...)
				}
			}
		}

		serializedEvents, err := serializeEvents(sourceEvents)
		if err != nil {
			return nil, err
		}

		result := &pb.EventSourceItemResult{
			SourceId:   source.SourceId,
			SourceType: source.SourceType,
			Events:     serializedEvents,
		}

		results = append(results, result)
	}

	response := &pb.BulkListEventsResponse{
		Results: results,
	}

	return response, nil
}
