package models

import (
	"strings"
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

const hostedSpendAtExpiryBatchSize = 100

type hostedCreditExpiry struct {
	OrganizationID uuid.UUID
	At             time.Time
	Nanos          int64
}

type expirySpendKey struct {
	organizationID uuid.UUID
	nanos          int64
}

type hostedSpendAtExpiryRow struct {
	OrganizationID uuid.UUID `gorm:"column:organization_id"`
	ExpiryNanos    int64     `gorm:"column:expiry_nanos"`
	CostMicros     int64     `gorm:"column:cost_micros"`
}

func billedMicrosAtExpiredGrantExpiries(tx *gorm.DB, grants []OrganizationLLMCreditGrant, now time.Time) (map[uuid.UUID]map[int64]int64, error) {
	pairs := expiredGrantSpendPairs(grants, now)
	billedAtExpiry := map[uuid.UUID]map[int64]int64{}
	for start := 0; start < len(pairs); start += hostedSpendAtExpiryBatchSize {
		end := start + hostedSpendAtExpiryBatchSize
		if end > len(pairs) {
			end = len(pairs)
		}
		batch := pairs[start:end]
		sums, err := sumHostedBilledMicrosAtExpiries(tx, batch)
		if err != nil {
			return nil, err
		}
		for _, pair := range batch {
			if billedAtExpiry[pair.OrganizationID] == nil {
				billedAtExpiry[pair.OrganizationID] = map[int64]int64{}
			}
			billedAtExpiry[pair.OrganizationID][pair.Nanos] = sums[expirySpendKey{
				organizationID: pair.OrganizationID,
				nanos:          pair.Nanos,
			}]
		}
	}
	return billedAtExpiry, nil
}

func expiredGrantSpendPairs(grants []OrganizationLLMCreditGrant, now time.Time) []hostedCreditExpiry {
	seen := map[expirySpendKey]struct{}{}
	pairs := make([]hostedCreditExpiry, 0)
	for _, grant := range grants {
		if !grant.IsExpired(now) || grant.ExpiresAt == nil {
			continue
		}
		at := grant.ExpiresAt.UTC()
		key := expirySpendKey{organizationID: grant.OrganizationID, nanos: at.UnixNano()}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		pairs = append(pairs, hostedCreditExpiry{
			OrganizationID: grant.OrganizationID,
			At:             at,
			Nanos:          key.nanos,
		})
	}
	return pairs
}

func sumHostedBilledMicrosAtExpiries(tx *gorm.DB, pairs []hostedCreditExpiry) (map[expirySpendKey]int64, error) {
	sums := map[expirySpendKey]int64{}
	if len(pairs) == 0 {
		return sums, nil
	}

	var query strings.Builder
	args := make([]any, 0, len(pairs)*3+1)
	query.WriteString(`
SELECT
	pairs.organization_id,
	pairs.expiry_nanos,
	COALESCE(SUM(events.cost_micros), 0) AS cost_micros
FROM (VALUES `)
	for i, pair := range pairs {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("(CAST(? AS uuid), CAST(? AS bigint), CAST(? AS timestamptz))")
		args = append(args, pair.OrganizationID, pair.Nanos, pair.At)
	}
	query.WriteString(`
) AS pairs(organization_id, expiry_nanos, expires_at)
LEFT JOIN workspace_usage_events AS events
	ON events.organization_id = pairs.organization_id
	AND events.funding_source = ?
	AND events.occurred_at <= pairs.expires_at
GROUP BY pairs.organization_id, pairs.expiry_nanos`)
	args = append(args, UsageFundingSourceHosted)

	var rows []hostedSpendAtExpiryRow
	err := tx.Raw(query.String(), args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		sums[expirySpendKey{organizationID: row.OrganizationID, nanos: row.ExpiryNanos}] = row.CostMicros
	}
	return sums, nil
}
