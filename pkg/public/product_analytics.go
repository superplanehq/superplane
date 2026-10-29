package public

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
)

// Capture only the committed result; account deletion has already removed memberships.
func captureAccountDeletion(ctx context.Context, actorID, accountID uuid.UUID, deletedAt time.Time, organizations []models.Organization, source string) {
	organizationIDs := make([]string, 0, len(organizations))
	for _, organization := range organizations {
		organizationIDs = append(organizationIDs, organization.ID.String())
	}
	properties := productanalytics.Properties{
		"account_id": accountID.String(), "actor_account_id": actorID.String(),
		"deletion_source": source, "deleted_at": deletedAt.UTC().Format(time.RFC3339Nano),
		"deleted_organization_ids": organizationIDs,
	}
	productanalytics.Capture(productanalytics.Event{
		Name:       productanalytics.EventAccountDeleted,
		EventID:    productanalytics.StableEventID(productanalytics.EventAccountDeleted, accountID.String()),
		DistinctID: actorID.String(), ActorType: productanalytics.ActorUser,
		Source: productanalytics.SourceFromContext(ctx, productanalytics.SourceUI), Properties: properties,
	})
	for _, organization := range organizations {
		productanalytics.Capture(productanalytics.Event{
			Name:       productanalytics.EventOrganizationDeleted,
			EventID:    productanalytics.StableEventID(productanalytics.EventOrganizationDeleted, organization.ID.String()),
			DistinctID: actorID.String(), OrganizationID: organization.ID.String(),
			ActorType: productanalytics.ActorUser, Source: productanalytics.SourceFromContext(ctx, productanalytics.SourceUI),
			Properties: productanalytics.Properties{"account_id": accountID.String(), "actor_account_id": actorID.String(),
				"deletion_source": "account_deletion", "deleted_at": deletedAt.UTC().Format(time.RFC3339Nano)},
		})
	}
}
