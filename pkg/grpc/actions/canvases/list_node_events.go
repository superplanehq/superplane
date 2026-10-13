package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListNodeEvents(ctx context.Context, db *gorm.DB, canvas *models.Canvas, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeEventsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)

	//
	// List and count events
	//
	events, err := models.ListCanvasEvents(db, canvas.ID, nodeID, int(limit)+1, beforeTime)
	if err != nil {
		return nil, err
	}

	events, hasNext := trimPage(events, int(limit))

	totalCount, err := models.CountCanvasEvents(db, canvas.ID, nodeID)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeCanvasEvents(events)
	if err != nil {
		return nil, err
	}

	return &pb.ListNodeEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNext,
		LastTimestamp: getLastEventTimestamp(events),
	}, nil
}
