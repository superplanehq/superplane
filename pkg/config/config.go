package config

import (
	"fmt"
	"os"
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

func UsesDatabaseSMTPEmailSettings() bool {
	return os.Getenv("OWNER_SETUP_ENABLED") == "yes"
}

func ResendEmailConfigured() bool {
	return os.Getenv("RESEND_API_KEY") != "" &&
		os.Getenv("EMAIL_FROM_NAME") != "" &&
		os.Getenv("EMAIL_FROM_ADDRESS") != ""
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
