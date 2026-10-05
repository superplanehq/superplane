package models

import (
	"sort"
	"time"
)

type hostedCreditSpend struct {
	RemainingMicros          int64
	IncludedRemainingMicros  int64
	PurchasedRemainingMicros int64
	WelcomeRemainingMicros   int64
	AdminRemainingMicros     int64
	SuperPlaneGrantMicros    int64
	PurchasedCreditMicros    int64
	WelcomeCreditExpiresAt   *time.Time
}

// creditBucket is the credit type a grant pays into. Its value is the spend
// order: hosted runs spend lower buckets first.
type creditBucket int

const (
	creditBucketWelcome creditBucket = iota
	creditBucketIncluded
	creditBucketPurchased
	creditBucketAdmin
	creditBucketOther
)

func grantCreditBucket(kind string) creditBucket {
	switch kind {
	case LLMCreditGrantKindWelcome, LLMCreditGrantKindTrialAdjustment:
		return creditBucketWelcome
	case LLMCreditGrantKindIncluded:
		return creditBucketIncluded
	case LLMCreditGrantKindTopup, LLMCreditGrantKindTopupRefund, LLMCreditGrantKindTopupAdjustment:
		return creditBucketPurchased
	case LLMCreditGrantKindAdmin, LLMCreditGrantKindAdminAdjustment:
		return creditBucketAdmin
	default:
		return creditBucketOther
	}
}

func isPurchasedGrantKind(kind string) bool {
	return grantCreditBucket(kind) == creditBucketPurchased
}

func isCreditAdjustmentKind(kind string) bool {
	switch kind {
	case LLMCreditGrantKindTrialAdjustment, LLMCreditGrantKindTopupAdjustment, LLMCreditGrantKindAdminAdjustment:
		return true
	default:
		return false
	}
}

func grantSpendPriority(kind string) int {
	return int(grantCreditBucket(kind))
}

func allocateHostedCreditSpend(
	grants []OrganizationLLMCreditGrant,
	billedMicros int64,
	billedAtOrBefore map[int64]int64,
	now time.Time,
) hostedCreditSpend {
	var welcomeExpiresAt *time.Time
	var superplane, purchased int64
	for _, grant := range grants {
		if grant.Kind == LLMCreditGrantKindWelcome {
			welcomeExpiresAt = grant.ExpiresAt
		}
		switch grantCreditBucket(grant.Kind) {
		case creditBucketWelcome, creditBucketIncluded, creditBucketAdmin:
			superplane += grant.AmountMicros
		case creditBucketPurchased:
			purchased += grant.AmountMicros
		}
	}

	expired, live := splitCreditGrantsForSpend(grants, now)
	consumedExpired := consumedExpiredGrantSpend(expired, billedAtOrBefore)

	spend := billedMicros - consumedExpired
	if spend < 0 {
		spend = 0
	}

	var includedRemaining, purchasedRemaining, adminRemaining, welcomeRemaining int64
	for _, grant := range live {
		left := grant.AmountMicros
		if left > 0 {
			take := left
			if take > spend {
				take = spend
			}
			left -= take
			spend -= take
		}
		switch grantCreditBucket(grant.Kind) {
		case creditBucketWelcome:
			welcomeRemaining += left
		case creditBucketIncluded:
			includedRemaining += left
		case creditBucketPurchased:
			purchasedRemaining += left
		case creditBucketAdmin:
			adminRemaining += left
		}
	}
	if includedRemaining < 0 {
		includedRemaining = 0
	}
	if purchasedRemaining < 0 {
		purchasedRemaining = 0
	}
	if adminRemaining < 0 {
		adminRemaining = 0
	}
	if welcomeRemaining < 0 {
		welcomeRemaining = 0
	}

	return hostedCreditSpend{
		RemainingMicros:          welcomeRemaining + includedRemaining + purchasedRemaining + adminRemaining,
		IncludedRemainingMicros:  includedRemaining,
		PurchasedRemainingMicros: purchasedRemaining,
		WelcomeRemainingMicros:   welcomeRemaining,
		AdminRemainingMicros:     adminRemaining,
		SuperPlaneGrantMicros:    superplane,
		PurchasedCreditMicros:    purchased,
		WelcomeCreditExpiresAt:   welcomeExpiresAt,
	}
}

