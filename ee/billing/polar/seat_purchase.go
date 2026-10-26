package polar

import (
	"context"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SeatPurchaseConfirmationWindow is how long a pending purchase waits for
// its own result before reconciliation reads Polar, and how long a purchase
// request can take to reach Polar. It is longer than the Polar request
// timeout. The end of the window proves nothing about the charge.
const SeatPurchaseConfirmationWindow = 5 * time.Minute

// SubscriptionReader is the part of the Polar API that purchase
// reconciliation uses.
type SubscriptionReader interface {
	GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionData, error)
}

// ReconcileSeatPurchase settles the in-flight seat purchase of an organization
// from the Polar subscription. It makes no write call to Polar, so it never
// buys seats. Capacity changes only through the applied snapshot.
//
// A pending purchase is left to its own request until the confirmation window
// passes. The snapshot settles the purchase only when it proves success, see
// billingmodels.SeatPurchase.SucceededBy. A snapshot that proves nothing leaves the
// purchase in flight: neither the seat count nor the elapsed time can show
// that Polar did not charge. The unresolved purchase monitor reports it, and
// an operator repairs it from Polar evidence.
func ReconcileSeatPurchase(ctx context.Context, db *gorm.DB, client SubscriptionReader, orgID uuid.UUID) error {
	purchase, err := billingmodels.FindInFlightSeatPurchase(db, orgID)
	if err != nil || purchase == nil || !seatPurchaseNeedsReconciliation(purchase, time.Now()) {
		return err
	}
	snapshot, err := client.GetSubscription(ctx, purchase.PolarSubscriptionID)
	if err != nil {
		return err
	}

	var alerts SeatCapacityAlerts
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return err
		}
		current, err := billingmodels.FindInFlightSeatPurchase(tx, orgID)
		if err != nil || current == nil || current.ID != purchase.ID {
			return err
		}
		if !current.SucceededBy(seatPurchaseEvidence(*snapshot)) {
			return nil
		}
		// SettleSeatPurchase, inside the subscription apply, records the
		// success, so the plan, the period, and the purchase match.
		alerts, err = StageOrganizationSubscription(ctx, tx, orgID, *snapshot)
		return err
	})
	if err != nil {
		return err
	}
	alerts.Emit()
	return nil
}

// RefreshSeatPurchaseState reads the current Polar subscription before a new
// purchase and applies the snapshot. A purchase sends an absolute seat count,
// so it must start from the seats Polar reports. A stale local count would
// send a count below the seats Polar has and remove seats that were added
// elsewhere. The snapshot repairs the capacity also when it proves nothing
// about an in-flight purchase. That purchase keeps its status.
func RefreshSeatPurchaseState(ctx context.Context, db *gorm.DB, client *Client, orgID uuid.UUID, subscriptionID string) error {
	snapshot, err := client.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}
	return ApplyOrganizationSubscription(ctx, db, orgID, *snapshot)
}

// SeatPurchaseResponseEvidence is the evidence in the snapshot that Polar
// returned for the update request of a purchase. Only the caller that sent
// that request can name it a response.
func SeatPurchaseResponseEvidence(snapshot SubscriptionData) billingmodels.SeatPurchaseEvidence {
	evidence := seatPurchaseEvidence(snapshot)
	evidence.Response = true
	return evidence
}

func seatPurchaseEvidence(snapshot SubscriptionData) billingmodels.SeatPurchaseEvidence {
	return billingmodels.SeatPurchaseEvidence{
		SubscriptionID: snapshot.ID,
		PeriodStart:    snapshot.CurrentPeriodStart.TimePtr(),
		Seats:          snapshot.Seats,
		ModifiedAt:     snapshot.ModifiedAt.TimePtr(),
	}
}

func seatPurchaseNeedsReconciliation(purchase *billingmodels.SeatPurchase, now time.Time) bool {
	return purchase.Status == billingmodels.SeatPurchaseStatusUnknown || now.Sub(purchase.CreatedAt) > SeatPurchaseConfirmationWindow
}
