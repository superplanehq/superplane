package control

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
)

// PublishOrganizationCapacity is called after the transaction commits. Polling
// remains the recovery path if a notification cannot be delivered.
func PublishOrganizationCapacity(ctx context.Context, organizationID uuid.UUID) error {
	db := database.DB(ctx)
	fleetIDs, err := models.ListOrganizationRunnerFleetIDs(db, organizationID)
	if err != nil {
		return err
	}
	var failures []error
	for _, fleetID := range fleetIDs {
		if err := Publish(Notification{FleetID: fleetID.String()}); err != nil {
			failures = append(failures, err)
		}
	}
	factories, err := models.ListFactories(db, organizationID)
	if err != nil {
		return errors.Join(append(failures, err)...)
	}
	for _, factory := range factories {
		if err := messages.PublishFactoryWorkOrderUpdated(factory.ID.String(), "", "runner_capacity_updated"); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
