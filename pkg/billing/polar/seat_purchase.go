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

// ReconcileSeatPurchase settles the in-flight seat purchase of an organization
// from the Polar subscription. It makes no write call to Polar, so it never
// buys seats. Capacity changes only through the applied snapshot.
//
// A pending purchase is left to its own request until the confirmation window
// passes. Polar can still be processing an unknown purchase inside the window,
// so only an applied or clearly rejected update settles it before then.
func ReconcileSeatPurchase(ctx context.Context, db *gorm.DB, client *Client, orgID uuid.UUID) error {
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

		status, settled := current.SettlementFor(snapshot.Seats, snapshot.ModifiedAt.TimePtr())
		if settled && status == models.SeatPurchaseStatusSucceeded {
			return ApplyOrganizationSubscription(ctx, tx, orgID, *snapshot)
		}
		if settled {
			return current.UpdateStatus(tx, status, models.SeatPurchaseErrorNotApplied)
		}
		if snapshot.Seats != nil && *snapshot.Seats == current.FromSeats && seatPurchaseWindowPassed(current, time.Now()) {
			return current.UpdateStatus(tx, models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorNotApplied)
		}
		return nil
	})
}

func seatPurchaseNeedsReconciliation(purchase *models.SeatPurchase, now time.Time) bool {
	return purchase.Status == models.SeatPurchaseStatusUnknown || seatPurchaseWindowPassed(purchase, now)
}

func seatPurchaseWindowPassed(purchase *models.SeatPurchase, now time.Time) bool {
	return now.Sub(purchase.CreatedAt) > SeatPurchaseConfirmationWindow
}
