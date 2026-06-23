package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxEmitCount(t *testing.T) {
	t.Run("defaults to 100", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "")
		assert.Equal(t, 100, MaxEmitCount())
	})

	t.Run("reads SUPERPLANE_MAX_EMIT_COUNT", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "25")
		assert.Equal(t, 25, MaxEmitCount())
	})

	t.Run("ignores invalid env values", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "not-a-number")
		assert.Equal(t, 100, MaxEmitCount())
	})
}

func TestMaxPayloadSize(t *testing.T) {
	t.Run("defaults to 64 KiB", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "")
		assert.Equal(t, 64*1024, MaxPayloadSize())
	})

	t.Run("reads SUPERPLANE_MAX_PAYLOAD_SIZE", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "8192")
		assert.Equal(t, 8192, MaxPayloadSize())
	})

	t.Run("ignores invalid env values", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "not-a-number")
		assert.Equal(t, 64*1024, MaxPayloadSize())
	})
}

func TestAgentProviderName(t *testing.T) {
	t.Run("defaults to anthropic", func(t *testing.T) {
		t.Setenv("SUPERPLANE_AGENT_PROVIDER", "")
		assert.Equal(t, AgentProviderAnthropic, AgentProviderName())
	})

	t.Run("reads configured provider", func(t *testing.T) {
		t.Setenv("SUPERPLANE_AGENT_PROVIDER", AgentProviderNative)
		assert.Equal(t, AgentProviderNative, AgentProviderName())
	})
}

func TestLoadNativeAgentConfig(t *testing.T) {
	t.Setenv("NATIVE_AGENT_API_KEY", "key")
	t.Setenv("NATIVE_AGENT_LLM_PROVIDER", NativeAgentLLMProviderAnthropic)
	t.Setenv("NATIVE_AGENT_BASE_URL", "https://example.test/v1")
	t.Setenv("NATIVE_AGENT_MODEL", "fast-model")
	t.Setenv("NATIVE_AGENT_MAX_STEPS", "7")
	t.Setenv("NATIVE_AGENT_MAX_TOOL_CALLS", "5")
	t.Setenv("NATIVE_AGENT_MAX_CONTEXT_CHARS", "9000")
	t.Setenv("NATIVE_AGENT_MAX_RETRIES", "4")

	cfg := LoadNativeAgentConfig()

	assert.True(t, cfg.Enabled())
	assert.Equal(t, NativeAgentLLMProviderAnthropic, cfg.LLMProvider)
	assert.Equal(t, "key", cfg.APIKey)
	assert.Equal(t, "https://example.test/v1", cfg.BaseURL)
	assert.Equal(t, "fast-model", cfg.Model)
	assert.Equal(t, 7, cfg.MaxSteps)
	assert.Equal(t, 5, cfg.MaxToolCalls)
	assert.Equal(t, 9000, cfg.MaxContextChars)
	assert.Equal(t, 4, cfg.MaxRetries)
}

func TestLoadNativeAgentConfigDefaultsLLMProviderToOpenAI(t *testing.T) {
	t.Setenv("NATIVE_AGENT_LLM_PROVIDER", "")

	cfg := LoadNativeAgentConfig()

	assert.Equal(t, NativeAgentLLMProviderOpenAI, cfg.LLMProvider)
}
