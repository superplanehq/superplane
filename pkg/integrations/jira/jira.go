package jira

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("jira", &Jira{}, &JiraWebhookHandler{})
}

type Jira struct{}

type Configuration struct {
	ClientID     string `json:"clientId" mapstructure:"clientId"`
	ClientSecret string `json:"clientSecret" mapstructure:"clientSecret"`
}

type Metadata struct {
	AuthType string    `json:"authType,omitempty" mapstructure:"authType"`
	State    *string   `json:"state,omitempty" mapstructure:"state"`
	CloudID  string    `json:"cloudId,omitempty" mapstructure:"cloudId"`
	BaseURL  string    `json:"baseUrl,omitempty" mapstructure:"baseUrl"`
	SiteName string    `json:"siteName,omitempty" mapstructure:"siteName"`
	User     *User     `json:"user,omitempty" mapstructure:"user"`
	Projects []Project `json:"projects" mapstructure:"projects"`
}

var oauthScopes = []string{
	"read:jira-work",
	"write:jira-work",
	"manage:jira-webhook",
	"offline_access",
}

const (
	atlassianDeveloperConsoleURL = "https://developer.atlassian.com/console/myapps/"

	oauthSetupDescription = `
Use this **Callback URL** when configuring OAuth 2.0 (3LO) in your Atlassian app:

` + "`%s`" + `

Required scopes:
` + "`%s`" + `

Click **Continue** to open the [Atlassian Developer Console](` + atlassianDeveloperConsoleURL + `), create the OAuth app, then copy its **Client ID** and **Client Secret** into SuperPlane.
`

	oauthConnectDescription = "Authorize SuperPlane to access Jira and create the issue webhook."
)

func (j *Jira) Name() string {
	return "jira"
}

func (j *Jira) Label() string {
	return "Jira"
}

func (j *Jira) Icon() string {
	return "jira"
}

func (j *Jira) Description() string {
	return "Manage and react to issues in Jira"
}

func (j *Jira) Instructions() string {
	return `
**Setup steps:**
1. Click **Connect** once with **Client ID** and **Client Secret** empty. The same setup box at the top of this modal will change to show a **Callback URL**. If you close the modal, you can also see it on the Jira integration details page in the yellow setup box.

2. Open the [Atlassian Developer Console](` + atlassianDeveloperConsoleURL + `), then select **Create app → OAuth 2.0 integration**.

   > **Required scopes:**  
   > ` + "`read:jira-work`" + ` · ` + "`write:jira-work`" + ` · ` + "`manage:jira-webhook`" + ` · ` + "`offline_access`" + `

3. In the Atlassian app, go to **Authorization → OAuth 2.0 (3LO)** and add the callback URL shown by SuperPlane.
4. Copy the Atlassian app **Client ID** and **Client Secret** into the fields below, then save.
5. Click **Continue** to authorize Jira. SuperPlane creates and manages the Jira issue webhook automatically.
`
}

func (j *Jira) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Description: "Client ID from your Atlassian OAuth 2.0 (3LO) app. This is not your Atlassian email address.",
			Placeholder: "Atlassian OAuth app Client ID",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Client secret from your Atlassian OAuth 2.0 (3LO) app.",
			Placeholder: "Atlassian OAuth app Client Secret",
		},
	}
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{}
}

func (j *Jira) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (j *Jira) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jira) Sync(ctx core.SyncContext) error {
	return j.oauthSync(ctx, loadConfiguration(ctx.Integration))
}

func (j *Jira) oauthSync(ctx core.SyncContext, config Configuration) error {
	// Atlassian rejects http://localhost callbacks; the WebhooksBaseURL (e.g. an
	// ngrok HTTPS tunnel) is the externally reachable origin. It falls back to
	// BaseURL when WEBHOOKS_BASE_URL is not set, so production stays unchanged.
	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", externalBaseURL(ctx.WebhooksBaseURL, ctx.BaseURL), ctx.Integration.ID())

	if config.ClientID == "" || config.ClientSecret == "" {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: fmt.Sprintf(oauthSetupDescription, callbackURL, strings.Join(oauthScopes, " ")),
			URL:         "https://developer.atlassian.com/console/myapps/",
			Method:      http.MethodGet,
		})
		return nil
	}

	if err := validateOAuthConfiguration(config); err != nil {
		return err
	}

	accessToken, _ := findSecret(ctx.Integration, OAuthAccessToken)
	if accessToken == "" {
		return j.handleOAuthNoAccessToken(ctx, callbackURL, config.ClientID)
	}

	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshToken)
	if refreshToken != "" {
		if err := j.refreshOAuthToken(ctx, config.ClientID, config.ClientSecret, refreshToken); err != nil {
			ctx.Logger.Errorf("failed to refresh Jira OAuth token: %v", err)
			return err
		}
	}

	if err := j.updateOAuthMetadata(ctx); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	// Best-effort: extend the 30-day expiry on every webhook this app owns.
	// Failures here are logged but do not fail the sync.
	j.refreshWebhooks(ctx)

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func (j *Jira) refreshWebhooks(ctx core.SyncContext) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("skipping Jira webhook refresh: %v", err)
		}
		return
	}

	webhooks, err := client.ListWebhooks()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to list Jira webhooks for refresh: %v", err)
		}
		return
	}

	if len(webhooks) == 0 {
		return
	}

	ids := make([]int64, 0, len(webhooks))
	for _, w := range webhooks {
		ids = append(ids, w.ID)
	}

	// Jira's /webhook/refresh accepts a max of 100 IDs per call.
	const refreshBatchSize = 100
	for start := 0; start < len(ids); start += refreshBatchSize {
		end := start + refreshBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		if _, err := client.RefreshWebhooks(ids[start:end]); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("failed to refresh Jira webhooks (batch %d-%d): %v", start, end, err)
		}
	}
}

