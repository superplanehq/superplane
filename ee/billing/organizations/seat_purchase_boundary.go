package organizations

import (
	"time"

	"github.com/superplanehq/superplane/ee/billing/polar"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
)

const (
	// seatPurchaseClockSkewMargin covers the difference between the clock of
	// this server and the clock that Polar uses to renew the subscription.
	seatPurchaseClockSkewMargin = 5 * time.Minute

	// seatPurchaseBoundaryWindow is the time before the end of the billing
	// period in which the server does not send a seat purchase to Polar.
	//
	// A purchase sends an absolute seat count. A scheduled seat reduction
	// applies at the renewal. If the renewal applies before Polar gets the
	// update, the absolute count is wrong: a 10 to 7 reduction with a
	// request for 1 more seat becomes a 7 to 11 purchase.
	//
	// The window is the Polar request margin plus the clock skew margin.
	// Polar can take that long to answer, so a request that starts outside
	// the window still lands before the renewal.
	seatPurchaseBoundaryWindow = polar.SeatPurchaseConfirmationWindow + seatPurchaseClockSkewMargin
)

const (
	messageSeatPurchaseRenewsSoon   = "The billing period renews soon. Buy seats after the renewal."
	messageSeatPurchasePeriodChange = "The billing period changed. Refresh and try again."
	messageSeatPurchaseBindingStale = "The subscription or billing period changed. Review the new price and confirm again."
)

// seatPurchaseClock reads the current time. Production code uses time.Now.
type seatPurchaseClock func() time.Time

// checkSeatPurchaseBoundary rejects a purchase that could reach Polar after
// the billing period ends.
func checkSeatPurchaseBoundary(periodEnd, now time.Time) error {
	if now.Add(seatPurchaseBoundaryWindow).Before(periodEnd) {
		return nil
	}
	return grpcerrors.FailedPrecondition(nil, messageSeatPurchaseRenewsSoon)
}

// checkSeatPurchasePeriodUnchanged rejects a purchase when the billing period
// is not the period that the caller saw. The caller must show a new preview.
func checkSeatPurchasePeriodUnchanged(seen, current *seatPurchaseState) error {
	if seen.periodStart.Equal(current.periodStart) && seen.periodEnd.Equal(current.periodEnd) {
		return nil
	}
	return grpcerrors.Conflict(nil, messageSeatPurchasePeriodChange)
}

// checkSeatPurchaseBinding rejects a purchase when the subscription or the
// billing period is not the one that the confirmed preview showed. A renewal
// that kept the seat count, or a replacement subscription with the same
// counts and dates, makes the old confirmation stale. The caller must show
// a new preview and get a new confirmation.
func checkSeatPurchaseBinding(binding *seatPurchaseBinding, current *seatPurchaseState) error {
	if binding.subscriptionID == current.subscriptionID &&
		binding.periodStart.Equal(current.periodStart) &&
		binding.periodEnd.Equal(current.periodEnd) {
		return nil
	}
	return grpcerrors.Conflict(nil, messageSeatPurchaseBindingStale)
}
