package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FactoryVelocitySyncRequestedRoutingKey: a user asked for a fresh velocity
// report -> FactoryVelocitySyncWorker reads the repository now.
//
// The velocity report is served from stored merges a background worker collects,
// and that worker does not run in the API process. Handing the request over
// keeps integration traffic out of the API and survives an API restart.
const FactoryVelocitySyncRequestedRoutingKey = "factory-velocity-sync-requested"

type FactoryVelocitySyncRequestedMessage struct {
	message *pb.FactoryVelocitySyncRequestedMessage
}

func NewFactoryVelocitySyncRequestedMessage(factoryID string) FactoryVelocitySyncRequestedMessage {
	return FactoryVelocitySyncRequestedMessage{
		message: &pb.FactoryVelocitySyncRequestedMessage{
			FactoryId: factoryID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m FactoryVelocitySyncRequestedMessage) Publish() error {
	return Publish(CanvasExchange, FactoryVelocitySyncRequestedRoutingKey, toBytes(m.message))
}

func PublishFactoryVelocitySyncRequested(factoryID string) error {
	return NewFactoryVelocitySyncRequestedMessage(factoryID).Publish()
}
