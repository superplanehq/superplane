package jira

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	setupReturnPathKey             = "setupReturnPath"
	setupReturnedIntegrationParam  = "jiraIntegrationId"
)

// oauthApp is the Atlassian 3LO app SuperPlane uses for this connection.
type oauthApp struct {
	ClientID     string
	ClientSecret string
	Hosted       bool
}

// UseHostedOAuth reports whether the process holds SuperPlane's Jira OAuth app.
func UseHostedOAuth() bool {
	return config.LoadJiraHostedOAuthConfig().Enabled()
}

// UseHostedInstall is true when this integration create should skip Client ID
// and Client Secret and authorize SuperPlane's Atlassian OAuth app.
func UseHostedInstall(integrationName string) bool {
	return integrationName == "jira" && UseHostedOAuth()
}

// HostedOAuthCallbackURL is the stable Atlassian callback for the hosted app.
func HostedOAuthCallbackURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/api/v1/jira/oauth/callback"
}

func resolveOAuthApp(integration core.IntegrationContext) oauthApp {
	if integration != nil {
		clientID, _ := integration.GetConfig("clientId")
		clientSecret, _ := integration.GetConfig("clientSecret")
		if len(clientID) > 0 && len(clientSecret) > 0 {
			return oauthApp{
				ClientID:     string(clientID),
				ClientSecret: string(clientSecret),
			}
		}
	}

	cfg := config.LoadJiraHostedOAuthConfig()
	if !cfg.Enabled() {
		return oauthApp{}
	}

	return oauthApp{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Hosted:       true,
	}
}

func oauthCallbackURL(baseURL string, integrationID fmt.Stringer, hosted bool) string {
	if hosted {
		return HostedOAuthCallbackURL(baseURL)
	}

	return fmt.Sprintf("%s/api/v1/integrations/%s/callback", strings.TrimRight(baseURL, "/"), integrationID)
}

func rememberSetupReturnPath(ctx core.SyncContext, metadata *Metadata) {
	path := firstSafeSetupReturnPath(setupReturnPathFromConfig(ctx.Configuration), metadata.SetupReturnPath)
	metadata.SetupReturnPath = path
}

func setupReturnPathFromConfig(configuration any) string {
	configMap, ok := configuration.(map[string]any)
	if !ok {
		return ""
	}

	path, _ := configMap[setupReturnPathKey].(string)
	return firstSafeSetupReturnPath(path)
}

func callbackRedirectURL(ctx core.HTTPRequestContext, settingsURL string) string {
	metadata := readMetadata(ctx.Integration)
	path := firstSafeSetupReturnPath(metadata.SetupReturnPath)
	if path == "" {
		return settingsURL
	}

	return strings.TrimRight(ctx.BaseURL, "/") + withReturnedIntegrationID(path, ctx.Integration.ID().String())
}

func withReturnedIntegrationID(path, integrationID string) string {
	if integrationID == "" {
		return path
	}

	pathname, existing, _ := strings.Cut(path, "?")
	params, err := url.ParseQuery(existing)
	if err != nil {
		params = url.Values{}
	}
	params.Set(setupReturnedIntegrationParam, integrationID)
	encoded := params.Encode()
	if encoded == "" {
		return pathname
	}
	return pathname + "?" + encoded
}

func firstSafeSetupReturnPath(paths ...string) string {
	for _, path := range paths {
		if isSafeIntegrationSetupReturnPath(path) {
			return path
		}
	}

	return ""
}

func isSafeIntegrationSetupReturnPath(path string) bool {
	if path == "" || strings.Contains(path, "://") || strings.ContainsAny(path, "\\\t\r\n ") {
		return false
	}

	pathname, _, _ := strings.Cut(path, "?")
	if pathname == "/onboarding" {
		return true
	}
	if !strings.HasPrefix(pathname, "/") || strings.HasPrefix(pathname, "//") {
		return false
	}

	rest := strings.TrimPrefix(pathname, "/")
	organization, after, ok := strings.Cut(rest, "/")
	return ok && organization != "" && after != ""
}
