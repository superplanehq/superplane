package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test__AllocateHostedCreditSpendWelcomeBeforeIncludedAndTopup(t *testing.T) {
	now := time.Now()
	welcomeEnd := now.Add(14 * 24 * time.Hour)
	includedEnd := now.Add(10 * 24 * time.Hour)
	topupEnd := now.AddDate(0, 12, 0)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindIncluded, AmountMicros: CentsToMicros(5000), ExpiresAt: &includedEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(5000), ExpiresAt: &topupEnd, CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(2000), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(13000), spend.RemainingMicros)
	assert.Equal(t, CentsToMicros(3000), spend.WelcomeRemainingMicros)
	assert.Equal(t, CentsToMicros(5000), spend.IncludedRemainingMicros)
	assert.Equal(t, CentsToMicros(5000), spend.PurchasedRemainingMicros)
	assert.Equal(t, int64(0), spend.AdminRemainingMicros)
}

func Test__AllocateHostedCreditSpendDoesNotCountWelcomeAsIncluded(t *testing.T) {
	now := time.Now()
	welcomeEnd := now.Add(14 * 24 * time.Hour)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeEnd, CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(876), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(4124), spend.RemainingMicros)
	assert.Equal(t, CentsToMicros(4124), spend.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), spend.IncludedRemainingMicros)
	assert.Equal(t, int64(0), spend.PurchasedRemainingMicros)
}

func Test__AllocateHostedCreditSpendIncludedBeforeTopup(t *testing.T) {
	now := time.Now()
	includedEnd := now.Add(10 * 24 * time.Hour)
	topupEnd := now.AddDate(0, 12, 0)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindIncluded, AmountMicros: CentsToMicros(5000), ExpiresAt: &includedEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(5000), ExpiresAt: &topupEnd, CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(2000), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(8000), spend.RemainingMicros)
	assert.Equal(t, int64(0), spend.WelcomeRemainingMicros)
	assert.Equal(t, CentsToMicros(3000), spend.IncludedRemainingMicros)
	assert.Equal(t, CentsToMicros(5000), spend.PurchasedRemainingMicros)
}

func Test__AllocateHostedCreditSpendStaggeredExpiry(t *testing.T) {
	now := time.Now()
	includedExpired := now.Add(-time.Hour)
	topupEnd := now.AddDate(0, 12, 0)
	created := now.Add(-20 * 24 * time.Hour)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindIncluded, AmountMicros: CentsToMicros(5000), ExpiresAt: &includedExpired, CreatedAt: created},
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(5000), ExpiresAt: &topupEnd, CreatedAt: created},
	}
	billedBefore := map[int64]int64{
		includedExpired.UTC().UnixNano(): CentsToMicros(4000),
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(4000), billedBefore, now)
	assert.Equal(t, CentsToMicros(5000), spend.RemainingMicros)
	assert.Equal(t, int64(0), spend.IncludedRemainingMicros)
	assert.Equal(t, CentsToMicros(5000), spend.PurchasedRemainingMicros)
}

func Test__AllocateHostedCreditSpendExpiredWelcomeDoesNotReduceTopup(t *testing.T) {
	now := time.Now()
	welcomeExpired := now.Add(-time.Minute)
	topupEnd := now.AddDate(0, 12, 0)
	created := now.Add(-15 * 24 * time.Hour)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeExpired, CreatedAt: created},
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(2500), ExpiresAt: &topupEnd, CreatedAt: now},
	}
	billedBefore := map[int64]int64{
		welcomeExpired.UTC().UnixNano(): CentsToMicros(2000),
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(2000), billedBefore, now)
	assert.Equal(t, CentsToMicros(2500), spend.RemainingMicros)
	assert.Equal(t, CentsToMicros(2500), spend.PurchasedRemainingMicros)
}

func Test__AllocateHostedCreditSpendAdminLast(t *testing.T) {
	now := time.Now()
	welcomeEnd := now.Add(14 * 24 * time.Hour)
	includedEnd := now.Add(10 * 24 * time.Hour)
	topupEnd := now.AddDate(0, 12, 0)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindIncluded, AmountMicros: CentsToMicros(5000), ExpiresAt: &includedEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(104278), ExpiresAt: &topupEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindAdmin, AmountMicros: CentsToMicros(11000), CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(0), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(125278), spend.RemainingMicros)
	assert.Equal(t, CentsToMicros(5000), spend.WelcomeRemainingMicros)
	assert.Equal(t, CentsToMicros(5000), spend.IncludedRemainingMicros)
	assert.Equal(t, CentsToMicros(104278), spend.PurchasedRemainingMicros)
	assert.Equal(t, CentsToMicros(11000), spend.AdminRemainingMicros)

	spent := allocateHostedCreditSpend(grants, CentsToMicros(124278), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(1000), spent.RemainingMicros)
	assert.Equal(t, int64(0), spent.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), spent.IncludedRemainingMicros)
	assert.Equal(t, int64(0), spent.PurchasedRemainingMicros)
	assert.Equal(t, CentsToMicros(1000), spent.AdminRemainingMicros)
}

