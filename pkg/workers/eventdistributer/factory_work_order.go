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
	NotifyPublicBoard(msg.FactoryId)
	return nil
}

const PublicBoardChangedEvent = "board_changed"

// PublicLineTopic is the hub key for guests watching one public line.
// It is not the member factory topic.
func PublicLineTopic(lineID string) string {
	return "public-line:" + lineID
}

// PublicBoardChangedMessage is the only payload a public line socket sends.
func PublicBoardChangedMessage() []byte {
	return []byte(`{"event":"board_changed"}`)
}

var (
	publicBoardHub         *ws.Hub
	publicBoardBroadcaster func(hub *ws.Hub, factoryID string)
)

// SetPublicBoardBroadcaster registers the guest-board fan-out. The member
// socket still receives factory id, order id, and reason.
func SetPublicBoardBroadcaster(hub *ws.Hub, broadcast func(hub *ws.Hub, factoryID string)) {
	publicBoardHub = hub
	publicBoardBroadcaster = broadcast
}

// NotifyPublicBoard tells public line sockets to refetch. It sends no card data.
func NotifyPublicBoard(factoryID string) {
	if publicBoardHub == nil || publicBoardBroadcaster == nil || factoryID == "" {
		return
	}
	publicBoardBroadcaster(publicBoardHub, factoryID)
}
