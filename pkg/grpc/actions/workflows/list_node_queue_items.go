package workflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListNodeQueueItems(ctx context.Context, registry *registry.Registry, workflowID, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeQueueItemsResponse, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
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

	serialized, err := SerializeNodeQueueItems(queueItems)
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

func SerializeNodeQueueItems(queueItems []models.WorkflowNodeQueueItem) ([]*pb.WorkflowNodeQueueItem, error) {
	//
	// Fetch all input records
	//
	inputEvents, err := models.FindWorkflowEvents(eventIDsFromQueueItems(queueItems))
	if err != nil {
		return nil, fmt.Errorf("error find input events: %v", err)
	}

	//
	// Combine everything into the response
	//
	result := make([]*pb.WorkflowNodeQueueItem, 0, len(queueItems))
	for _, queueItem := range queueItems {
		input, err := getInputForQueueItem(queueItem, inputEvents)
		if err != nil {
			return nil, err
		}

		serializedQueueItem := &pb.WorkflowNodeQueueItem{
			Id:         queueItem.ID.String(),
			WorkflowId: queueItem.WorkflowID.String(),
			NodeId:     queueItem.NodeID,
			CreatedAt:  timestamppb.New(*queueItem.CreatedAt),
			Input:      input,
		}

		if queueItem.RootEvent != nil {
			serializedQueueItem.RootEvent, err = SerializeWorkflowEvent(*queueItem.RootEvent)
			if err != nil {
				log.Errorf("Failed to serialize workflow event: %v", err)
				return nil, status.Error(codes.Internal, "failed to list node queue items")
			}
		}

		result = append(result, serializedQueueItem)
	}

	return result, nil
}

func getLastQueueItemTimestamp(queueItems []models.WorkflowNodeQueueItem) *timestamppb.Timestamp {
	if len(queueItems) > 0 {
		return timestamppb.New(*queueItems[len(queueItems)-1].CreatedAt)
	}
	return nil
}

func eventIDsFromQueueItems(queueItems []models.WorkflowNodeQueueItem) []string {
	ids := make([]string, len(queueItems))
	for i, queueItem := range queueItems {
		ids[i] = queueItem.EventID.String()
	}

	return ids
}

func getInputForQueueItem(queueItem models.WorkflowNodeQueueItem, events []models.WorkflowEvent) (*structpb.Struct, error) {
	for _, event := range events {
		if event.ID.String() == queueItem.EventID.String() {
			eventData, ok := event.Data.Data().(map[string]any)
			if !ok {
				return nil, fmt.Errorf("event data cannot be turned into input for queue item %s", queueItem.ID.String())
			}

			data, err := structpb.NewStruct(eventData)
			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, fmt.Errorf("input not found for queue item %s", queueItem.ID.String())
}
