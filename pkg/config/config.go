package config

import (
	"fmt"
	"os"
)

const MaxWebhookPayloadSize = 512 * 1024

func RabbitMQURL() (string, error) {
	URL := os.Getenv("RABBITMQ_URL")
	if URL == "" {
		return "", fmt.Errorf("RABBITMQ_URL not set")
	}

	return URL, nil
}

func MaxEmitCount() int {
	return intFromEnv("SUPERPLANE_MAX_EMIT_COUNT", 100)
}

// EventRetentionWindowDays is how long finished canvas runs stay before
// EventRetentionWorker deletes them. Default is 180 days (~6 months) so
// leftover SaaS 14-day org values do not drop run history early.
func EventRetentionWindowDays() int {
	return intFromEnv("EVENT_RETENTION_WINDOW_DAYS", 180)
}

func MaxPayloadSize() int {
	return intFromEnv("SUPERPLANE_MAX_PAYLOAD_SIZE", 512*1024)
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
