package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents/native"
	"github.com/superplanehq/superplane/pkg/config"
)

func TestNewProviderReturnsNilWhenSelectedProviderIsDisabled(t *testing.T) {
	provider, err := NewProvider(Config{
		Provider: config.AgentProviderNative,
		Native:   config.NativeAgentConfig{},
	})

	require.NoError(t, err)
	assert.Nil(t, provider)
}

func TestNewProviderCreatesNativeProvider(t *testing.T) {
	provider, err := NewProvider(Config{
		Provider: config.AgentProviderNative,
		Native: config.NativeAgentConfig{
			APIKey:   "test-key",
			Model:    "fast-model",
			MaxSteps: 4,
		},
		NativeStore: native.NewMemoryStore(),
	})

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, native.ProviderName, provider.Name())
}

func TestNewProviderCreatesNativeProviderWithAnthropicLLM(t *testing.T) {
	provider, err := NewProvider(Config{
		Provider: config.AgentProviderNative,
		Native: config.NativeAgentConfig{
			LLMProvider: config.NativeAgentLLMProviderAnthropic,
			APIKey:      "test-key",
			Model:       "claude-test-model",
			MaxSteps:    4,
		},
		NativeStore: native.NewMemoryStore(),
	})

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, native.ProviderName, provider.Name())
}

func TestNewProviderRejectsUnsupportedNativeLLMProvider(t *testing.T) {
	provider, err := NewProvider(Config{
		Provider: config.AgentProviderNative,
		Native: config.NativeAgentConfig{
			LLMProvider: "bogus",
			APIKey:      "test-key",
			Model:       "fast-model",
		},
		NativeStore: native.NewMemoryStore(),
	})

	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "unsupported native agent LLM provider")
}

func TestNewProviderRejectsUnsupportedProvider(t *testing.T) {
	provider, err := NewProvider(Config{Provider: "bogus"})

	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "unsupported agent provider")
}
