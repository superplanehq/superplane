package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const MaxWebhookPayloadSize = 512 * 1024

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

const (
	EnvGitHubAppID            = "SUPERPLANE_GITHUB_APP_ID"
	EnvGitHubAppSlug          = "SUPERPLANE_GITHUB_APP_SLUG"
	EnvGitHubAppPrivateKey    = "SUPERPLANE_GITHUB_APP_PRIVATE_KEY"
	EnvGitHubAppWebhookSecret = "SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET"
)

// GitHubHostedAppConfig is SuperPlane Cloud's public GitHub App. The process
// holds the credentials. New connections store only the GitHub installation id.
type GitHubHostedAppConfig struct {
	ID            int64
	Slug          string
	PrivateKey    string
	WebhookSecret string
}

// LoadGitHubHostedAppConfig reads the public GitHub App from the process
// environment. Self-hosted leaves these empty. Enabled() is false unless
// every required value is set.
func LoadGitHubHostedAppConfig() GitHubHostedAppConfig {
	idRaw := strings.TrimSpace(os.Getenv(EnvGitHubAppID))
	slug := strings.TrimSpace(os.Getenv(EnvGitHubAppSlug))
	privateKey := normalizePEM(os.Getenv(EnvGitHubAppPrivateKey))
	webhookSecret := strings.TrimSpace(os.Getenv(EnvGitHubAppWebhookSecret))
	if idRaw == "" || slug == "" || privateKey == "" || webhookSecret == "" {
		return GitHubHostedAppConfig{}
	}

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || id <= 0 {
		return GitHubHostedAppConfig{}
	}

	return GitHubHostedAppConfig{
		ID:            id,
		Slug:          slug,
		PrivateKey:    privateKey,
		WebhookSecret: webhookSecret,
	}
}

// Enabled reports whether Cloud holds a complete public GitHub App.
func (c GitHubHostedAppConfig) Enabled() bool {
	return c.ID > 0 && c.Slug != "" && c.PrivateKey != "" && c.WebhookSecret != ""
}

func normalizePEM(value string) string {
	value = strings.TrimSpace(value)
	return strings.ReplaceAll(value, `\n`, "\n")
}
