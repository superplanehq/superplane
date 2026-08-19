package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateWorkOrderAssignees(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateWorkOrderAssigneesRequest,
) (*pb.UpdateWorkOrderAssigneesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	updatedBy, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to update work order assignees")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	tx := database.DB(ctx)
	assigneeIDs, err := parseAssigneeIDs(tx, orgID, req.GetAssigneeIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	var newlyAssignedIDs []string
	err = tx.Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		previousAssignees := order.AssigneeIDs()
		if err := order.UpdateAssignees(tx, assigneeIDs, updatedBy); err != nil {
			return err
		}

		newlyAssignedIDs = newAssigneeIDs(previousAssignees, assigneeIDs)
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factoryevents.EventTypeOrderAssigneesUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	if len(newlyAssignedIDs) > 0 {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:  orgID.String(),
			FactoryID:       factoryID.String(),
			OrderID:         orderID.String(),
			EventType:       factoryevents.EventTypeOrderAssigneesUpdated,
			ActorUserID:     updatedBy.String(),
			AssignedUserIDs: newlyAssignedIDs,
		}
		if err := notification.Publish(); err != nil {
			log.WithError(err).Warnf("Failed to publish work order notification for order %s", orderID)
		}
	}

	factory, err := models.FindFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	order, err := factory.FindWorkOrder(tx, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	return &pb.UpdateWorkOrderAssigneesResponse{
		Order: serialized,
	}, nil
}

// newAssigneeIDs returns the IDs present in next but not in previous —
// the users who should receive a "you were assigned" notification.
func newAssigneeIDs(previous, next []uuid.UUID) []string {
	previousSet := make(map[uuid.UUID]struct{}, len(previous))
	for _, id := range previous {
		previousSet[id] = struct{}{}
	}

	var added []string
	for _, id := range next {
		if _, ok := previousSet[id]; !ok {
			added = append(added, id.String())
		}
	}
	return added
}
