package entitlements

import (
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
}

func (s SeatUsage) JoinLimit() *int {
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
	override, err := models.FindOrganizationSeatLimitOverride(tx, orgID)
	if err != nil {
		return nil, err
	}
	maximum := seatMaximum(plan.Catalog.SeatLimit, override)
	if !plan.Catalog.PerSeat {
		usage.MemberLimit = maximum
		return usage, nil
	}

	purchased := 0
	if plan.PurchasedSeats != nil {
		purchased = *plan.PurchasedSeats
	}
	available := max(purchased-usage.Occupied, 0)
	renewal := max(usage.Occupied, 1)
	invitable := max(purchased-usage.Occupied-usage.PendingInvitations, 0)
	usage.Invitable = &invitable
	usage.PerSeat = true
	usage.Purchased = &purchased
	usage.Available = &available
	usage.PlanMaximum = maximum
	usage.RenewalQuantity = &renewal
	if maximum != nil {
		purchasable := max(*maximum-purchased, 0)
		usage.Purchasable = &purchasable
	}
	return usage, nil
}

func seatMaximum(catalogLimit *int, override *models.OrganizationSeatLimitOverride) *int {
	if override != nil {
		return override.SeatLimit
	}
	return catalogLimit
}
