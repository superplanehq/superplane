package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultMaxEmitCount   = 100
	maxEmitCountEnvVar    = "SUPERPLANE_MAX_EMIT_COUNT"
	defaultMaxPayloadSize = 64 * 1024
	maxPayloadSizeEnvVar  = "SUPERPLANE_MAX_PAYLOAD_SIZE"
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

// MaxEmitCount returns the maximum number of events a single execution may emit at once.
// Defaults to 100. Override with SUPERPLANE_MAX_EMIT_COUNT.
func MaxEmitCount() int {
	if v := os.Getenv(maxEmitCountEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	return defaultMaxEmitCount
}

// MaxPayloadSize returns the maximum serialized event payload size in bytes.
// Defaults to 64 KiB. Override with SUPERPLANE_MAX_PAYLOAD_SIZE.
func MaxPayloadSize() int {
	if v := os.Getenv(maxPayloadSizeEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	return defaultMaxPayloadSize
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
