package entitlements

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ErrInvalidPaidSeats is returned for a seat-based plan that stores no paid
// seat capacity. Polar always bills a seat-based subscription for at least
// one seat, so this state needs repair. It is never a renewal count.
var ErrInvalidPaidSeats = errors.New("seat-based subscription has no paid seats")

// IsSeatBased reports a plan that Polar renews per seat. A past-due, unpaid,
// or paused subscription can move the row to a catalog plan that is not per
// seat, but the row keeps its purchased seats, so the purchased seats also
// mark a seat-based plan.
func IsSeatBased(plan *models.OrganizationBillingPlan) bool {
	return plan.Catalog.PerSeat || plan.PurchasedSeats != nil
}

// RenewalSeats is the seat count that the next period of a seat-based plan
// must renew with: the occupied seats, at least 1, and never more than the
// paid seats. The count never adds seats.
//
// The count depends only on the paid seats and the members. It ignores the
// payment state, the admission limits, and the catalog maximum, so a fallback
// catalog after a payment failure does not change it.
//
// It returns nil for a plan that is not seat-based, and ErrInvalidPaidSeats
// when the plan stores no paid seats.
func RenewalSeats(plan *models.OrganizationBillingPlan, occupied int) (*int, error) {
	if !IsSeatBased(plan) {
		return nil, nil
	}
	if plan.PurchasedSeats == nil || *plan.PurchasedSeats < 1 {
		return nil, ErrInvalidPaidSeats
	}
	seats := min(max(occupied, 1), *plan.PurchasedSeats)
	return &seats, nil
}

// ResolveRenewalSeats counts the active members of the organization and
// returns RenewalSeats for the plan. It returns nil when billing is off.
func ResolveRenewalSeats(tx *gorm.DB, orgID uuid.UUID, plan *models.OrganizationBillingPlan) (*int, error) {
	if !models.BillingEnabled() {
		return nil, nil
	}
	occupied, err := models.CountActiveHumanUsersByOrganization(tx, orgID.String())
	if err != nil {
		return nil, err
	}
	return RenewalSeats(plan, int(occupied))
}
