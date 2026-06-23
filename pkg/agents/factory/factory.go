package factory

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/agents/anthropic"
	"github.com/superplanehq/superplane/pkg/agents/native"
	"github.com/superplanehq/superplane/pkg/agents/native/llm"
	nativeanthropic "github.com/superplanehq/superplane/pkg/agents/native/llm/anthropic"
	"github.com/superplanehq/superplane/pkg/agents/native/llm/openai"
	"github.com/superplanehq/superplane/pkg/config"
)

type Config struct {
	Provider    string
	Anthropic   config.AnthropicAgentConfig
	Native      config.NativeAgentConfig
	NativeStore native.Store
}

func LoadConfig() Config {
	return Config{
		Provider:  config.AgentProviderName(),
		Anthropic: config.LoadAnthropicAgentConfig(),
		Native:    config.LoadNativeAgentConfig(),
	}
}

func NewProvider(cfg Config) (agents.Provider, error) {
	switch cfg.Provider {
	case "", config.AgentProviderAnthropic:
		if !cfg.Anthropic.Enabled() {
			return nil, nil
		}
		return anthropic.New(anthropic.Config{
			APIKey:        cfg.Anthropic.APIKey,
			AgentID:       cfg.Anthropic.AgentID,
			EnvironmentID: cfg.Anthropic.EnvironmentID,
		})
	case config.AgentProviderNative:
		if !cfg.Native.Enabled() {
			return nil, nil
		}
		client, err := newNativeLLMClient(cfg.Native)
		if err != nil {
			return nil, err
		}
		store := cfg.NativeStore
		if store == nil {
			store = native.NewDatabaseStore()
		}
		return native.New(native.Config{
			Client:          client,
			Model:           cfg.Native.Model,
			MaxSteps:        cfg.Native.MaxSteps,
			MaxToolCalls:    cfg.Native.MaxToolCalls,
			MaxContextChars: cfg.Native.MaxContextChars,
			Store:           store,
		})
	default:
		return nil, fmt.Errorf("unsupported agent provider %q", cfg.Provider)
	}
}

func newNativeLLMClient(cfg config.NativeAgentConfig) (llm.Client, error) {
	switch cfg.LLMProvider {
	case "", config.NativeAgentLLMProviderOpenAI:
		return openai.New(openai.Config{
			APIKey:     cfg.APIKey,
			BaseURL:    cfg.BaseURL,
			Model:      cfg.Model,
			MaxRetries: cfg.MaxRetries,
		})
	case config.NativeAgentLLMProviderAnthropic:
		return nativeanthropic.New(nativeanthropic.Config{
			APIKey:     cfg.APIKey,
			BaseURL:    cfg.BaseURL,
			Model:      cfg.Model,
			MaxRetries: cfg.MaxRetries,
		})
	default:
		return nil, fmt.Errorf("unsupported native agent LLM provider %q", cfg.LLMProvider)
	}
}
