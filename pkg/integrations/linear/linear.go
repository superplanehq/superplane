package linear

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	OAuthAccessToken  = "accessToken"
	OAuthRefreshToken = "refreshToken"

	// scopeList is requested comma-separated on the authorize URL. The admin
	// scope is needed because the On Issue trigger manages webhooks, which
	// Linear restricts to workspace admins or OAuth tokens with admin scope.
	scopeList = "read,write,admin"
)

const (
	appSetupDescription = `
- Click **Continue** to open Linear's OAuth application form.
- Click **Create** at the bottom of the form.
- Copy the **Client ID** and **Client Secret** into the fields below and click **Save**.
`

	appConnectDescription = `Click **Continue** to authorize SuperPlane to access your Linear workspace.`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("linear", &Linear{}, &LinearWebhookHandler{})
}

type Linear struct{}

type Metadata struct {
	State        *string `json:"state,omitempty" mapstructure:"state,omitempty"`
	User         *User   `json:"user,omitempty" mapstructure:"user,omitempty"`
	Teams        []Team  `json:"teams" mapstructure:"teams"`
	Organization string  `json:"organization,omitempty" mapstructure:"organization,omitempty"`
	URLKey       string  `json:"urlKey,omitempty" mapstructure:"urlKey,omitempty"`
}

const installationInstructions = `
SuperPlane connects to Linear with OAuth.

1. Click **Connect** below to start the setup wizard. 
2. On the connection wizard that opens, click **Connect** which will open Linear's OAuth application form pre-filled with everything SuperPlane needs.
3. Click **Create** on Linear, then copy the **Client ID** and **Client Secret** in the input fields provided on SuperPlane.
4. Click **Save**, then once saved, click **Continue** on the integration configuration to authorize SuperPlane in your Linear workspace.

**Permissions:** SuperPlane requests the read, write and admin scopes. The admin scope is required because the On Issue trigger registers Linear webhooks, and Linear only allows webhook management with admin access — so connect as a **workspace admin**.

**Note:** actions performed by SuperPlane are attributed to the user who authorized the connection, and the integration stops working if that user leaves the workspace.
`

func (l *Linear) Name() string {
	return "linear"
}

func (l *Linear) Label() string {
	return "Linear"
}

func (l *Linear) Icon() string {
	return "linear"
}

func (l *Linear) Description() string {
	return "Manage and react to issues in Linear"
}

func (l *Linear) Instructions() string {
	return installationInstructions
}

func (l *Linear) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Description: "OAuth Client ID from your Linear application. Leave empty and click Save to start the setup wizard.",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "OAuth Client Secret from your Linear application",
		},
	}
}

func (l *Linear) Actions() []core.Action {
	return []core.Action{
		&CreateIssue{},
	}
}

func (l *Linear) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (l *Linear) Sync(ctx core.SyncContext) error {
	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	//
	// Sensitive configuration values are stored encrypted, and only
	// GetConfig decrypts them - never read the client secret from
	// ctx.Configuration directly.
	//
	clientID, _ := ctx.Integration.GetConfig("clientId")
	clientSecret, _ := ctx.Integration.GetConfig("clientSecret")

	//
	// Without app credentials, guide the user through creating the OAuth app.
	// Linear has no API for this, but its creation form accepts manifest query
	// parameters, so the form opens fully pre-filled.
	//
	if len(clientID) == 0 || len(clientSecret) == 0 {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appSetupDescription,
			URL:         appCreateURL(ctx.BaseURL, callbackURL),
			Method:      "GET",
		})

		return nil
	}

	//
	// With credentials but no access token, ask the user to authorize the app.
	//
	accessToken, _ := findSecret(ctx.Integration, OAuthAccessToken)
	if accessToken == "" {
		return l.requestAuthorization(ctx, string(clientID), callbackURL)
	}

	//
	// Linear access tokens expire after 24 hours,
	// so refresh on every scheduled resync.
	//
	if err := l.refreshToken(ctx, string(clientID), string(clientSecret)); err != nil {
		ctx.Logger.Errorf("Failed to refresh token: %v", err)
		return err
	}

	if err := l.updateMetadata(ctx); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

