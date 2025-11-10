package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

type QueueItemWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleWorkflowQueueItemCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received queue_item_created event")

	pbMsg := &pb.QueueItemCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal queue_item_created message: %w", err)
	}

	// Broadcast minimal payload; client will refetch the queue list.
	payloadJSON, _ := json.Marshal(map[string]string{
		"id":          pbMsg.Id,
		"workflow_id": pbMsg.WorkflowId,
		"node_id":     pbMsg.NodeId,
	})
	wsEvent, err := json.Marshal(QueueItemWebsocketEvent{Event: "queue_item_created", Payload: json.RawMessage(payloadJSON)})
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(pbMsg.WorkflowId, wsEvent)
	log.Debugf("Broadcasted queue_item_created to workflow %s", pbMsg.WorkflowId)
	return nil
}

func HandleWorkflowQueueItemDeleted(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received queue_item_deleted event")

	pbMsg := &pb.QueueItemDeleted{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal queue_item_deleted message: %w", err)
	}

	// Minimal payload for deletion
	minimal := map[string]string{
		"id":          pbMsg.Id,
		"workflow_id": pbMsg.WorkflowId,
		"node_id":     pbMsg.NodeId,
	}
	payloadJSON, _ := json.Marshal(minimal)
	wsEvent, err := json.Marshal(QueueItemWebsocketEvent{Event: "queue_item_deleted", Payload: json.RawMessage(payloadJSON)})
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(pbMsg.WorkflowId, wsEvent)
	log.Debugf("Broadcasted queue_item_deleted to workflow %s", pbMsg.WorkflowId)
	return nil
}

// No DB lookup to avoid races between create/delete.
