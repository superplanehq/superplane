package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// FactoryWorkOrderUpdatedRoutingKey: factory mutations / run fan-out -> EventDistributer -> websocket clients.
	FactoryWorkOrderUpdatedRoutingKey = "factory-work-order-updated"
)

type FactoryWorkOrderUpdatedMessage struct {
	message *pb.FactoryWorkOrderUpdatedMessage
}

func NewFactoryWorkOrderUpdatedMessage(factoryID, orderID, reason string) FactoryWorkOrderUpdatedMessage {
	return FactoryWorkOrderUpdatedMessage{
		message: &pb.FactoryWorkOrderUpdatedMessage{
			FactoryId: factoryID,
			OrderId:   orderID,
			Reason:    reason,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m FactoryWorkOrderUpdatedMessage) Publish() error {
	return Publish(CanvasExchange, FactoryWorkOrderUpdatedRoutingKey, toBytes(m.message))
}

func PublishFactoryWorkOrderUpdated(factoryID, orderID, reason string) error {
	return NewFactoryWorkOrderUpdatedMessage(factoryID, orderID, reason).Publish()
}
