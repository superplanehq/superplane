package development

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var localBillingNonWelcomeGrantKinds = []string{
	billingmodels.LLMCreditGrantKindAdmin,
	billingmodels.LLMCreditGrantKindIncluded,
	billingmodels.LLMCreditGrantKindIncludedAdjustment,
	billingmodels.LLMCreditGrantKindTopup,
	billingmodels.LLMCreditGrantKindTopupRefund,
	billingmodels.LLMCreditGrantKindTrialAdjustment,
	billingmodels.LLMCreditGrantKindTopupAdjustment,
	billingmodels.LLMCreditGrantKindAdminAdjustment,
}

// ResetLocalAfterOnboarding wipes factory runtime data and restores trial
// billing with a full welcome credit window. It keeps organizations,
// workspaces, apps, lines, intakes, and integrations.
func ResetLocalAfterOnboarding(tx *gorm.DB) error {
	var factories []models.Factory
	if err := tx.Find(&factories).Error; err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	for i := range factories {
		if err := models.NewFactoryRuntimeCleaner(tx, &factories[i]).Run(); err != nil {
			return fmt.Errorf("reset workspace %s: %w", factories[i].ID, err)
		}
	}

	var orgIDs []uuid.UUID
	if err := tx.Model(&models.Organization{}).Pluck("id", &orgIDs).Error; err != nil {
		return fmt.Errorf("list organizations: %w", err)
	}

	for _, orgID := range orgIDs {
		if err := ResetOrganizationBillingTrial(tx, orgID); err != nil {
			return fmt.Errorf("reset billing for organization %s: %w", orgID, err)
		}
	}

	return nil
}

// ResetOrganizationBillingTrial puts one organization on a 14-day system
// trial, drops extra credit grants, refreshes welcome credit expiry, and
// deletes that organization's spend ledger.
func ResetOrganizationBillingTrial(tx *gorm.DB, orgID uuid.UUID) error {
	now := time.Now()
	catalog, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanTrial)
	if err != nil {
		return err
	}
	trialEnd := now.AddDate(0, 0, catalog.TrialLengthDays)

	result := tx.Exec(`
		UPDATE organization_billing_plans
		SET
			plan = ?,
 plan_id = ?,
			plan_source = ?,
			polar_subscription_id = NULL,
			polar_subscription_status = '',
			cancel_at_period_end = FALSE,
			past_due_since = NULL,
			polar_modified_at = NULL,
			current_period_start = NULL,
			current_period_end = NULL,
			trial_started_at = ?,
			trial_ends_at = ?,
			updated_at = ?
		WHERE organization_id = ?
	`, billingmodels.BillingPlanTrial, catalog.ID, billingmodels.BillingPlanSourceSystem, now, trialEnd, now, orgID)
	if result.Error != nil {
		return fmt.Errorf("reset billing plan: %w", result.Error)
	}

	result = tx.Exec(`
		UPDATE organization_llm_settings
		SET polar_customer_id = NULL, updated_at = ?
		WHERE organization_id = ? AND polar_customer_id IS NOT NULL
	`, now, orgID)
	if result.Error != nil {
		return fmt.Errorf("clear billing customer: %w", result.Error)
	}

	if err := tx.Where("organization_id = ? AND kind IN ?", orgID, localBillingNonWelcomeGrantKinds).
		Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error; err != nil {
		return fmt.Errorf("delete extra credit grants: %w", err)
	}

	if err := tx.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", orgID, billingmodels.LLMCreditGrantKindWelcome).
		Update("expires_at", trialEnd).Error; err != nil {
		return fmt.Errorf("refresh welcome credit: %w", err)
	}

	if err := tx.Where("organization_id = ?", orgID).
		Delete(&models.WorkspaceUsageEvent{}).Error; err != nil {
		return fmt.Errorf("delete usage events: %w", err)
	}

	return nil
}
