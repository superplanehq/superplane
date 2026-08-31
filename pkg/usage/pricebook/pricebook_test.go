package pricebook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateMicros_KnownModel(t *testing.T) {
	// 1M sonnet input tokens at $3/M = 300 cents = 3_000_000 micros.
	got := EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0)
	assert.Equal(t, int64(3_000_000), got)
}

func TestEstimateMicros_LongestPrefixWins(t *testing.T) {
	mini := EstimateMicros("openai", "gpt-4o-mini-2024-07-18", 1_000_000, 0, 0, 0, 0)
	full := EstimateMicros("openai", "gpt-4o-2024-08-06", 1_000_000, 0, 0, 0, 0)
	assert.Equal(t, int64(150_000), mini)
	assert.Equal(t, int64(2_500_000), full)
}

func TestEstimateMicros_MiniDoesNotInheritFlagshipRate(t *testing.T) {
	o3Mini := EstimateMicros("openai", "o3-mini", 1_000_000, 0, 0, 0, 0)
	o3 := EstimateMicros("openai", "o3", 1_000_000, 0, 0, 0, 0)
	gpt5Mini := EstimateMicros("openai", "gpt-5-mini", 1_000_000, 0, 0, 0, 0)
	gpt5 := EstimateMicros("openai", "gpt-5", 1_000_000, 0, 0, 0, 0)

	assert.Equal(t, int64(1_100_000), o3Mini)
	assert.Equal(t, int64(20_000_000), o3)
	assert.Equal(t, int64(250_000), gpt5Mini)
	assert.Equal(t, int64(1_250_000), gpt5)
}

func TestEstimateMicros_DatedClaudeFamilyIDs(t *testing.T) {
	sonnet := EstimateMicros("anthropic", "claude-3-5-sonnet-20241022", 1_000_000, 0, 0, 0, 0)
	opus := EstimateMicros("anthropic", "claude-3-opus-20240229", 1_000_000, 0, 0, 0, 0)
	assert.Equal(t, int64(3_000_000), sonnet)
	assert.Equal(t, int64(15_000_000), opus)
}

func TestEstimateMicros_ProviderPrefixedModel(t *testing.T) {
	got := EstimateMicros("openrouter", "openai/gpt-5-mini", 1_000_000, 0, 0, 0, 0)
	assert.Equal(t, int64(250_000), got)
}

func TestEstimateMicros_UnknownModelIsZero(t *testing.T) {
	got := EstimateMicros("openai", "unknown-lab-model", 10_000, 10_000, 0, 0, 0)
	assert.Equal(t, int64(0), got)
}

func TestIsPriced(t *testing.T) {
	assert.True(t, IsPriced("claude-sonnet-4-6"))
	assert.False(t, IsPriced("unknown-lab-model"))
}

func TestEstimateMicros_OpenAICacheReadIsPriced(t *testing.T) {
	uncached := EstimateMicros("openai", "gpt-4o", 1_000_000, 0, 0, 0, 0)
	cached := EstimateMicros("openai", "gpt-4o", 0, 0, 1_000_000, 0, 0)
	miniCached := EstimateMicros("openai", "gpt-4o-mini", 0, 0, 1_000_000, 0, 0)

	assert.Equal(t, int64(2_500_000), uncached)
	assert.Equal(t, int64(250_000), cached)
	assert.Equal(t, int64(10_000), miniCached)
}

func TestMicrosToCents(t *testing.T) {
	assert.Equal(t, int64(3), MicrosToCents(30_000))
	assert.Equal(t, int64(0), MicrosToCents(9_999))
	assert.Equal(t, int64(0), MicrosToCents(-1))
}

func TestEstimateComputeMicros(t *testing.T) {
	t.Cleanup(Reset)
	SetComputeRates(map[string]int64{"e1-large-amd64": 50})

	assert.Equal(t, int64(500), EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 10))
	assert.Equal(t, int64(0), EstimateComputeMicros("e1-large-amd64", "local", 10))
	assert.Equal(t, int64(0), EstimateComputeMicros("e1-tiny-amd64", "e1-tiny-amd64", 10))
	assert.Equal(t, int64(0), EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 0))
}

func TestEstimateComputeMicros_DefaultCatalogRates(t *testing.T) {
	t.Cleanup(Reset)
	Reset()

	assert.Equal(t, int64(5560), EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 10))
	assert.Equal(t, int64(1390), EstimateComputeMicros("e1-tiny-arm64", "e1-tiny-arm64", 10))
	assert.Equal(t, int64(0), EstimateComputeMicros("e1-large-amd64", "local", 10))
}
