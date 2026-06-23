package config

import (
	"fmt"
	"os"
	"strings"
)

func RabbitMQURL() (string, error) {
	URL := os.Getenv("RABBITMQ_URL")
	if URL == "" {
		return "", fmt.Errorf("RABBITMQ_URL not set")
	}

	return URL, nil
}

func UsageGRPCURL() string {
	return os.Getenv("USAGE_GRPC_URL")
}

func MaxEmitCount() int {
	return intFromEnv("SUPERPLANE_MAX_EMIT_COUNT", 100)
}

func MaxPayloadSize() int {
	return intFromEnv("SUPERPLANE_MAX_PAYLOAD_SIZE", 64*1024)
}

// AnthropicAgentConfig holds the credentials and identifiers needed to talk
// to a single Anthropic managed agent. Empty values mean managed agents are
// disabled on this installation.
type AnthropicAgentConfig struct {
	APIKey        string
	AgentID       string
	EnvironmentID string
}

const (
	AgentProviderAnthropic = "anthropic"
	AgentProviderNative    = "native"

	NativeAgentLLMProviderOpenAI    = "openai"
	NativeAgentLLMProviderAnthropic = "anthropic"
)

// AgentProviderName selects which agents.Provider implementation should back
// chat sessions. Empty env keeps existing Anthropic managed-agent behavior.
func AgentProviderName() string {
	name := strings.TrimSpace(os.Getenv("SUPERPLANE_AGENT_PROVIDER"))
	if name == "" {
		return AgentProviderAnthropic
	}
	return name
}

// LoadAnthropicAgentConfig reads the env vars for the Anthropic managed-agents
// integration. If any required value is missing, Enabled() returns false.
func LoadAnthropicAgentConfig() AnthropicAgentConfig {
	return AnthropicAgentConfig{
		APIKey:        os.Getenv("ANTHROPIC_API_KEY"),
		AgentID:       os.Getenv("ANTHROPIC_AGENT_ID"),
		EnvironmentID: os.Getenv("ANTHROPIC_ENVIRONMENT_ID"),
	}
}

// Enabled reports whether the Anthropic provider has the credentials it
// needs to run.
func (c AnthropicAgentConfig) Enabled() bool {
	return c.APIKey != "" && c.AgentID != "" && c.EnvironmentID != ""
}

type NativeAgentConfig struct {
	LLMProvider     string
	APIKey          string
	BaseURL         string
	Model           string
	MaxSteps        int
	MaxToolCalls    int
	MaxContextChars int
	MaxRetries      int
}

func LoadNativeAgentConfig() NativeAgentConfig {
	return NativeAgentConfig{
		LLMProvider:     nativeAgentLLMProviderName(),
		APIKey:          os.Getenv("NATIVE_AGENT_API_KEY"),
		BaseURL:         os.Getenv("NATIVE_AGENT_BASE_URL"),
		Model:           os.Getenv("NATIVE_AGENT_MODEL"),
		MaxSteps:        intFromEnv("NATIVE_AGENT_MAX_STEPS", 12),
		MaxToolCalls:    intFromEnv("NATIVE_AGENT_MAX_TOOL_CALLS", 20),
		MaxContextChars: intFromEnv("NATIVE_AGENT_MAX_CONTEXT_CHARS", 120000),
		MaxRetries:      intFromEnv("NATIVE_AGENT_MAX_RETRIES", 3),
	}
}

func (c NativeAgentConfig) Enabled() bool {
	return c.APIKey != "" && c.Model != ""
}

func nativeAgentLLMProviderName() string {
	name := strings.TrimSpace(os.Getenv("NATIVE_AGENT_LLM_PROVIDER"))
	if name == "" {
		return NativeAgentLLMProviderOpenAI
	}
	return name
}
