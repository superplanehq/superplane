package config

import (
	"encoding/json"
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

// SessionResource defines a file to mount in agent sessions.
type SessionResource struct {
	FileID    string `json:"file_id"`
	MountPath string `json:"mount_path"`
}

// LoadSessionResources reads the session resources config from ANTHROPIC_SESSION_RESOURCES_FILE env var.
func LoadSessionResources() []SessionResource {
	path := os.Getenv("ANTHROPIC_SESSION_RESOURCES_FILE")
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var resources []SessionResource
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil
	}
	return resources
}
