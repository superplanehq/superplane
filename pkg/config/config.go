package config

import (
	"fmt"
	"os"
	"strconv"
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
	CanvasStorageDriverDisabled    = "disabled"
	CanvasStorageDriverCodeStorage = "code_storage"
	CanvasStorageDriverLocalGit    = "local_git"
)

const (
	defaultCanvasStorageLocalRoot      = "/var/lib/superplane/canvas-repos"
	defaultCanvasStorageDefaultBranch  = "main"
	defaultCanvasStorageMaxFileBytes   = 10 * 1024 * 1024
	defaultCanvasStorageMaxCommitBytes = 25 * 1024 * 1024
)

// CanvasStorageConfig holds the configuration for Git-backed canvas files.
// The driver is disabled by default so existing deployments keep their current
// DB-only behavior until canvas file storage is explicitly enabled.
type CanvasStorageConfig struct {
	Driver                    string
	LocalRoot                 string
	DefaultBranch             string
	MaxFileBytes              int64
	MaxCommitBytes            int64
	CodeStorageName           string
	CodeStoragePrivateKey     string
	CodeStoragePrivateKeyPath string
	CodeStorageAPIBaseURL     string
	CodeStorageStorageBaseURL string
}

func LoadCanvasStorageConfig() CanvasStorageConfig {
	driver := strings.TrimSpace(os.Getenv("CANVAS_STORAGE_DRIVER"))
	if driver == "" {
		driver = CanvasStorageDriverDisabled
	}

	localRoot := strings.TrimSpace(os.Getenv("CANVAS_STORAGE_LOCAL_ROOT"))
	if localRoot == "" {
		localRoot = defaultCanvasStorageLocalRoot
	}

	defaultBranch := strings.TrimSpace(os.Getenv("CANVAS_STORAGE_DEFAULT_BRANCH"))
	if defaultBranch == "" {
		defaultBranch = defaultCanvasStorageDefaultBranch
	}

	return CanvasStorageConfig{
		Driver:                    driver,
		LocalRoot:                 localRoot,
		DefaultBranch:             defaultBranch,
		MaxFileBytes:              loadInt64Env("CANVAS_STORAGE_MAX_FILE_BYTES", defaultCanvasStorageMaxFileBytes),
		MaxCommitBytes:            loadInt64Env("CANVAS_STORAGE_MAX_COMMIT_BYTES", defaultCanvasStorageMaxCommitBytes),
		CodeStorageName:           strings.TrimSpace(os.Getenv("CODE_STORAGE_NAME")),
		CodeStoragePrivateKey:     strings.TrimSpace(os.Getenv("CODE_STORAGE_PRIVATE_KEY")),
		CodeStoragePrivateKeyPath: strings.TrimSpace(os.Getenv("CODE_STORAGE_PRIVATE_KEY_PATH")),
		CodeStorageAPIBaseURL:     strings.TrimSpace(os.Getenv("CODE_STORAGE_API_BASE_URL")),
		CodeStorageStorageBaseURL: strings.TrimSpace(os.Getenv("CODE_STORAGE_STORAGE_BASE_URL")),
	}
}

func loadInt64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
