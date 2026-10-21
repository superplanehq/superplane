package canvases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListNodeQueueItems(ctx context.Context, db *gorm.DB, canvas *models.Canvas, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeQueueItemsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)

	//
	// List and count queue items
	//
	queueItems, err := models.ListNodeQueueItems(db, canvas.ID, nodeID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	totalCount, err := models.CountNodeQueueItems(db, canvas.ID, nodeID)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeQueueItems(db, canvas.ID, queueItems)
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

func SerializeNodeQueueItems(db *gorm.DB, workflowID uuid.UUID, queueItems []models.CanvasNodeQueueItem) ([]*pb.CanvasNodeQueueItem, error) {
	inputEvents, err := loadInputEventsForQueueItems(db, queueItems)
	if err != nil {
		return nil, err
	}

	blockingInfo, err := loadBlockingExecutionsInfo(db, workflowID, queueItems)
	if err != nil {
		return nil, err
	}

	return serializeNodeQueueItemsWithInputEvents(queueItems, inputEvents, blockingInfo)
}

type blockingExecutionsInfo struct {
	executionsByKey      map[string][]models.CanvasNodeExecution
	concurrencyMaxByNode map[string]int
}

func (b blockingExecutionsInfo) executionsFor(queueItem models.CanvasNodeQueueItem) []models.CanvasNodeExecution {
	if queueItem.QueueName == nil {
		return nil
	}

	return b.executionsByKey[blockingExecutionsKey(queueItem.NodeID, *queueItem.QueueName)]
}

func (b blockingExecutionsInfo) concurrencyMaxFor(queueItem models.CanvasNodeQueueItem) int {
	return b.concurrencyMaxByNode[queueItem.NodeID]
}

func loadBlockingExecutionsInfo(db *gorm.DB, workflowID uuid.UUID, queueItems []models.CanvasNodeQueueItem) (blockingExecutionsInfo, error) {
	nodeIDs := distinctNodeIDs(queueItems)
	queueNames := distinctResolvedQueueNames(queueItems)

	nodes, err := models.FindCanvasNodesByIDs(db, workflowID, nodeIDs)
	if err != nil {
		return blockingExecutionsInfo{}, fmt.Errorf("error listing nodes for queue items: %v", err)
	}

	concurrencyMaxByNode := make(map[string]int, len(nodes))
	for _, node := range nodes {
		concurrencyMaxByNode[node.NodeID] = node.ConcurrencySpec().EffectiveMax()
	}

	executions, err := models.ListActiveNodeExecutionsInQueues(db, workflowID, nodeIDs, queueNames)
	if err != nil {
		return blockingExecutionsInfo{}, fmt.Errorf("error listing blocking executions: %v", err)
	}

	executionsByKey := make(map[string][]models.CanvasNodeExecution, len(queueNames))
	for _, execution := range executions {
		if execution.QueueName == nil {
			continue
		}
		key := blockingExecutionsKey(execution.NodeID, *execution.QueueName)
		executionsByKey[key] = append(executionsByKey[key], execution)
	}

	return blockingExecutionsInfo{
		executionsByKey:      executionsByKey,
		concurrencyMaxByNode: concurrencyMaxByNode,
	}, nil
}

func blockingExecutionsKey(nodeID, queueName string) string {
	return nodeID + "\x00" + queueName
}

func distinctNodeIDs(queueItems []models.CanvasNodeQueueItem) []string {
	seen := make(map[string]struct{}, len(queueItems))
	ids := make([]string, 0, len(queueItems))

	for _, queueItem := range queueItems {
		if _, ok := seen[queueItem.NodeID]; ok {
			continue
		}
		seen[queueItem.NodeID] = struct{}{}
		ids = append(ids, queueItem.NodeID)
	}

	return ids
}

func distinctResolvedQueueNames(queueItems []models.CanvasNodeQueueItem) []string {
	seen := make(map[string]struct{}, len(queueItems))
	names := make([]string, 0, len(queueItems))

	for _, queueItem := range queueItems {
		if queueItem.QueueName == nil {
			continue
		}
		if _, ok := seen[*queueItem.QueueName]; ok {
			continue
		}
		seen[*queueItem.QueueName] = struct{}{}
		names = append(names, *queueItem.QueueName)
	}

	return names
}

func loadInputEventsForQueueItems(db *gorm.DB, queueItems []models.CanvasNodeQueueItem) ([]models.CanvasEvent, error) {
	inputEvents, err := models.FindCanvasEvents(db, eventIDsFromQueueItems(queueItems))
	if err != nil {
		return nil, fmt.Errorf("error find input events: %v", err)
	}

	return inputEvents, nil
}

func serializeNodeQueueItemsWithInputEvents(
	queueItems []models.CanvasNodeQueueItem,
	inputEvents []models.CanvasEvent,
	blockingInfo blockingExecutionsInfo,
) ([]*pb.CanvasNodeQueueItem, error) {
	inputEventsByID := indexEventsByID(inputEvents)
	result := make([]*pb.CanvasNodeQueueItem, 0, len(queueItems))
	for _, queueItem := range queueItems {
		input, err := getInputForQueueItem(queueItem, inputEventsByID)
		if err != nil {
			log.WithError(err).Warnf("Serializing queue item %s with empty input", queueItem.ID.String())
			input = &structpb.Struct{}
		}

		serializedQueueItem := &pb.CanvasNodeQueueItem{
			Id:             queueItem.ID.String(),
			CanvasId:       queueItem.WorkflowID.String(),
			NodeId:         queueItem.NodeID,
			CreatedAt:      timestamppb.New(*queueItem.CreatedAt),
			Input:          input,
			ConcurrencyMax: int32(blockingInfo.concurrencyMaxFor(queueItem)),
		}

		if queueItem.RootEvent != nil {
			serializedQueueItem.RootEvent, err = SerializeCanvasEvent(*queueItem.RootEvent)
			if err != nil {
				log.Errorf("Failed to serialize workflow event: %v", err)
				return nil, grpcerrors.Internal(err, "failed to list node queue items")
			}
		}

		for _, execution := range blockingInfo.executionsFor(queueItem) {
			serializedQueueItem.BlockingExecutions = append(
				serializedQueueItem.BlockingExecutions,
				SerializeNodeExecutionRef(execution, nil),
			)
		}

		result = append(result, serializedQueueItem)
	}

	return result, nil
}

func indexEventsByID(events []models.CanvasEvent) map[string]models.CanvasEvent {
	eventsByID := make(map[string]models.CanvasEvent, len(events))
	for _, event := range events {
		eventsByID[event.ID.String()] = event
	}

	return eventsByID
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

func getInputForQueueItem(queueItem models.CanvasNodeQueueItem, eventsByID map[string]models.CanvasEvent) (*structpb.Struct, error) {
	event, ok := eventsByID[queueItem.EventID.String()]
	if !ok {
		return nil, fmt.Errorf("input not found for queue item %s", queueItem.ID.String())
	}

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
