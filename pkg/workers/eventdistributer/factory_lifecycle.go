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
	FactoryAppUpdatedEvent = "factory_app_updated"
	FactoryUpdatedEvent    = "factory_updated"
)

type factoryAppUpdatedPayload struct {
	FactoryID string `json:"factoryId"`
	AppID     string `json:"appId,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type factoryAppWebsocketEvent struct {
	Event   string                   `json:"event"`
	Payload factoryAppUpdatedPayload `json:"payload"`
}

type factoryUpdatedPayload struct {
	FactoryID string `json:"factoryId"`
	Reason    string `json:"reason,omitempty"`
}

type factoryUpdatedWebsocketEvent struct {
	Event   string                `json:"event"`
	Payload factoryUpdatedPayload `json:"payload"`
}

func HandleFactoryAppUpdated(messageBody []byte, wsHub *ws.Hub) error {
	pbMsg := &pb.FactoryAppUpdatedMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal factory app updated: %w", err)
	}

	return BroadcastFactoryAppUpdated(wsHub, pbMsg)
}

func BroadcastFactoryAppUpdated(wsHub *ws.Hub, msg *pb.FactoryAppUpdatedMessage) error {
	if msg == nil || msg.FactoryId == "" {
		return fmt.Errorf("missing factoryId in factory app updated")
	}

	payload, err := json.Marshal(factoryAppWebsocketEvent{
		Event: FactoryAppUpdatedEvent,
		Payload: factoryAppUpdatedPayload{
			FactoryID: msg.FactoryId,
			AppID:     msg.AppId,
			Reason:    msg.Reason,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal factory app websocket event: %w", err)
	}

	topic := FactoryWebsocketTopic(msg.FactoryId)
	wsHub.BroadcastToWorkflow(topic, payload)
	log.Debugf("Broadcasted factory_app_updated to factory %s (app %s, reason %s)", msg.FactoryId, msg.AppId, msg.Reason)
	return nil
}

func HandleFactoryUpdated(messageBody []byte, wsHub *ws.Hub) error {
	pbMsg := &pb.FactoryUpdatedMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal factory updated: %w", err)
	}

	return BroadcastFactoryUpdated(wsHub, pbMsg)
}

func BroadcastFactoryUpdated(wsHub *ws.Hub, msg *pb.FactoryUpdatedMessage) error {
	if msg == nil || msg.FactoryId == "" {
		return fmt.Errorf("missing factoryId in factory updated")
	}

	payload, err := json.Marshal(factoryUpdatedWebsocketEvent{
		Event: FactoryUpdatedEvent,
		Payload: factoryUpdatedPayload{
			FactoryID: msg.FactoryId,
			Reason:    msg.Reason,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal factory updated websocket event: %w", err)
	}

	topic := FactoryWebsocketTopic(msg.FactoryId)
	wsHub.BroadcastToWorkflow(topic, payload)
	log.Debugf("Broadcasted factory_updated to factory %s (reason %s)", msg.FactoryId, msg.Reason)
	return nil
}
