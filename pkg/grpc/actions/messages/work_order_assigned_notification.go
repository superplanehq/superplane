package messages

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

// TaskOwnerActorName names SuperPlane as the actor of an automated owner
// assignment.
const TaskOwnerActorName = "SuperPlane"

// PublishWorkOrderOwnersAssigned notifies the people who became owners of a
// task. actorUserID is nil when SuperPlane chose the owner. Call it after the
// transaction commits.
func PublishWorkOrderOwnersAssigned(order *models.FactoryWorkOrder, actorUserID *uuid.UUID, assigned []uuid.UUID, reason string) {
	if order == nil || len(assigned) == 0 {
		return
	}

	message := FactoryWorkOrderNotificationMessage{
		OrganizationID:   order.OrganizationID.String(),
		FactoryID:        order.FactoryID.String(),
		OrderID:          order.ID.String(),
		EventType:        factory.EventTypeOrderAssigneesUpdated,
		AssignedUserIDs:  make([]string, 0, len(assigned)),
		AssignmentReason: reason,
	}
	for _, userID := range assigned {
		message.AssignedUserIDs = append(message.AssignedUserIDs, userID.String())
	}
	if actorUserID != nil {
		message.ActorUserID = actorUserID.String()
	} else {
		message.ActorName = TaskOwnerActorName
	}

	if err := publishWorkOrderNotification(message); err != nil {
		log.WithError(err).Warnf("Failed to publish owner notification for order %s", order.ID)
	}
}
