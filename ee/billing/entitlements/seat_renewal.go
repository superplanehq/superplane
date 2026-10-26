package entitlements

import (
	"errors"

	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
)

// ErrInvalidPaidSeats is returned for a seat-based plan that stores no paid
// seat capacity. Polar always bills a seat-based subscription for at least
// one seat, so this state needs repair. It is never a renewal count.
var ErrInvalidPaidSeats = errors.New("seat-based subscription has no paid seats")

// isSeatBased reports a plan that Polar renews per seat. A past-due, unpaid,
// or paused subscription can move the row to a catalog plan that is not per
// seat, but the row keeps its purchased seats, so the purchased seats also
// mark a seat-based plan.
func isSeatBased(plan *billingmodels.OrganizationBillingPlan) bool {
	return plan.Catalog.PerSeat || plan.PurchasedSeats != nil
}

// RenewalSeats is the renewal estimate that the billing summary shows: the
// occupied seats, at least 1, and never more than the paid seats. It keeps
// the formula of the superseded D14 decision until the seat removal
// presentation replaces it. Renewal itself keeps the paid seats: no flow
// sends this count to Polar.
//
// The count depends only on the paid seats and the members. It ignores the
// payment state, the admission limits, and the catalog maximum, so a fallback
// catalog after a payment failure does not change it.
//
// It returns nil for a plan that is not seat-based, and ErrInvalidPaidSeats
// when the plan stores no paid seats.
func RenewalSeats(plan *billingmodels.OrganizationBillingPlan, occupied int) (*int, error) {
	if !isSeatBased(plan) {
		return nil, nil
	}
	if plan.PurchasedSeats == nil || *plan.PurchasedSeats < 1 {
		return nil, ErrInvalidPaidSeats
	}
	seats := min(max(occupied, 1), *plan.PurchasedSeats)
	return &seats, nil
}