// appCreateURL pre-fills Linear's OAuth application form via manifest query
// parameters, so the user only clicks Create and copies the credentials.
func appCreateURL(baseURL, callbackURL string) string {
	params := url.Values{}
	params.Set("distribution", "private")
	params.Set("developer.name", "SuperPlane")
	params.Set("oauth.client_name", "SuperPlane")
	params.Set("oauth.client_uri", baseURL)
	params.Set("oauth.redirect_uris", callbackURL)

	return fmt.Sprintf("%s?%s", AppsNewURL, params.Encode())
}

func (l *Linear) requestAuthorization(ctx core.SyncContext, clientID, callbackURL string) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("Failed to decode metadata while setting state: %v", err)
	}

	if metadata.State == nil {
		state, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		metadata.State = &state
		ctx.Integration.SetMetadata(metadata)
	}

	authorizeURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&prompt=consent&actor=user",
		AuthorizeURL,
		url.QueryEscape(clientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape(scopeList),
		url.QueryEscape(*metadata.State),
	)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appConnectDescription,
		URL:         authorizeURL,
		Method:      "GET",
	})

	return nil
}

// refreshToken exchanges the stored refresh token for a fresh token pair.
// Linear rotates refresh tokens, so both secrets are replaced on success and
// cleared on failure to route the user back to the authorize step.
func (l *Linear) refreshToken(ctx core.SyncContext, clientID, clientSecret string) error {
	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshToken)
	if refreshToken == "" {
		//
		// Linear access tokens always expire within 24 hours, so an access token
		// without a refresh token is a dead end. Clear it so the next sync sends
		// the user back to the authorize step instead of reporting ready.
		//
		_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(""))
		return fmt.Errorf("no refresh token stored - re-authorize the integration")
	}

	ctx.Logger.Info("Refreshing Linear token")
	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(clientID, clientSecret, refreshToken)
	if err != nil {
		_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(""))
		_ = ctx.Integration.SetSecret(OAuthRefreshToken, []byte(""))
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	if err := storeTokens(ctx.Integration, tokenResponse); err != nil {
		return err
	}

	ctx.Logger.Info("Token refreshed successfully")
	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

func storeTokens(integration core.IntegrationContext, tokenResponse *TokenResponse) error {
	if tokenResponse.AccessToken != "" {
		if err := integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
			return fmt.Errorf("failed to save access token: %v", err)
		}
	}

	if tokenResponse.RefreshToken != "" {
		if err := integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			return fmt.Errorf("failed to save refresh token: %v", err)
		}
	}

	return nil
}

func (l *Linear) updateMetadata(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	viewer, err := client.GetViewer()
	if err != nil {
		return fmt.Errorf("error verifying Linear credentials: %v", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("error listing teams: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		User:         viewer.User,
		Teams:        teams,
		Organization: viewer.Organization.Name,
		URLKey:       viewer.Organization.URLKey,
	})

	return nil
}

func (l *Linear) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	clientID, err := ctx.Integration.GetConfig("clientId")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	expectedState := ""
	if metadata.State != nil {
		expectedState = *metadata.State
	}

	settingsURL := fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID())
	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.HandleCallback(ctx.Request, string(clientID), string(clientSecret), expectedState, redirectURI)
	if err != nil {
		ctx.Logger.Errorf("Callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	if err := storeTokens(ctx.Integration, tokenResponse); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := ctx.Integration.ScheduleResync(tokenResponse.GetExpiration()); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	//
	// The tokens are stored and a resync is scheduled at this point, so a
	// metadata failure must not strand the user on a bare error page: surface
	// it through the integration state and send them back to settings, where
	// a manual or scheduled sync retries with the saved tokens.
	//
	if err := l.updateMetadata(core.SyncContext{
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}); err != nil {
		ctx.Logger.Errorf("Callback error: failed to update metadata: %v", err)
		ctx.Integration.Error(fmt.Sprintf("connected, but failed to load workspace data: %v", err))
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
}

func findSecret(integration core.IntegrationContext, name string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value), nil
		}
	}

	return "", nil
}

func (l *Linear) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *Linear) Hooks() []core.Hook {
	return []core.Hook{}
}

func (l *Linear) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
