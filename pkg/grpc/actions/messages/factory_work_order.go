package messages

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
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

// PublishFactoryCodexConnectionUpdated contains no personal account data.
func PublishFactoryCodexConnectionUpdated(factoryID string) error {
	return PublishFactoryWorkOrderUpdated(factoryID, "", "codex_connection")
}

// PublishCodexConnectionChanges runs after commit. Login completion can change
// eligibility and catalogs even when it does not change a task's state.
func PublishCodexConnectionChanges(db *gorm.DB, previous []models.FactoryCodexConnection) {
	published := map[string]bool{}
	for _, before := range previous {
		current, err := models.FindFactoryCodexConnectionByID(db, before.ID)
		if err != nil {
			log.WithError(err).Warn("Failed to load completed ChatGPT sign-in")
			continue
		}
		if current.Status == before.Status && current.ReservationGeneration == before.ReservationGeneration {
			continue
		}
		factoryID := current.FactoryID.String()
		if published[factoryID] {
			continue
		}
		if err := PublishFactoryCodexConnectionUpdated(factoryID); err != nil {
			log.WithError(err).Warn("Failed to publish ChatGPT connection change")
		}
		published[factoryID] = true
	}
}
