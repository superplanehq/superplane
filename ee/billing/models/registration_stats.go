package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	RegistrationStatsPageSize   = 5
	registrationCreditBatchSize = 200
)

type RegistrationStats struct {
	Items             []RegisteredOrganization `json:"items"`
	Total             int64                    `json:"total"`
	TrialGrantedCents *int64                   `json:"trial_granted_cents,omitempty"`
	TrialSpentCents   *int64                   `json:"trial_spent_cents,omitempty"`
	// PlanActivations counts activations in the period. It is not tied to the
	// registration cohort, and it is returned on every page.
	PlanActivations PlanActivationCounts `json:"plan_activations"`
	// PlanActivationTrackingStartedAt is the first moment that activations are tracked.
	PlanActivationTrackingStartedAt time.Time `json:"plan_activation_tracking_started_at"`
}

type RegisteredOrganization struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	OwnerEmail           string    `json:"owner_email"`
	Plan                 string    `json:"plan"`
	AvailableCreditCents int64     `json:"available_credit_cents"`
}

// DescribeRegistrationStats groups active organizations by registration time.
// Credit totals are lifetime values for that cohort, independent of pagination.
// They need every organization in the cohort, so only the first page (offset 0)
// calculates them, and report zero totals when no organization matches. Later
// pages load credit for their own rows only.
//
// Plan activation totals use activation time and do not depend on the cohort,
// the page, or the offset. They are calculated before any early return.
func DescribeRegistrationStats(tx *gorm.DB, start, end time.Time, offset int) (RegistrationStats, error) {
	report := RegistrationStats{Items: []RegisteredOrganization{}}
	trackingStart, err := PlanActivationTrackingStartedAt(tx)
	if err != nil {
		return report, err
	}
	report.PlanActivationTrackingStartedAt = trackingStart
	report.PlanActivations, err = CountPlanActivations(tx, start, end, trackingStart)
	if err != nil {
		return report, err
	}
	cohort := func() *gorm.DB {
		return tx.Model(&coremodels.Organization{}).Where("organizations.created_at >= ? AND organizations.created_at < ?", start, end)
	}
	var ids []uuid.UUID
	if err := cohort().Order("created_at DESC, id DESC").Pluck("id", &ids).Error; err != nil {
		return report, err
	}
	report.Total = int64(len(ids))
	if offset == 0 {
		if err := addTrialCreditTotals(tx, &report, ids); err != nil {
			return report, err
		}
	}
	if offset >= len(ids) {
		return report, nil
	}
	pageIDs := ids[offset:min(offset+RegistrationStatsPageSize, len(ids))]
	credits, err := hostedCreditSpendByOrganization(tx, pageIDs)
	if err != nil {
		return report, err
	}
	owners := tx.Model(&coremodels.User{}).
		Select("organization_id, STRING_AGG(DISTINCT email, ', ' ORDER BY email) AS owner_email").
		Where("organization_id IN ? AND is_owner = ? AND type = ?", pageIDs, true, coremodels.UserTypeHuman).Group("organization_id")
	err = cohort().Select("organizations.id, organizations.name, COALESCE(owners.owner_email, '') AS owner_email").
		Joins("LEFT JOIN (?) AS owners ON owners.organization_id = organizations.id", owners).
		Where("organizations.id IN ?", pageIDs).Order("organizations.created_at DESC, organizations.id DESC").Scan(&report.Items).Error
	if err != nil {
		return report, err
	}
	for i := range report.Items {
		item := &report.Items[i]
		item.AvailableCreditCents = billing.MicrosToCents(credits[item.ID].RemainingMicros)
		plan, err := FindOrganizationBillingPlan(tx, item.ID)
		if err != nil {
			return report, err
		}
		item.Plan = "None"
		if plan != nil {
			item.Plan = plan.Catalog.Name
		}
	}
	return report, nil
}

// addTrialCreditTotals sums trial credit in bounded batches of organizations.
func addTrialCreditTotals(tx *gorm.DB, report *RegistrationStats, ids []uuid.UUID) error {
	var granted, spent int64
	for start := 0; start < len(ids); start += registrationCreditBatchSize {
		credits, err := hostedCreditSpendWithTrialHistory(tx, ids[start:min(start+registrationCreditBatchSize, len(ids))])
		if err != nil {
			return err
		}
		for _, credit := range credits {
			granted += credit.TrialGrantedMicros
			spent += credit.TrialSpentMicros
		}
	}
	grantedCents, spentCents := billing.MicrosToCents(granted), billing.MicrosToCents(spent)
	report.TrialGrantedCents = &grantedCents
	report.TrialSpentCents = &spentCents
	return nil
}
