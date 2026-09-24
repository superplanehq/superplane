package sentry

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
)

const (
	configPrivateApp = "privateApp"

	SecretAccessToken  = "accessToken"
	SecretRefreshToken = "refreshToken"
)

// HostedApp is SuperPlane Cloud's public Sentry app. The process holds the
// credentials. New integrations store the installation UUID and tokens.
type HostedApp struct {
	Slug         string
	ClientID     string
	ClientSecret string
}

// HostedAppFromEnv returns the public Sentry app when Cloud holds complete
// credentials. Self-hosted leaves them empty.
func HostedAppFromEnv() (HostedApp, bool) {
	cfg := config.LoadSentryHostedAppConfig()
	if !cfg.Enabled() {
		return HostedApp{}, false
	}

	return HostedApp{
		Slug:         cfg.Slug,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
	}, true
}

func HostedAppConfigured() bool {
	return config.LoadSentryHostedAppConfig().Enabled()
}

func UseHostedApp() bool {
	return HostedAppConfigured()
}

// PreferHostedInstall is the default Sentry create path when Cloud holds the
// public app. A privateApp configuration flag opts out and uses a personal
// token plus an internal integration.
func PreferHostedInstall(integrationName string, configuration map[string]any) bool {
	return integrationName == "sentry" && UseHostedApp() && !WantsPrivateApp(integrationName, configuration)
}

func WantsPrivateApp(integrationName string, configuration map[string]any) bool {
	if integrationName != "sentry" || configuration == nil {
		return false
	}
	value, ok := configuration[configPrivateApp].(bool)
	return ok && value
}

func HostedAppExternalInstallURL(slug string) string {
	return fmt.Sprintf("https://sentry.io/sentry-apps/%s/external-install/", url.PathEscape(slug))
}

func HostedAppInstallURL(baseURL, state string) string {
	return strings.TrimRight(baseURL, "/") + "/api/v1/sentry/app/install?state=" + url.QueryEscape(state)
}

func HostedAppSetupURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/api/v1/sentry/app/setup"
}

func HostedAppWebhookURL(webhooksBaseURL string) string {
	return strings.TrimRight(webhooksBaseURL, "/") + "/api/v1/sentry/app/webhook"
}