func Test__AllocateHostedCreditSpendPositiveAdjustmentAddsToItsType(t *testing.T) {
	now := time.Now()
	welcomeEnd := now.Add(14 * 24 * time.Hour)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeEnd, CreatedAt: now.Add(-time.Hour)},
		{Kind: LLMCreditGrantKindTrialAdjustment, AmountMicros: CentsToMicros(1000), ExpiresAt: &welcomeEnd, CreatedAt: now},
		{Kind: LLMCreditGrantKindTopupAdjustment, AmountMicros: CentsToMicros(2000), CreatedAt: now},
		{Kind: LLMCreditGrantKindAdminAdjustment, AmountMicros: CentsToMicros(3000), CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(5500), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(500), spend.WelcomeRemainingMicros)
	assert.Equal(t, CentsToMicros(2000), spend.PurchasedRemainingMicros)
	assert.Equal(t, CentsToMicros(3000), spend.AdminRemainingMicros)
	assert.Equal(t, CentsToMicros(2000), spend.PurchasedCreditMicros)
	assert.Equal(t, CentsToMicros(9000), spend.SuperPlaneGrantMicros)
}

func Test__AllocateHostedCreditSpendNegativeAdjustmentLowersRemaining(t *testing.T) {
	now := time.Now()
	welcomeEnd := now.Add(14 * 24 * time.Hour)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeEnd, CreatedAt: now.Add(-time.Hour)},
		{Kind: LLMCreditGrantKindTrialAdjustment, AmountMicros: -CentsToMicros(2000), CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(2000), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(1000), spend.WelcomeRemainingMicros)
	assert.Equal(t, CentsToMicros(1000), spend.RemainingMicros)
	assert.Equal(t, CentsToMicros(3000), spend.SuperPlaneGrantMicros)
}

func Test__AllocateHostedCreditSpendSpillsPastLoweredType(t *testing.T) {
	now := time.Now()
	welcomeEnd := now.Add(14 * 24 * time.Hour)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindWelcome, AmountMicros: CentsToMicros(5000), ExpiresAt: &welcomeEnd, CreatedAt: now.Add(-time.Hour)},
		{Kind: LLMCreditGrantKindTrialAdjustment, AmountMicros: -CentsToMicros(4000), CreatedAt: now},
		{Kind: LLMCreditGrantKindAdmin, AmountMicros: CentsToMicros(5000), CreatedAt: now.Add(-time.Hour)},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(3000), map[int64]int64{}, now)
	assert.Equal(t, int64(0), spend.WelcomeRemainingMicros)
	assert.Equal(t, CentsToMicros(3000), spend.AdminRemainingMicros)
	assert.Equal(t, CentsToMicros(3000), spend.RemainingMicros)
}

func Test__AllocateHostedCreditSpendNegativeAdjustmentLowersLastSpentGrantFirst(t *testing.T) {
	now := time.Now()
	created := now.Add(-time.Hour)
	soonEnd := now.AddDate(0, 1, 0)
	laterEnd := now.AddDate(0, 12, 0)
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(10000), ExpiresAt: &soonEnd, CreatedAt: created},
		{Kind: LLMCreditGrantKindTopup, AmountMicros: CentsToMicros(10000), ExpiresAt: &laterEnd, CreatedAt: created},
		{Kind: LLMCreditGrantKindTopupAdjustment, AmountMicros: -CentsToMicros(13000), CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, CentsToMicros(5000), map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(2000), spend.PurchasedRemainingMicros)
	assert.Equal(t, CentsToMicros(7000), spend.PurchasedCreditMicros)
}

func Test__AllocateHostedCreditSpendNegativeAdjustmentSkipsLaterGrants(t *testing.T) {
	now := time.Now()
	grants := []OrganizationLLMCreditGrant{
		{Kind: LLMCreditGrantKindAdmin, AmountMicros: CentsToMicros(5000), CreatedAt: now.Add(-2 * time.Hour)},
		{Kind: LLMCreditGrantKindAdminAdjustment, AmountMicros: -CentsToMicros(5000), CreatedAt: now.Add(-time.Hour)},
		{Kind: LLMCreditGrantKindAdmin, AmountMicros: CentsToMicros(2000), CreatedAt: now},
	}

	spend := allocateHostedCreditSpend(grants, 0, map[int64]int64{}, now)
	assert.Equal(t, CentsToMicros(2000), spend.AdminRemainingMicros)
}
