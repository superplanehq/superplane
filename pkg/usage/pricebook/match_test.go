package pricebook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchModel_LongestPrefixWins(t *testing.T) {
	prefixes := []PrefixRate{
		{Prefix: "gpt-4o", Rate: Rate{Input: 250}},
		{Prefix: "gpt-4o-mini", Rate: Rate{Input: 15}},
	}

	mini, ok := MatchModel("openai/gpt-4o-mini-2024-07-18", prefixes, nil)
	assert.True(t, ok)
	assert.Equal(t, ModelMatch{Key: "gpt-4o-mini", Mode: "prefix"}, mini)

	full, ok := MatchModel("gpt-4o-2024-08-06", prefixes, nil)
	assert.True(t, ok)
	assert.Equal(t, ModelMatch{Key: "gpt-4o", Mode: "prefix"}, full)
}

func TestMatchModel_FamilyFallback(t *testing.T) {
	families := []FamilyRate{
		{Token: "opus", Rate: Rate{Input: 1500}},
		{Token: "sonnet", Rate: Rate{Input: 300}},
	}

	got, ok := MatchModel("claude-3-5-sonnet-20241022", nil, families)
	assert.True(t, ok)
	assert.Equal(t, ModelMatch{Key: "sonnet", Mode: "family"}, got)
}

func TestCatalogModelID(t *testing.T) {
	assert.Equal(t, "x-ai/grok-4.6", CatalogModelID("openrouter/x-ai/grok-4.6"))
	assert.Equal(t, "x-ai/grok-4.6", CatalogModelID("x-ai/grok-4.6"))
	assert.Equal(t, "anthropic/claude-sonnet-4-6", CatalogModelID("openrouter/openrouter/anthropic/claude-sonnet-4-6"))
	assert.Equal(t, "claude-sonnet-4-6", CatalogModelID("claude-sonnet-4-6"))
	assert.Equal(t, "openrouter/free", CatalogModelID("openrouter/free"))
	assert.Equal(t, "openrouter/free", CatalogModelID("openrouter/openrouter/free"))
}

func TestCentsPerMillionFromUSDPerToken(t *testing.T) {
	cents, ok := CentsPerMillionFromUSDPerToken("0.000003")
	assert.True(t, ok)
	assert.Equal(t, int64(300), cents)

	cents, ok = CentsPerMillionFromUSDPerToken("")
	assert.True(t, ok)
	assert.Equal(t, int64(0), cents)

	_, ok = CentsPerMillionFromUSDPerToken("-0.1")
	assert.False(t, ok)

	_, ok = CentsPerMillionFromUSDPerToken("not-a-number")
	assert.False(t, ok)
}
