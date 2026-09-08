package seed

import (
	"fmt"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	defaultEmail         = "dev@localhost"
	defaultPassword      = "superplane"
	defaultName          = "Dev Owner"
	defaultOrganization  = "Demo"
	defaultWorkspace     = "Dev"
	defaultGitHubName    = "github"
	defaultClaudeName    = "claude"
	defaultLineName      = "plan-and-implement"
	defaultAgentModel    = "sonnet"
	defaultPlanningModel = "opus"
	defaultBranchName    = "main"
)

// Config holds local seed inputs from the process environment.
type Config struct {
	Email             string
	Password          string
	Name              string
	OrganizationName  string
	WorkspaceName     string
	GitHubApp         common.HostedApp
	InstallationID    string
	AppRepository     string
	BacklogRepository string
	DefaultBranch     string
	AnthropicAPIKey   string
	BaseURL           string
	WebhooksBaseURL   string
}

// LoadConfig reads seed settings from the process environment.
func LoadConfig() Config {
	app, _ := common.HostedAppFromEnv()
	baseURL := strings.TrimSpace(os.Getenv("BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	webhooksBaseURL := strings.TrimSpace(os.Getenv("WEBHOOKS_BASE_URL"))
	if webhooksBaseURL == "" {
		webhooksBaseURL = baseURL
	}

	return Config{
		Email:             envOrDefault("SEED_EMAIL", defaultEmail),
		Password:          envOrDefault("SEED_PASSWORD", defaultPassword),
		Name:              envOrDefault("SEED_NAME", defaultName),
		OrganizationName:  envOrDefault("SEED_ORG", defaultOrganization),
		WorkspaceName:     envOrDefault("SEED_WORKSPACE", defaultWorkspace),
		GitHubApp:         app,
		InstallationID:    strings.TrimSpace(os.Getenv("SUPERPLANE_SEED_GITHUB_INSTALLATION_ID")),
		AppRepository:     strings.TrimSpace(os.Getenv("SUPERPLANE_SEED_GITHUB_REPO")),
		BacklogRepository: strings.TrimSpace(os.Getenv("SUPERPLANE_SEED_GITHUB_BACKLOG_REPO")),
		DefaultBranch:     envOrDefault("SUPERPLANE_SEED_GITHUB_DEFAULT_BRANCH", defaultBranchName),
		AnthropicAPIKey:   strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		BaseURL:           strings.TrimRight(baseURL, "/"),
		WebhooksBaseURL:   strings.TrimRight(webhooksBaseURL, "/"),
	}
}

// Validate reports missing GitHub App credentials or a missing Claude key.
func (c Config) Validate() error {
	if !c.GitHubAppConfigured() {
		return fmt.Errorf(
			"hosted GitHub App is not configured: set %s, %s, %s, and %s",
			config.EnvGitHubAppID,
			config.EnvGitHubAppSlug,
			config.EnvGitHubAppPrivateKey,
			config.EnvGitHubAppWebhookSecret,
		)
	}
	if c.AnthropicAPIKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is required")
	}
	if strings.TrimSpace(c.Email) == "" {
		return fmt.Errorf("SEED_EMAIL is required")
	}
	if strings.TrimSpace(c.Password) == "" {
		return fmt.Errorf("SEED_PASSWORD is required")
	}
	return nil
}

// GitHubAppConfigured reports whether the process holds a complete hosted App.
func (c Config) GitHubAppConfigured() bool {
	return c.GitHubApp.ID > 0 && c.GitHubApp.Slug != "" && c.GitHubApp.PrivateKey != "" && c.GitHubApp.WebhookSecret != ""
}

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}
