package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RemainingHostedCreditMicros returns remaining hosted credit for each
// organization. An organization with no grants is 0. Spend follows
// allocateHostedCreditSpend, including expired welcome credit and negative
// adjustments.
func RemainingHostedCreditMicros(tx *gorm.DB, organizationIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	remaining := make(map[uuid.UUID]int64, len(organizationIDs))
	if len(organizationIDs) == 0 {
		return remaining, nil
	}
	for _, id := range organizationIDs {
		remaining[id] = 0
	}

	var grants []OrganizationLLMCreditGrant
	err := tx.Where("organization_id IN ?", organizationIDs).Find(&grants).Error
	if err != nil {
		return nil, err
	}

	billed, err := sumHostedBilledMicrosByOrganization(tx, organizationIDs, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	billedAtExpiry, err := billedMicrosAtExpiredGrantExpiries(tx, grants, now)
	if err != nil {
		return nil, err
	}

	grantsByOrganization := make(map[uuid.UUID][]OrganizationLLMCreditGrant)
	for _, grant := range grants {
		grantsByOrganization[grant.OrganizationID] = append(grantsByOrganization[grant.OrganizationID], grant)
	}

	for _, id := range organizationIDs {
		orgGrants := grantsByOrganization[id]
		if len(orgGrants) == 0 {
			continue
		}
		spend := allocateHostedCreditSpend(orgGrants, billed[id], billedAtExpiry[id], now)
		remaining[id] = spend.RemainingMicros
	}
	return remaining, nil
}

type organizationHostedSpend struct {
	OrganizationID uuid.UUID `gorm:"column:organization_id"`
	CostMicros     int64     `gorm:"column:cost_micros"`
}

func sumHostedBilledMicrosByOrganization(tx *gorm.DB, organizationIDs []uuid.UUID, atOrBefore *time.Time) (map[uuid.UUID]int64, error) {
	sums := map[uuid.UUID]int64{}
	if len(organizationIDs) == 0 {
		return sums, nil
	}

	query := tx.Model(&WorkspaceUsageEvent{}).
		Select("organization_id, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Where("organization_id IN ? AND funding_source = ?", organizationIDs, UsageFundingSourceHosted)
	if atOrBefore != nil {
		query = query.Where("occurred_at <= ?", *atOrBefore)
	}

	var rows []organizationHostedSpend
	err := query.Group("organization_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		sums[row.OrganizationID] = row.CostMicros
	}
	return sums, nil
}

func billedMicrosAtExpiredGrantExpiries(tx *gorm.DB, grants []OrganizationLLMCreditGrant, now time.Time) (map[uuid.UUID]map[int64]int64, error) {
	type expiryGroup struct {
		at   time.Time
		orgs map[uuid.UUID]struct{}
	}

	groups := map[int64]*expiryGroup{}
	order := make([]int64, 0)
	for _, grant := range grants {
		if !grant.IsExpired(now) || grant.ExpiresAt == nil {
			continue
		}
		at := grant.ExpiresAt.UTC()
		key := at.UnixNano()
		group, ok := groups[key]
		if !ok {
			group = &expiryGroup{at: at, orgs: map[uuid.UUID]struct{}{}}
			groups[key] = group
			order = append(order, key)
		}
		group.orgs[grant.OrganizationID] = struct{}{}
	}

	billedAtExpiry := map[uuid.UUID]map[int64]int64{}
	for _, key := range order {
		group := groups[key]
		orgIDs := make([]uuid.UUID, 0, len(group.orgs))
		for id := range group.orgs {
			orgIDs = append(orgIDs, id)
		}
		sums, err := sumHostedBilledMicrosByOrganization(tx, orgIDs, &group.at)
		if err != nil {
			return nil, err
		}
		for _, id := range orgIDs {
			if billedAtExpiry[id] == nil {
				billedAtExpiry[id] = map[int64]int64{}
			}
			billedAtExpiry[id][key] = sums[id]
		}
	}
	return billedAtExpiry, nil
}
