package entitlements

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SeatUsage separates purchased capacity from the maximum allowed purchase.
type SeatUsage struct {
	PerSeat         bool `json:"per_seat"`
	Purchased       *int `json:"purchased"`
	Occupied        int  `json:"occupied"`
	Available       *int `json:"available"`
	PlanMaximum     *int `json:"plan_maximum"`
	Purchasable     *int `json:"purchasable"`
	RenewalQuantity *int `json:"renewal_quantity"`
	MemberLimit     *int `json:"member_limit"`
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
	usage := &SeatUsage{Occupied: int(occupied)}
	if !models.BillingEnabled() {
		return usage, nil
	}
	plan, err := models.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	maximum, err := SeatMaximumForPlan(tx, orgID, &plan.Catalog)
	if err != nil {
		return nil, err
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
	renewal := max(usage.Occupied, 1)
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
