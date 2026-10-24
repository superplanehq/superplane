package polar

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SeatPurchaseConfirmationWindow is how long a pending purchase can wait for
// its own result, and how long Polar can take to apply an unknown purchase.
// It is longer than the Polar request timeout.
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
// passes. Polar can still be processing an unknown purchase inside the window,
// so only an applied or clearly rejected update settles it before then.
//
// A snapshot that proves the outcome is applied, so the plan, the period, and
// the purchase match what Polar reports. A retry after a renewal then works
// from the refreshed plan.
func ReconcileSeatPurchase(ctx context.Context, db *gorm.DB, client SubscriptionReader, orgID uuid.UUID) error {
	purchase, err := models.FindInFlightSeatPurchase(db, orgID)
	if err != nil || purchase == nil || !seatPurchaseNeedsReconciliation(purchase, time.Now()) {
		return err
	}
	snapshot, err := client.GetSubscription(ctx, purchase.PolarSubscriptionID)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return err
		}
		current, err := models.FindInFlightSeatPurchase(tx, orgID)
		if err != nil || current == nil || current.ID != purchase.ID {
			return err
		}

		// A snapshot that proves the outcome is applied. SettleSeatPurchase,
		// inside ApplyOrganizationSubscription, records the proved status.
		if _, settled := current.SettlementFor(snapshot.Seats, snapshot.ModifiedAt.TimePtr()); settled {
			return ApplyOrganizationSubscription(ctx, tx, orgID, *snapshot)
		}
		return failExpiredSeatPurchase(tx, current, snapshot.Seats, time.Now())
	})
}

// RefreshSeatPurchaseState reads the current Polar subscription before a new
// purchase and applies the snapshot. A purchase sends an absolute seat count,
// so it must start from the seats Polar reports. A stale local count would
// send a count below the seats Polar has and remove seats that were added
// elsewhere. Applying the snapshot also settles the in-flight purchase, and
// an expired one fails so it cannot block new purchases forever.
func RefreshSeatPurchaseState(ctx context.Context, db *gorm.DB, client *Client, orgID uuid.UUID, subscriptionID string) error {
	snapshot, err := client.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return err
		}
		if err := ApplyOrganizationSubscription(ctx, tx, orgID, *snapshot); err != nil {
			return err
		}
		purchase, err := models.FindInFlightSeatPurchase(tx, orgID)
		if err != nil || purchase == nil {
			return err
		}
		return failExpiredSeatPurchase(tx, purchase, snapshot.Seats, time.Now())
	})
}

// failExpiredSeatPurchase fails an in-flight purchase whose confirmation
// window passed while Polar still reports fewer seats than the target. The
// purchase can no longer reach its target, so it must not block new
// purchases. Counts between the original and the target clear here too.
func failExpiredSeatPurchase(tx *gorm.DB, purchase *models.SeatPurchase, seats *int, now time.Time) error {
	if seats == nil || *seats >= purchase.ToSeats || !seatPurchaseWindowPassed(purchase, now) {
		return nil
	}
	return purchase.UpdateStatus(tx, models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorNotApplied)
}

func seatPurchaseNeedsReconciliation(purchase *models.SeatPurchase, now time.Time) bool {
	return purchase.Status == models.SeatPurchaseStatusUnknown || seatPurchaseWindowPassed(purchase, now)
}

func seatPurchaseWindowPassed(purchase *models.SeatPurchase, now time.Time) bool {
	return now.Sub(purchase.CreatedAt) > SeatPurchaseConfirmationWindow
}
