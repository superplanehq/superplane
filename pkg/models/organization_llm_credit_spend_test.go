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
