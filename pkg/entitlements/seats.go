package entitlements

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SeatUsage separates purchased capacity from the maximum allowed purchase.
type SeatUsage struct {
	PendingInvitations int  `json:"pending_invitations"`
	Invitable          *int `json:"invitable"`
	PerSeat            bool `json:"per_seat"`
	Purchased          *int `json:"purchased"`
	Occupied           int  `json:"occupied"`
	Available          *int `json:"available"`
	PlanMaximum        *int `json:"plan_maximum"`
	Purchasable        *int `json:"purchasable"`
	RenewalQuantity    *int `json:"renewal_quantity"`
	MemberLimit        *int `json:"member_limit"`
	// PaymentBlocked is true while a seat-based subscription has a failed
	// renewal payment or is paused. Seats stay locked at the current member count.
	PaymentBlocked bool `json:"payment_blocked"`
}

func (s SeatUsage) JoinLimit() *int {
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
	if !models.BillingEnabled() {
		return &SeatUsage{Occupied: int(occupied)}, nil
	}
	plan, err := models.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	return resolvePlanSeats(tx, orgID, plan, int(occupied))
}

// ResolveSeatsForPlan resolves seat usage with the plan the caller already
// loaded, so seat counts and prices come from a single plan read.
func ResolveSeatsForPlan(tx *gorm.DB, orgID uuid.UUID, plan *models.OrganizationBillingPlan) (*SeatUsage, error) {
	occupied, err := models.CountActiveHumanUsersByOrganization(tx, orgID.String())
	if err != nil {
		return nil, err
	}
	if !models.BillingEnabled() {
		return &SeatUsage{Occupied: int(occupied)}, nil
	}
	return resolvePlanSeats(tx, orgID, plan, int(occupied))
}

func resolvePlanSeats(tx *gorm.DB, orgID uuid.UUID, plan *models.OrganizationBillingPlan, occupied int) (*SeatUsage, error) {
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
	// A plan without paid seats needs repair and has no renewal count.
	renewal, err := RenewalSeats(plan, usage.Occupied)
	if err != nil && !errors.Is(err, ErrInvalidPaidSeats) {
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
func paymentBlocked(plan *models.OrganizationBillingPlan) bool {
	return models.PolarSubscriptionIsPaymentBlocked(plan.PolarSubscriptionStatus) && plan.PurchasedSeats != nil
}

func lockSeatsAtOccupied(usage *SeatUsage, plan *models.OrganizationBillingPlan, maximum *int) {
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
func SeatMaximumForPlan(tx *gorm.DB, orgID uuid.UUID, catalog *models.BillingPlan) (*int, error) {
	override, err := models.FindOrganizationSeatLimitOverride(tx, orgID)
	if err != nil {
		return nil, err
	}
	return SeatMaximum(catalog.SeatLimit, override), nil
}

// SeatMaximum applies a loaded organization override to a catalog limit.
// A nil override limit means unlimited, rather than falling back to the catalog.
func SeatMaximum(catalogLimit *int, override *models.OrganizationSeatLimitOverride) *int {
	if override != nil {
		return override.SeatLimit
	}
	return catalogLimit
}
