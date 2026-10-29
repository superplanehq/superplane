package polar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/productanalytics"
)

// ObservePaymentMethods reuses a completed provider read. Persistence and
// analytics are best effort and never change the caller's billing response.
func ObservePaymentMethods(ctx context.Context, organizationID uuid.UUID, snapshot PaymentMethodSnapshot) {
	if !snapshot.Complete {
		return
	}
	observation := billingmodels.PaymentMethodObservation{
		OrganizationID: organizationID, ReadStartedAt: snapshot.ReadStartedAt, ObservedAt: snapshot.ObservedAt,
		PresentHashes: make(map[string]bool),
	}
	for _, method := range snapshot.Methods {
		digest := sha256.Sum256([]byte(organizationID.String() + ":" + method.ID))
		hash := hex.EncodeToString(digest[:])
		observation.PresentHashes[hash] = method.Expired
		if method.Usable {
			observation.UsableHashes = append(observation.UsableHashes, hash)
		}
	}
	loss, err := billingmodels.ObserveOrganizationPaymentMethods(database.DB(ctx), observation)
	if err != nil {
		log.WithError(err).WithField("organization_id", organizationID).Warn("Failed to persist payment method observation")
		return
	}
	if loss == nil {
		return
	}
	organization := organizationID.String()
	productanalytics.Capture(productanalytics.Event{
		Name:           productanalytics.EventPaymentMethodRemoved,
		EventID:        productanalytics.StableEventID(productanalytics.EventPaymentMethodRemoved, organization, loss.OccurrenceID.String()),
		DistinctID:     productanalytics.SyntheticDistinctID(organization, productanalytics.ActorSystem, "billing"),
		OrganizationID: organization, ActorType: productanalytics.ActorSystem, Source: productanalytics.SourcePolarAPI,
		Properties: productanalytics.Properties{"provider": "polar", "detection_method": "availability_check", "reason": loss.Reason,
			"observed_at": loss.ObservedAt.UTC().Format(time.RFC3339Nano), "transition_id": loss.OccurrenceID.String()},
	})
}
