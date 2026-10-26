package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
)

// OrganizationDeletionClearance captures the billing snapshot verified before
// deletion. Validate checks the same snapshot under the organization lock.
type OrganizationDeletionClearance struct {
	organizationID   uuid.UUID
	hasBillingPlan   bool
	subscriptionID   string
	polarModifiedAt  *time.Time
	planUpdatedAt    time.Time
	providerVerified bool
}

func NewLocalDeletionClearance(orgID uuid.UUID, plan *OrganizationBillingPlan) (*OrganizationDeletionClearance, error) {
	if plan.MayBeBillable(time.Now()) {
		return nil, billing.ErrOrganizationSubscriptionLive
	}
	return newDeletionClearance(orgID, plan, false), nil
}

func NewVerifiedDeletionClearance(plan *OrganizationBillingPlan) *OrganizationDeletionClearance {
	if plan == nil {
		return nil
	}
	return newDeletionClearance(plan.OrganizationID, plan, true)
}

func newDeletionClearance(orgID uuid.UUID, plan *OrganizationBillingPlan, providerVerified bool) *OrganizationDeletionClearance {
	clearance := &OrganizationDeletionClearance{organizationID: orgID, providerVerified: providerVerified}
	if plan == nil {
		return clearance
	}
	clearance.hasBillingPlan = true
	clearance.subscriptionID = storedPolarSubscriptionID(plan)
	clearance.polarModifiedAt = plan.PolarModifiedAt
	clearance.planUpdatedAt = plan.UpdatedAt
	return clearance
}

// Validate locks the billing row inside the caller-owned deletion transaction.
// Checkout claims use the organization lock, so unresolved intents cannot race
// past this check. All provider requests finish before this method is called.
func (c *OrganizationDeletionClearance) Validate(tx *gorm.DB, orgID uuid.UUID) error {
	if c == nil || c.organizationID != orgID {
		return billing.ErrOrganizationSubscriptionUnverified
	}
	locked, err := lockOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if locked.MayBeBillable(time.Now()) && !c.providerVerified {
		return billing.ErrOrganizationSubscriptionLive
	}
	if !c.matches(locked) {
		return billing.ErrBillingPlanChanged
	}
	checkout, err := FindUnresolvedOrganizationBillingCheckout(tx, orgID)
	if err != nil {
		return err
	}
	if checkout != nil {
		return billing.ErrOrganizationCheckoutPending
	}
	return nil
}

func (c *OrganizationDeletionClearance) matches(plan *OrganizationBillingPlan) bool {
	if plan == nil {
		return !c.hasBillingPlan
	}
	return c.hasBillingPlan && c.subscriptionID == storedPolarSubscriptionID(plan) &&
		sameOptionalTime(c.polarModifiedAt, plan.PolarModifiedAt) && c.planUpdatedAt.Equal(plan.UpdatedAt)
}

func storedPolarSubscriptionID(plan *OrganizationBillingPlan) string {
	if plan == nil || plan.PolarSubscriptionID == nil {
		return ""
	}
	return strings.TrimSpace(*plan.PolarSubscriptionID)
}

func sameOptionalTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

var (
	ErrOrganizationSubscriptionLive       = billing.ErrOrganizationSubscriptionLive
	ErrOrganizationSubscriptionUnverified = billing.ErrOrganizationSubscriptionUnverified
	ErrOrganizationCheckoutPending        = billing.ErrOrganizationCheckoutPending
)
