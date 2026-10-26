package entitlements

import (
	"errors"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SeatUsage separates purchased capacity from the maximum allowed purchase.
type SeatUsage = billing.SeatUsage

func JoinLimit(s *SeatUsage) *int {
	if s.PaymentBlocked {
		occupied := s.Occupied
		return &occupied
	}
	if s.PerSeat {
		return s.Purchased
	}
	return s.MemberLimit
}

func ResolveSeats(tx *gorm.DB, orgID uuid.UUID) (*SeatUsage, error) {
	occupied, err := models.CountActiveHumanUsersByOrganization(tx, orgID.String())
	if err != nil {
		return nil, err
	}
	if !billingmodels.BillingEnabled() {
		return &SeatUsage{Occupied: int(occupied)}, nil
	}
	plan, err := billingmodels.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	return resolvePlanSeats(tx, orgID, plan, int(occupied))
}

// ResolveSeatsForPlan resolves seat usage with the plan the caller already
// loaded, so seat counts and prices come from a single plan read.
func ResolveSeatsForPlan(tx *gorm.DB, orgID uuid.UUID, plan *billingmodels.OrganizationBillingPlan) (*SeatUsage, error) {
	occupied, err := models.CountActiveHumanUsersByOrganization(tx, orgID.String())
	if err != nil {
		return nil, err
	}
	if !billingmodels.BillingEnabled() {
		return &SeatUsage{Occupied: int(occupied)}, nil
	}
	return resolvePlanSeats(tx, orgID, plan, int(occupied))
}

func resolvePlanSeats(tx *gorm.DB, orgID uuid.UUID, plan *billingmodels.OrganizationBillingPlan, occupied int) (*SeatUsage, error) {
	usage := &SeatUsage{Occupied: occupied}
	pending, err := models.CountPendingOrganizationInvitations(tx, orgID)
	if err != nil {
		return nil, err
	}
	usage.PendingInvitations = int(pending)
	maximum, err := SeatMaximumForPlan(tx, orgID, &plan.Catalog)
	if err != nil {
		return nil, err
	}
	// A past_due snapshot moves the plan row to None or Trial after the grace
	// period, and these are not per-seat. The lock must not depend on the catalog.
	if paymentBlocked(plan) {
		lockSeatsAtOccupied(usage, plan, maximum)
		// The renewal estimate ignores the payment state, so it does not
		// change while the payment fails.
		renewal, err := paidRenewalSeats(plan, usage.Occupied)
		if err != nil {
			return nil, err
		}
		usage.RenewalQuantity = renewal
		return usage, nil
	}
	if !plan.Catalog.PerSeat {
		usage.MemberLimit = maximum
		return usage, nil
	}

	purchased := 0
	if plan.PurchasedSeats != nil {
		purchased = *plan.PurchasedSeats
	}
	available := max(purchased-usage.Occupied, 0)
	invitable := max(purchased-usage.Occupied-usage.PendingInvitations, 0)
	usage.Invitable = &invitable
	usage.PerSeat = true
	usage.Purchased = &purchased
	usage.Available = &available
	usage.PlanMaximum = maximum
	renewal, err := paidRenewalSeats(plan, usage.Occupied)
	if err != nil {
		return nil, err
	}
	usage.RenewalQuantity = renewal
	if maximum != nil {
		purchasable := max(*maximum-purchased, 0)
		usage.Purchasable = &purchasable
	}
	return usage, nil
}

// paymentBlocked reports a seat-based subscription whose renewal payment failed
// or that the customer paused. Stored purchased seats tell a seat-based
// subscription from one without seats.
func paymentBlocked(plan *billingmodels.OrganizationBillingPlan) bool {
	return billingmodels.PolarSubscriptionIsPaymentBlocked(plan.PolarSubscriptionStatus) && plan.PurchasedSeats != nil
}

// paidRenewalSeats returns RenewalSeats for a seat summary. A plan without
// paid seats needs repair and has no renewal count, so that state is nil and
// not an error.
func paidRenewalSeats(plan *billingmodels.OrganizationBillingPlan, occupied int) (*int, error) {
	renewal, err := RenewalSeats(plan, occupied)
	if errors.Is(err, ErrInvalidPaidSeats) {
		return nil, nil
	}
	return renewal, err
}

func lockSeatsAtOccupied(usage *SeatUsage, plan *billingmodels.OrganizationBillingPlan, maximum *int) {
	none := 0
	usage.PaymentBlocked = true
	usage.MemberLimit = maximum
	usage.Invitable = &none
	usage.Purchasable = &none
	purchased := *plan.PurchasedSeats
	usage.Purchased = &purchased
}

// SeatMaximumForPlan resolves the purchase maximum for a target catalog plan.
// An organization override replaces the catalog maximum, including unlimited.
func SeatMaximumForPlan(tx *gorm.DB, orgID uuid.UUID, catalog *billingmodels.BillingPlan) (*int, error) {
	override, err := billingmodels.FindOrganizationSeatLimitOverride(tx, orgID)
	if err != nil {
		return nil, err
	}
	return SeatMaximum(catalog.SeatLimit, override), nil
}

// SeatMaximum applies a loaded organization override to a catalog limit.
// A nil override limit means unlimited, rather than falling back to the catalog.
func SeatMaximum(catalogLimit *int, override *billingmodels.OrganizationSeatLimitOverride) *int {
	if override != nil {
		return override.SeatLimit
	}
	return catalogLimit
}
