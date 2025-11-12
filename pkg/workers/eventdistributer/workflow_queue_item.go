package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/workflows"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type QueueItemWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

const (
	QueueItemCreatedEvent  = "queue_item_created"
	QueueItemConsumedEvent = "queue_item_consumed"
)

func HandleWorkflowQueueItemCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received queue_item_created event")

	pbMsg := &pb.WorkflowNodeQueueItemMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal queue_item_created message: %w", err)
	}

	return handleQueueItemState(pbMsg.WorkflowId, pbMsg.Id, wsHub, QueueItemCreatedEvent)
}

func HandleWorkflowQueueItemConsumed(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received queue_item_consumed event")

	pbMsg := &pb.WorkflowNodeQueueItemMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal queue_item_consumed message: %w", err)
	}

	return handleQueueItemState(pbMsg.WorkflowId, pbMsg.Id, wsHub, QueueItemConsumedEvent)
}

func handleQueueItemState(workflowID string, queueItemID string, wsHub *ws.Hub, eventName string) error {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return fmt.Errorf("failed to parse workflow id: %w", err)
	}

	queueItemUUID, err := uuid.Parse(queueItemID)
	if err != nil {
		return fmt.Errorf("failed to parse queue item id: %w", err)
	}

	queueItem, err := models.FindNodeQueueItem(workflowUUID, queueItemUUID)
	if err != nil {
		return fmt.Errorf("failed to find queue item: %w", err)
	}

	serializedQueueItems, err := workflows.SerializeNodeQueueItems([]models.WorkflowNodeQueueItem{*queueItem})
	if err != nil {
		return fmt.Errorf("failed to serialize queue item: %w", err)
	}

	if len(serializedQueueItems) == 0 {
		return fmt.Errorf("no serialized queue items")
	}

	serializedQueueItemJSON, err := protojson.Marshal(serializedQueueItems[0])
	if err != nil {
		return fmt.Errorf("failed to marshal queue item: %w", err)
	}

	wsEvent, err := json.Marshal(QueueItemWebsocketEvent{
		Event:   eventName,
		Payload: json.RawMessage(serializedQueueItemJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(workflowID, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, workflowID)

	return nil
}