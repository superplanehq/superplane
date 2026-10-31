package messages

import (
	"github.com/google/uuid"
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
		if err := current.AttachLoginLease(db); err != nil {
			log.WithError(err).Warn("Failed to load ChatGPT catalog activity")
			continue
		}
		if current.Status == before.Status && current.ReservationGeneration == before.ReservationGeneration &&
			catalogTaskID(current) == catalogTaskID(&before) && current.CatalogCancellationRequested == before.CatalogCancellationRequested {
			continue
		}
		organizationID := current.OrganizationID.String()
		if published[organizationID] {
			continue
		}
		if err := PublishOrganizationCodexConnectionUpdated(db, current.OrganizationID); err != nil {
			log.WithError(err).Warn("Failed to publish ChatGPT connection change")
		}
		published[organizationID] = true
	}
}

func catalogTaskID(connection *models.FactoryCodexConnection) uuid.UUID {
	if connection.CatalogLease == nil {
		return uuid.Nil
	}
	return connection.CatalogLease.TaskID
}

// PublishOrganizationCodexConnectionUpdated carries no personal account details.
func PublishOrganizationCodexConnectionUpdated(db *gorm.DB, organizationID uuid.UUID) error {
	factories, err := models.ListFactories(db, organizationID)
	if err != nil {
		return err
	}
	for _, factory := range factories {
		if err := PublishFactoryCodexConnectionUpdated(factory.ID.String()); err != nil {
			return err
		}
	}
	return nil
}
