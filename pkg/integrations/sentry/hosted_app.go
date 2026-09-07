package sentry

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	secretInstallationToken = "token"
	secretRefreshToken      = "refreshToken"
	secretTokenExpiresAt    = "tokenExpiresAt"

	hostedInstallDescription = `
Click **Continue** to open Sentry. Select the Sentry organization, then install SuperPlane.
`
)

// HostedApp is SuperPlane Cloud's public Sentry app. The process holds the
// credentials. New integrations store only the Sentry installation id.
type HostedApp struct {
	ClientID     string
	ClientSecret string
	Slug         string
	BaseURL      string
}

// HostedAppFromEnv returns the public Sentry app when Cloud holds complete
// credentials. Self-hosted leaves them empty.
func HostedAppFromEnv() (HostedApp, bool) {
	cfg := config.LoadSentryHostedAppConfig()
	if !cfg.Enabled() {
		return HostedApp{}, false
	}

	return HostedApp{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Slug:         cfg.Slug,
		BaseURL:      cfg.BaseURL,
	}, true
}

func HostedAppConfigured() bool {
	return config.LoadSentryHostedAppConfig().Enabled()
}

// UseHostedApp reports whether new Sentry connections for this organization
// should install SuperPlane's public Sentry app. The org must have
// factory_sentry_intake, and the process must hold the Cloud app credentials.
func UseHostedApp(orgID string) bool {
	if !HostedAppConfigured() {
		return false
	}
	return sentryIntakeEnabled(orgID)
}

// UseHostedInstall is true when this integration create should skip the
// configuration form and install the public Sentry app.
func UseHostedInstall(orgID, integrationName string) bool {
	return integrationName == "sentry" && UseHostedApp(orgID)
}

// PreferHostedInstall is the default Sentry create path when the public app
// is configured and factory_sentry_intake is on.
func PreferHostedInstall(orgID, integrationName string, _ map[string]any) bool {
	return UseHostedInstall(orgID, integrationName)
}

func ExternalInstallURL(baseURL, slug string) string {
	return fmt.Sprintf("%s/sentry-apps/%s/external-install/", strings.TrimRight(baseURL, "/"), url.PathEscape(slug))
}

func HostedAppStartURL(appBaseURL, integrationID string) string {
	return strings.TrimRight(appBaseURL, "/") + "/api/v1/sentry/app/start?integration=" + url.QueryEscape(integrationID)
}

func (m Metadata) AllowsStartedBy(userID string) bool {
	if m.StartedByUserID == "" {
		return true
	}

	return userID != "" && m.StartedByUserID == userID
}

var sentryIntakeEnabled = sentryIntakeEnabledForOrg

func sentryIntakeEnabledForOrg(orgID string) bool {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return false
	}

	enabled, err := models.HasExperimentalFeature(id, features.FeatureFactorySentryIntake)
	return err == nil && enabled
}

func withSentryIntakeEnabledForTest(fn func(string) bool) func() {
	previous := sentryIntakeEnabled
	sentryIntakeEnabled = fn
	return func() { sentryIntakeEnabled = previous }
}
