package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
)

// RequireCodexSetupAdmission checks new sign-in and catalog allocations.
// Setup belongs to the organization and does not consume workspace budgets.
func RequireCodexSetupAdmission(tx *gorm.DB, organizationID uuid.UUID) error {
	if err := requireCodexSubscriptionEligible(tx, organizationID); err != nil {
		return err
	}
	if !billing.Enabled() {
		return nil
	}
	if err := billing.AssertOrganizationPaymentMethod(tx, organizationID); err != nil {
		return err
	}
	return billing.AssertHostedRunAllowed(tx, organizationID, nil)
}
