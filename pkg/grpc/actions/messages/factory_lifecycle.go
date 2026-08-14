package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// FactoryAppUpdatedRoutingKey: factory-scoped canvas mutations -> EventDistributer -> websocket clients.
	FactoryAppUpdatedRoutingKey = "factory-app-updated"
	// FactoryUpdatedRoutingKey: factory / line mutations -> EventDistributer -> websocket clients.
	FactoryUpdatedRoutingKey = "factory-updated"
)

type FactoryAppUpdatedMessage struct {
	message *pb.FactoryAppUpdatedMessage
}

func NewFactoryAppUpdatedMessage(factoryID, appID, reason string) FactoryAppUpdatedMessage {
	return FactoryAppUpdatedMessage{
		message: &pb.FactoryAppUpdatedMessage{
			FactoryId: factoryID,
			AppId:     appID,
			Reason:    reason,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m FactoryAppUpdatedMessage) Publish() error {
	return Publish(CanvasExchange, FactoryAppUpdatedRoutingKey, toBytes(m.message))
}

func PublishFactoryAppUpdated(factoryID, appID, reason string) error {
	return NewFactoryAppUpdatedMessage(factoryID, appID, reason).Publish()
}

type FactoryUpdatedMessage struct {
	message *pb.FactoryUpdatedMessage
}

func NewFactoryUpdatedMessage(factoryID, reason string) FactoryUpdatedMessage {
	return FactoryUpdatedMessage{
		message: &pb.FactoryUpdatedMessage{
			FactoryId: factoryID,
			Reason:    reason,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m FactoryUpdatedMessage) Publish() error {
	return Publish(CanvasExchange, FactoryUpdatedRoutingKey, toBytes(m.message))
}

func PublishFactoryUpdated(factoryID, reason string) error {
	return NewFactoryUpdatedMessage(factoryID, reason).Publish()
}