// externalBaseURL returns the externally reachable origin for OAuth callbacks.
// Falls back to baseURL when webhooksBaseURL is empty (e.g. WEBHOOKS_BASE_URL not set).
func externalBaseURL(webhooksBaseURL, baseURL string) string {
	if strings.TrimSpace(webhooksBaseURL) != "" {
		return webhooksBaseURL
	}

	return baseURL
}

func validateOAuthConfiguration(config Configuration) error {
	if strings.Contains(config.ClientID, "@") {
		return fmt.Errorf("clientId must be the Atlassian OAuth app Client ID, not an email address")
	}

	return nil
}

func (j *Jira) handleOAuthNoAccessToken(ctx core.SyncContext, callbackURL, clientID string) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("failed to decode Jira metadata while setting OAuth state: %v", err)
	}

	if metadata.State == nil {
		state, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate OAuth state: %v", err)
		}

		metadata.State = &state
		metadata.AuthType = AuthTypeOAuth
		ctx.Integration.SetMetadata(metadata)
	}

	authURL := jiraOAuthURL(clientID, callbackURL, *metadata.State)
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: oauthConnectDescription,
		URL:         authURL,
		Method:      http.MethodGet,
	})

	return nil
}

func (j *Jira) refreshOAuthToken(ctx core.SyncContext, clientID, clientSecret, refreshToken string) error {
	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(clientID, clientSecret, refreshToken)
	if err != nil {
		_ = ctx.Integration.SetSecret(OAuthRefreshToken, []byte(""))
		_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(""))
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	if tokenResponse.AccessToken != "" {
		if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
			return fmt.Errorf("failed to save access token: %v", err)
		}
	}

	if tokenResponse.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			return fmt.Errorf("failed to save refresh token: %v", err)
		}
	}

	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

func (j *Jira) updateOAuthMetadata(ctx core.SyncContext) error {
	accessToken, err := requireOAuthSecret(ctx.Integration, OAuthAccessToken)
	if err != nil {
		return err
	}

	auth := NewAuth(ctx.HTTP)
	resources, err := auth.AccessibleResources(accessToken)
	if err != nil {
		return err
	}

	resource, err := firstJiraResource(resources)
	if err != nil {
		return err
	}

	client := NewOAuthClient(ctx.HTTP, accessToken, resource.ID)
	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying Jira OAuth credentials: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("error listing Jira projects: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		AuthType: AuthTypeOAuth,
		CloudID:  resource.ID,
		BaseURL:  resource.URL,
		SiteName: resource.Name,
		User:     user,
		Projects: projects,
	})

	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	config := loadConfiguration(ctx.Integration)
	if config.ClientID == "" || config.ClientSecret == "" {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	j.handleCallback(ctx, config)
}

func (j *Jira) handleCallback(ctx core.HTTPRequestContext, config Configuration) {
	// The redirect_uri sent to Atlassian during token exchange must match the
	// one used at /authorize time exactly — see oauthSync for the same rule.
	externalURL := externalBaseURL(ctx.WebhooksBaseURL, ctx.BaseURL)
	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", externalURL, ctx.Integration.ID())
	// The post-OAuth redirect back to the SuperPlane UI uses BaseURL since
	// that's where the user's browser session lives.
	redirectBaseURL := ctx.BaseURL

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if metadata.State == nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.HandleCallback(ctx.Request, config, *metadata.State, callbackURL)
	if err != nil {
		ctx.Logger.Errorf("Jira OAuth callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request, j.integrationSettingsURL(redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()), http.StatusSeeOther)
		return
	}

	if tokenResponse.AccessToken != "" {
		if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
			ctx.Response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if tokenResponse.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			ctx.Response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Schedule a near-immediate Sync so metadata population + verification
	// happens with full SyncContext (Logger, error state transitions). This
	// runs in the background; we still try the inline updateOAuthMetadata
	// below as a best-effort fast-path so the integration goes Ready before
	// the user lands on the settings page.
	if err := ctx.Integration.ScheduleResync(time.Second); err != nil {
		ctx.Logger.Errorf("Jira OAuth callback: failed to schedule resync: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Best-effort: try to populate metadata now so the user sees Ready right
	// away. If anything Atlassian-facing fails (transient network, token not
	// yet propagated, etc.), don't 500 — the scheduled Sync above will retry.
	if err := j.updateOAuthMetadata(core.SyncContext{
		Logger:      ctx.Logger,
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}); err != nil {
		ctx.Logger.Warnf("Jira OAuth callback: inline metadata update failed, deferring to Sync: %v", err)
	} else {
		ctx.Integration.RemoveBrowserAction()
		ctx.Integration.Ready()
	}

	http.Redirect(ctx.Response, ctx.Request, j.integrationSettingsURL(redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()), http.StatusSeeOther)
}

func (j *Jira) integrationSettingsURL(baseURL, organizationID, integrationID string) string {
	return fmt.Sprintf("%s/%s/settings/integrations/%s", baseURL, url.PathEscape(organizationID), url.PathEscape(integrationID))
}

func (t *TokenResponse) GetExpiration() time.Duration {
	if t.ExpiresIn > 0 {
		seconds := t.ExpiresIn / 2
		if seconds < 1 {
			seconds = 1
		}
		return time.Duration(seconds) * time.Second
	}

	return time.Hour
}

func (j *Jira) Hooks() []core.Hook {
	return []core.Hook{}
}

func (j *Jira) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
