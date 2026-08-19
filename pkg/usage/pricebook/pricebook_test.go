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

func TestEstimateMicros_UnknownModelIsZero(t *testing.T) {
	got := EstimateMicros("openai", "unknown-lab-model", 10_000, 10_000, 0, 0, 0)
	assert.Equal(t, int64(0), got)
}

func TestMicrosToCents(t *testing.T) {
	assert.Equal(t, int64(3), MicrosToCents(30_000))
	assert.Equal(t, int64(0), MicrosToCents(9_999))
	assert.Equal(t, int64(0), MicrosToCents(-1))
}
