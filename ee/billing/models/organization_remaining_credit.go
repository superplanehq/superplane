package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// RemainingHostedCreditMicros returns remaining hosted credit for each
// organization. An organization with no grants is 0. Spend follows
// allocateHostedCreditSpend, including expired welcome credit and negative
// adjustments.
func RemainingHostedCreditMicros(tx *gorm.DB, organizationIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	credits, err := hostedCreditSpendByOrganization(tx, organizationIDs)
	if err != nil {
		return nil, err
	}
	remaining := make(map[uuid.UUID]int64, len(credits))
	for id, credit := range credits {
		remaining[id] = credit.RemainingMicros
	}
	return remaining, nil
}

func hostedCreditSpendByOrganization(tx *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]hostedCreditSpend, error) {
	return loadHostedCreditSpend(tx, ids, false)
}

// hostedCreditSpendWithTrialHistory also loads the billed amount at each grant
// creation time, which lifetime trial spend needs.
func hostedCreditSpendWithTrialHistory(tx *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]hostedCreditSpend, error) {
	return loadHostedCreditSpend(tx, ids, true)
}

func loadHostedCreditSpend(tx *gorm.DB, ids []uuid.UUID, withTrialHistory bool) (map[uuid.UUID]hostedCreditSpend, error) {
	credits := make(map[uuid.UUID]hostedCreditSpend, len(ids))
	if len(ids) == 0 {
		return credits, nil
	}
	var grants []OrganizationLLMCreditGrant
	if err := tx.Where("organization_id IN ?", ids).Find(&grants).Error; err != nil {
		return nil, err
	}
	billed, err := sumHostedBilledMicrosByOrganization(tx, ids, nil)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	pairs := expiredGrantSpendPairs(grants, now)
	if withTrialHistory {
		pairs = append(pairs, trialGrantCreationPairs(grants, pairs)...)
	}
	atExpiry, err := billedMicrosAtSpendPairs(tx, pairs)
	if err != nil {
		return nil, err
	}
	byOrganization := map[uuid.UUID][]OrganizationLLMCreditGrant{}
	for _, grant := range grants {
		byOrganization[grant.OrganizationID] = append(byOrganization[grant.OrganizationID], grant)
	}
	for _, id := range ids {
		credits[id] = allocateHostedCreditSpend(byOrganization[id], billed[id], atExpiry[id], now)
	}
	return credits, nil
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

	query := tx.Model(&coremodels.WorkspaceUsageEvent{}).
		Select("organization_id, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Where("organization_id IN ? AND funding_source = ?", organizationIDs, coremodels.UsageFundingSourceHosted)
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

func billedMicrosAtSpendPairs(tx *gorm.DB, pairs []hostedCreditExpiry) (map[uuid.UUID]map[int64]int64, error) {
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

// trialGrantCreationPairs returns the creation time of each grant of an
// organization that has trial credit, except times that are already in known.
// Lifetime trial spend splits usage at these times. Usage at the creation time
// itself belongs to the new grant, so the billed amount covers usage strictly
// before it: the query time is one microsecond (the database precision) earlier.
func trialGrantCreationPairs(grants []OrganizationLLMCreditGrant, known []hostedCreditExpiry) []hostedCreditExpiry {
	hasTrial := map[uuid.UUID]bool{}
	for _, grant := range grants {
		if grantCreditBucket(grant.Kind) == creditBucketWelcome {
			hasTrial[grant.OrganizationID] = true
		}
	}
	seen := map[expirySpendKey]struct{}{}
	for _, pair := range known {
		seen[expirySpendKey{organizationID: pair.OrganizationID, nanos: pair.Nanos}] = struct{}{}
	}
	pairs := make([]hostedCreditExpiry, 0)
	for _, grant := range grants {
		if !hasTrial[grant.OrganizationID] {
			continue
		}
		at := grant.CreatedAt.UTC()
		key := expirySpendKey{organizationID: grant.OrganizationID, nanos: at.UnixNano()}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		pairs = append(pairs, hostedCreditExpiry{OrganizationID: grant.OrganizationID, At: at.Add(-time.Microsecond), Nanos: key.nanos})
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
	args = append(args, coremodels.UsageFundingSourceHosted)

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
