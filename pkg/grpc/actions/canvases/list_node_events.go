package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListNodeEvents(ctx context.Context, db *gorm.DB, canvas *models.Canvas, nodeID string, limit uint32, before *timestamppb.Timestamp, beforeID string) (*pb.ListNodeEventsResponse, error) {
	limit = getLimit(limit)
	cursor := getCursor(before, beforeID)

	//
	// List and count events
	//
	events, err := models.ListCanvasEvents(db, canvas.ID, nodeID, int(limit), cursor)
	if err != nil {
		return nil, err
	}

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
		HasNextPage:   hasNextPage(len(events), int(limit), totalCount),
		LastTimestamp: getLastEventTimestamp(events),
		LastId:        getLastID(len(events), func() uuid.UUID { return events[len(events)-1].ID }),
	}, nil
}