func splitCreditGrantsForSpend(grants []OrganizationLLMCreditGrant, now time.Time) (expired, live []OrganizationLLMCreditGrant) {
	for _, grant := range applyNegativeCreditAdjustments(grants) {
		if grant.IsExpired(now) {
			expired = append(expired, grant)
			continue
		}
		live = append(live, grant)
	}
	sort.SliceStable(expired, func(i, j int) bool {
		return grantExpiresBefore(expired[i], expired[j])
	})
	sortGrantsBySpendOrder(live)
	return expired, live
}

func consumedExpiredGrantSpend(expired []OrganizationLLMCreditGrant, billedAtOrBefore map[int64]int64) int64 {
	consumed := int64(0)
	for _, grant := range expired {
		if grant.AmountMicros <= 0 || grant.ExpiresAt == nil {
			continue
		}
		billed := billedAtOrBefore[grant.ExpiresAt.UTC().UnixNano()]
		available := billed - consumed
		if available < 0 {
			available = 0
		}
		take := grant.AmountMicros
		if take > available {
			take = available
		}
		consumed += take
	}
	return consumed
}

func liveHostedSpendMicros(grants []OrganizationLLMCreditGrant, billedMicros int64, billedAtOrBefore map[int64]int64, now time.Time) int64 {
	expired, _ := splitCreditGrantsForSpend(grants, now)
	spend := billedMicros - consumedExpiredGrantSpend(expired, billedAtOrBefore)
	if spend < 0 {
		return 0
	}
	return spend
}

func liveWelcomeCapacityMicros(grants []OrganizationLLMCreditGrant, now time.Time) int64 {
	_, live := splitCreditGrantsForSpend(grants, now)
	var capacity int64
	for _, grant := range live {
		if grantCreditBucket(grant.Kind) != creditBucketWelcome || grant.AmountMicros <= 0 {
			continue
		}
		capacity += grant.AmountMicros
	}
	return capacity
}

// applyNegativeCreditAdjustments returns a copy of grants without negative
// adjustment rows. Each negative adjustment lowers the amount of the same-type
// grants that were live when the admin made it, starting with the grant that
// hosted runs spend last. A lower amount, not a negative spend line, lets spend
// move on to the next credit type when the lowered type runs out.
func applyNegativeCreditAdjustments(grants []OrganizationLLMCreditGrant) []OrganizationLLMCreditGrant {
	var reductions []OrganizationLLMCreditGrant
	capacity := make([]OrganizationLLMCreditGrant, 0, len(grants))
	for _, grant := range grants {
		if isCreditAdjustmentKind(grant.Kind) && grant.AmountMicros < 0 {
			reductions = append(reductions, grant)
			continue
		}
		capacity = append(capacity, grant)
	}

	sort.SliceStable(reductions, func(i, j int) bool {
		return reductions[i].CreatedAt.Before(reductions[j].CreatedAt)
	})
	for _, reduction := range reductions {
		reduceGrantCapacity(capacity, reduction)
	}
	return capacity
}

func reduceGrantCapacity(grants []OrganizationLLMCreditGrant, reduction OrganizationLLMCreditGrant) {
	bucket := grantCreditBucket(reduction.Kind)
	candidates := make([]int, 0)
	for i, grant := range grants {
		if grantCreditBucket(grant.Kind) != bucket || grant.AmountMicros <= 0 {
			continue
		}
		if grant.CreatedAt.After(reduction.CreatedAt) || grant.IsExpired(reduction.CreatedAt) {
			continue
		}
		candidates = append(candidates, i)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return grantExpiresBefore(grants[candidates[j]], grants[candidates[i]])
	})

	left := -reduction.AmountMicros
	for _, index := range candidates {
		if left <= 0 {
			return
		}
		take := grants[index].AmountMicros
		if take > left {
			take = left
		}
		grants[index].AmountMicros -= take
		left -= take
	}
}

func sortGrantsBySpendOrder(grants []OrganizationLLMCreditGrant) {
	sort.SliceStable(grants, func(i, j int) bool {
		pi, pj := grantSpendPriority(grants[i].Kind), grantSpendPriority(grants[j].Kind)
		if pi != pj {
			return pi < pj
		}
		return grantExpiresBefore(grants[i], grants[j])
	})
}

func grantExpiresBefore(a, b OrganizationLLMCreditGrant) bool {
	if a.ExpiresAt == nil && b.ExpiresAt == nil {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	if a.ExpiresAt == nil {
		return false
	}
	if b.ExpiresAt == nil {
		return true
	}
	if a.ExpiresAt.Equal(*b.ExpiresAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	return a.ExpiresAt.Before(*b.ExpiresAt)
}
