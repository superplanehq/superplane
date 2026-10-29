package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/models"
)

func TestModelThinkingLevelsForClaudeCode(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"claude-opus-5-5":            {"low", "medium", "high", "xhigh", "max"},
		"claude-opus-4-7":            {"low", "medium", "high", "xhigh", "max"},
		"claude-sonnet-4-6":          {"low", "medium", "high", "max"},
		"claude-opus-4-6":            {"low", "medium", "high", "max"},
		"claude-opus-4-5-20251101":   {"low", "medium", "high"},
		"claude-sonnet-4-5-20250929": nil,
		"claude-sonnet-4-20250514":   nil,
		"claude-haiku-4-5":           nil,
		"claude-3-7-sonnet-20250219": nil,
		"opus":                       nil,
	}
	for model, want := range cases {
		assert.Equal(t, want, nilIfEmpty(ModelThinkingLevels(models.UsageProviderAnthropic, model)), model)
	}
}

func TestModelThinkingLevelsForCodex(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"gpt-5":              {"minimal", "low", "medium", "high"},
		"gpt-5-mini":         {"minimal", "low", "medium", "high"},
		"gpt-5-codex":        {"low", "medium", "high"},
		"gpt-5.1":            {"none", "low", "medium", "high"},
		"gpt-5.1-codex-max":  {"low", "medium", "high", "xhigh"},
		"gpt-5.2-codex":      {"low", "medium", "high", "xhigh"},
		"gpt-5.3-codex":      {"none", "low", "medium", "high", "xhigh"},
		"gpt-5.4":            {"none", "low", "medium", "high", "xhigh"},
		"gpt-5-pro":          {"high"},
		"gpt-5.4-pro":        {"medium", "high", "xhigh"},
		"gpt-5.2-chat":       {"medium"},
		"o3":                 {"low", "medium", "high"},
		"o4-mini":            {"low", "medium", "high"},
		"gpt-4o":             nil,
		"gpt-4.1":            nil,
		"gpt-5-2025-08-07":   {"minimal", "low", "medium", "high"},
		"text-embedding-3-l": nil,
	}
	for model, want := range cases {
		assert.Equal(t, want, nilIfEmpty(ModelThinkingLevels(models.UsageProviderOpenAI, model)), model)
	}
}

func TestModelThinkingLevelsForOpenCode(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"anthropic/claude-opus-5.5":            {"low", "medium", "high"},
		"anthropic/claude-sonnet-4.6":          {"low", "medium", "high"},
		"openai/gpt-5.4":                       {"none", "low", "medium", "high", "xhigh"},
		"openai/gpt-4o":                        nil,
		"google/gemini-2.5-pro":                {"low", "medium", "high"},
		"google/gemini-3-pro-preview":          {"low", "medium", "high"},
		"x-ai/grok-3-mini":                     {"low", "high"},
		"x-ai/grok-4.6":                        nil,
		"deepseek/deepseek-r1":                 nil,
		"moonshotai/kimi-k2":                   nil,
		"openrouter/anthropic/claude-opus-4.7": {"low", "medium", "high"},
	}
	for model, want := range cases {
		assert.Equal(t, want, nilIfEmpty(ModelThinkingLevels(models.UsageProviderOpenRouter, model)), model)
	}
}

func TestModelThinkingLevelsWithoutCatalogProvider(t *testing.T) {
	t.Parallel()

	assert.Empty(t, ModelThinkingLevels(models.UsageProviderCustom, "gpt-5.4"))
	assert.Empty(t, ModelThinkingLevels(models.UsageProviderAnthropic, ""))
}

func TestModelThinkingLevelsReturnsCopy(t *testing.T) {
	t.Parallel()

	levels := ModelThinkingLevels(models.UsageProviderAnthropic, "claude-opus-5-5")
	levels[0] = "changed"
	assert.Equal(t, ThinkingLevelLow, ModelThinkingLevels(models.UsageProviderAnthropic, "claude-opus-5-5")[0])
}

func TestSupportedThinkingLevelsKeepsKnownLevelsInOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"low", "high", "xhigh"}, SupportedThinkingLevels([]string{"xhigh", "High", "turbo", "low"}))
	assert.Empty(t, SupportedThinkingLevels(nil))
}

func nilIfEmpty(levels []string) []string {
	if len(levels) == 0 {
		return nil
	}
	return levels
}
