package canvases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListNodeQueueItems(ctx context.Context, registry *registry.Registry, workflowID, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeQueueItemsResponse, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)

	//
	// List and count queue items
	//
	queueItems, err := models.ListNodeQueueItems(wfID, nodeID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	totalCount, err := models.CountNodeQueueItems(wfID, nodeID)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeQueueItems(database.DB(ctx), queueItems)
	if err != nil {
		return nil, err
	}

	return &pb.ListNodeQueueItemsResponse{
		Items:         serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNextPage(len(queueItems), int(limit), totalCount),
		LastTimestamp: getLastQueueItemTimestamp(queueItems),
	}, nil
}

func SerializeNodeQueueItems(db *gorm.DB, queueItems []models.CanvasNodeQueueItem) ([]*pb.CanvasNodeQueueItem, error) {
	//
	// Fetch all input records
	//
	inputEvents, err := models.FindCanvasEvents(db, eventIDsFromQueueItems(queueItems))
	if err != nil {
		return nil, fmt.Errorf("error find input events: %v", err)
	}

	//
	// Combine everything into the response
	//
	result := make([]*pb.CanvasNodeQueueItem, 0, len(queueItems))
	for _, queueItem := range queueItems {
		input, err := getInputForQueueItem(queueItem, inputEvents)
		if err != nil {
			return nil, err
		}

		serializedQueueItem := &pb.CanvasNodeQueueItem{
			Id:        queueItem.ID.String(),
			CanvasId:  queueItem.WorkflowID.String(),
			NodeId:    queueItem.NodeID,
			CreatedAt: timestamppb.New(*queueItem.CreatedAt),
			Input:     input,
		}

		if queueItem.RootEvent != nil {
			serializedQueueItem.RootEvent, err = SerializeCanvasEvent(*queueItem.RootEvent)
			if err != nil {
				log.Errorf("Failed to serialize workflow event: %v", err)
				return nil, grpcerrors.Internal(err, "failed to list node queue items")
			}
		}

		result = append(result, serializedQueueItem)
	}

	return result, nil
}

func getLastQueueItemTimestamp(queueItems []models.CanvasNodeQueueItem) *timestamppb.Timestamp {
	if len(queueItems) > 0 {
		return timestamppb.New(*queueItems[len(queueItems)-1].CreatedAt)
	}
	return nil
}

func eventIDsFromQueueItems(queueItems []models.CanvasNodeQueueItem) []string {
	ids := make([]string, len(queueItems))
	for i, queueItem := range queueItems {
		ids[i] = queueItem.EventID.String()
	}

	return ids
}

func getInputForQueueItem(queueItem models.CanvasNodeQueueItem, events []models.CanvasEvent) (*structpb.Struct, error) {
	for _, event := range events {
		if event.ID.String() == queueItem.EventID.String() {
			eventData, ok := event.Data.Data().(map[string]any)
			if !ok {
				return nil, fmt.Errorf("event data cannot be turned into input for queue item %s", queueItem.ID.String())
			}

			data, err := newStructpbStruct(eventData)
			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, fmt.Errorf("input not found for queue item %s", queueItem.ID.String())
}
