package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const RegistrationStatsPageSize = 5

type RegistrationStats struct {
	Items             []RegisteredOrganization `json:"items"`
	Total             int64                    `json:"total"`
	TrialGrantedCents int64                    `json:"trial_granted_cents"`
	TrialSpentCents   int64                    `json:"trial_spent_cents"`
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
func DescribeRegistrationStats(tx *gorm.DB, start, end time.Time, offset int) (RegistrationStats, error) {
	report := RegistrationStats{Items: []RegisteredOrganization{}}
	cohort := func() *gorm.DB {
		return tx.Model(&coremodels.Organization{}).Where("organizations.created_at >= ? AND organizations.created_at < ?", start, end)
	}
	var ids []uuid.UUID
	if err := cohort().Order("created_at DESC, id DESC").Pluck("id", &ids).Error; err != nil {
		return report, err
	}
	report.Total = int64(len(ids))
	if len(ids) == 0 {
		return report, nil
	}
	credits, err := hostedCreditSpendByOrganization(tx, ids)
	if err != nil {
		return report, err
	}
	var granted, spent int64
	for _, credit := range credits {
		granted += credit.TrialGrantedMicros
		spent += credit.TrialSpentMicros
	}
	report.TrialGrantedCents = billing.MicrosToCents(granted)
	report.TrialSpentCents = billing.MicrosToCents(spent)
	if offset >= len(ids) {
		return report, nil
	}
	pageIDs := ids[offset:min(offset+RegistrationStatsPageSize, len(ids))]
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
