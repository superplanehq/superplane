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
	SuperPlaneGrantMicros    int64
	PurchasedCreditMicros    int64
	WelcomeCreditExpiresAt   *time.Time
}

func isPurchasedGrantKind(kind string) bool {
	switch kind {
	case LLMCreditGrantKindTopup, LLMCreditGrantKindTopupRefund:
		return true
	default:
		return false
	}
}

func grantSpendPriority(kind string) int {
	switch kind {
	case LLMCreditGrantKindWelcome:
		return 0
	case LLMCreditGrantKindIncluded:
		return 1
	case LLMCreditGrantKindTopup, LLMCreditGrantKindTopupRefund:
		return 2
	case LLMCreditGrantKindAdmin:
		return 3
	default:
		return 4
	}
}

func allocateHostedCreditSpend(
	grants []OrganizationLLMCreditGrant,
	billedMicros int64,
	billedAtOrBefore map[int64]int64,
	now time.Time,
) hostedCreditSpend {
	var expired, live []OrganizationLLMCreditGrant
	var welcomeExpiresAt *time.Time
	var superplane, purchased int64
	for _, grant := range grants {
		switch grant.Kind {
		case LLMCreditGrantKindWelcome:
			superplane += grant.AmountMicros
			welcomeExpiresAt = grant.ExpiresAt
		case LLMCreditGrantKindAdmin, LLMCreditGrantKindIncluded:
			superplane += grant.AmountMicros
		default:
			if isPurchasedGrantKind(grant.Kind) {
				purchased += grant.AmountMicros
			}
		}
		if grant.IsExpired(now) {
			expired = append(expired, grant)
			continue
		}
		live = append(live, grant)
	}

	sort.SliceStable(expired, func(i, j int) bool {
		return grantExpiresBefore(expired[i], expired[j])
	})
	sort.SliceStable(live, func(i, j int) bool {
		pi, pj := grantSpendPriority(live[i].Kind), grantSpendPriority(live[j].Kind)
		if pi != pj {
			return pi < pj
		}
		return grantExpiresBefore(live[i], live[j])
	})

	consumedExpired := int64(0)
	for _, grant := range expired {
		if grant.AmountMicros <= 0 || grant.ExpiresAt == nil {
			continue
		}
		billed := billedAtOrBefore[grant.ExpiresAt.UTC().UnixNano()]
		available := billed - consumedExpired
		if available < 0 {
			available = 0
		}
		take := grant.AmountMicros
		if take > available {
			take = available
		}
		consumedExpired += take
	}

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
		switch {
		case grant.Kind == LLMCreditGrantKindWelcome:
			welcomeRemaining += left
			includedRemaining += left
		case grant.Kind == LLMCreditGrantKindIncluded:
			includedRemaining += left
		case isPurchasedGrantKind(grant.Kind):
			purchasedRemaining += left
		case grant.Kind == LLMCreditGrantKindAdmin:
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
		RemainingMicros:          includedRemaining + purchasedRemaining + adminRemaining,
		IncludedRemainingMicros:  includedRemaining,
		PurchasedRemainingMicros: purchasedRemaining,
		WelcomeRemainingMicros:   welcomeRemaining,
		SuperPlaneGrantMicros:    superplane,
		PurchasedCreditMicros:    purchased,
		WelcomeCreditExpiresAt:   welcomeExpiresAt,
	}
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
