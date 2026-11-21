package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Numbers from #9049: one Codex Implement step where 98.6% of input was cached.
func TestOpenAIUsage_CachedTokensAreNotCountedTwice(t *testing.T) {
	var u openAIUsage
	u.PromptTokens = 56_305_375
	u.CompletionTokens = 223_089
	u.TotalTokens = 56_528_464
	u.PromptTokensDetails.CachedTokens = 55_524_598

	usage := u.toUsage()

	assert.Equal(t, int64(780_777), usage.InputTokens)
	assert.Equal(t, int64(55_524_598), usage.CacheReadTokens)
	assert.Equal(t, int64(223_089), usage.OutputTokens)
	assert.Equal(t, int64(56_528_464), usage.TotalTokens)
}

func TestOpenAIUsage_NoCacheLeavesInputUnchanged(t *testing.T) {
	var u openAIUsage
	u.PromptTokens = 1_000
	u.CompletionTokens = 50

	usage := u.toUsage()

	assert.Equal(t, int64(1_000), usage.InputTokens)
	assert.Equal(t, int64(0), usage.CacheReadTokens)
}
