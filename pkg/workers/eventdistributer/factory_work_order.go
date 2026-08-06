package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

const (
	FactoryTopicPrefix    = "factory:"
	WorkOrderUpdatedEvent = "work_order_updated"
)

// FactoryWebsocketTopic is the hub key shared by publisher and subscriber.
func FactoryWebsocketTopic(factoryID string) string {
	return FactoryTopicPrefix + factoryID
}

type factoryWorkOrderUpdatedPayload struct {
	FactoryID string `json:"factoryId"`
	OrderID   string `json:"orderId,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type factoryWorkOrderWebsocketEvent struct {
	Event   string                         `json:"event"`
	Payload factoryWorkOrderUpdatedPayload `json:"payload"`
}

func HandleFactoryWorkOrderUpdated(messageBody []byte, wsHub *ws.Hub) error {
	pbMsg := &pb.FactoryWorkOrderUpdatedMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal factory work order updated: %w", err)
	}

	return BroadcastFactoryWorkOrderUpdated(wsHub, pbMsg)
}

func BroadcastFactoryWorkOrderUpdated(wsHub *ws.Hub, msg *pb.FactoryWorkOrderUpdatedMessage) error {
	if msg == nil || msg.FactoryId == "" {
		return fmt.Errorf("missing factoryId in factory work order updated")
	}

	payload, err := json.Marshal(factoryWorkOrderWebsocketEvent{
		Event: WorkOrderUpdatedEvent,
		Payload: factoryWorkOrderUpdatedPayload{
			FactoryID: msg.FactoryId,
			OrderID:   msg.OrderId,
			Reason:    msg.Reason,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal factory work order websocket event: %w", err)
	}

	topic := FactoryWebsocketTopic(msg.FactoryId)
	wsHub.BroadcastToWorkflow(topic, payload)
	log.Debugf("Broadcasted work_order_updated to factory %s (order %s, reason %s)", msg.FactoryId, msg.OrderId, msg.Reason)
	return nil
}
