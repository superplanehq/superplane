package githubapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	publicAppHomepageURL     = "https://superplane.com"
	githubAppCreateURL       = "https://github.com/settings/apps/new"
	githubManifestConvertFmt = "https://api.github.com/app-manifests/%s/conversions"
	createdPath              = "/api/v1/github/app/created"
	setupPath                = "/api/v1/github/app/setup"
	webhookPath              = "/api/v1/github/app/webhook"
)

// CreateForm is the GitHub App Manifest POST the browser sends to GitHub.
type CreateForm struct {
	URL    string
	Method string
	Form   map[string]string
}

type convertedApp struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	WebhookSecret string `json:"webhook_secret"`
	PEM           string `json:"pem"`
}

func CreateURL() string {
	return githubAppCreateURL
}

func OAuthCallbackURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/auth/github/callback"
}

func CreatedPath() string {
	return createdPath
}

func PublicManifestJSON(baseURL, webhooksBaseURL string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	webhooksBaseURL = strings.TrimRight(strings.TrimSpace(webhooksBaseURL), "/")
	if baseURL == "" || webhooksBaseURL == "" {
		return "", fmt.Errorf("base URL and webhooks base URL are required")
	}

	manifest := map[string]any{
		"name":   "SuperPlane",
		"public": false,
		"url":    publicAppHomepageURL,
		"default_permissions": map[string]string{
			"issues":                      "write",
			"actions":                     "write",
			"checks":                      "read",
			"contents":                    "write",
			"pull_requests":               "write",
			"repository_hooks":            "write",
			"statuses":                    "write",
			"deployments":                 "write",
			"organization_administration": "read",
			"members":                     "read",
		},
		"default_events": []string{"member"},
		"setup_url":      baseURL + setupPath,
		"redirect_url":   baseURL + createdPath,
		"callback_urls":  []string{OAuthCallbackURL(baseURL)},
		"hook_attributes": map[string]any{
			"url": webhooksBaseURL + webhookPath,
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode GitHub App manifest: %w", err)
	}
	return string(data), nil
}

func ConvertManifest(httpCtx core.HTTPContext, code string) (config.GitHubHostedAppConfig, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("GitHub App manifest code is required")
	}
	if httpCtx == nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("HTTP client is required")
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(githubManifestConvertFmt, url.PathEscape(code)), nil)
	if err != nil {
		return config.GitHubHostedAppConfig{}, err
	}
	response, err := httpCtx.Do(req)
	if err != nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("convert GitHub App manifest: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("read GitHub App manifest conversion: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("convert GitHub App manifest: GitHub returned %d", response.StatusCode)
	}

	var app convertedApp
	if err := json.Unmarshal(body, &app); err != nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("decode GitHub App manifest conversion: %w", err)
	}

	cfg := config.GitHubHostedAppConfig{
		ID:            app.ID,
		Slug:          strings.TrimSpace(app.Slug),
		PrivateKey:    strings.TrimSpace(app.PEM),
		WebhookSecret: strings.TrimSpace(app.WebhookSecret),
		ClientID:      strings.TrimSpace(app.ClientID),
		ClientSecret:  strings.TrimSpace(app.ClientSecret),
	}
	if !cfg.Enabled() || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("GitHub App manifest conversion is incomplete")
	}
	return cfg, nil
}
