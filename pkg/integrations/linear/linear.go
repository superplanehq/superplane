package linear

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

const (
	OAuthAccessToken  = "accessToken"
	OAuthRefreshToken = "refreshToken"

	// scopeWithAdmin is requested when SuperPlane creates webhooks through the
	// API. Linear allows webhookCreate only for a workspace admin or a token
	// with the admin scope. scopeReadWrite is enough when the Linear OAuth
	// application sends events to SuperPlane's webhook URL.
	scopeWithAdmin = "read,write,admin"
	scopeReadWrite = "read,write"

	// WebhookSecretConfig is the signing secret of a customer Linear OAuth application.
	WebhookSecretConfig = "webhookSecret"

	// tokenRefreshMargin is how much of the access token lifetime is left when
	// it gets refreshed. Linear rotates refresh tokens, so only the most
	// recently issued one keeps working: refreshing on every sync lets two
	// overlapping runs - a scheduled resync racing a manual sync - spend the
	// same refresh token, and the loser would tear down credentials that the
	// winner just renewed. Refreshing only near expiration keeps them apart.
	tokenRefreshMargin = 2 * time.Hour
)

const (
	appSetupDescription = `
- Click **Continue** to open Linear's OAuth application form prefilled with the required information.
- Once on Linear, click **Create** at the bottom of the form which will take you to the credentials page.
- Copy the **Client ID** and **Client Secret** into the fields below and click **Save**.
`

	appConnectDescription = `Click **Continue** to authorize SuperPlane to access your Linear workspace.`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("linear", &Linear{}, &LinearWebhookHandler{})
}

type Linear struct{}

type Metadata struct {
	State                *string `json:"state,omitempty" mapstructure:"state,omitempty"`
	User                 *User   `json:"user,omitempty" mapstructure:"user,omitempty"`
	Teams                []Team  `json:"teams" mapstructure:"teams"`
	Organization         string  `json:"organization,omitempty" mapstructure:"organization,omitempty"`
	OrganizationID       string  `json:"organizationId,omitempty" mapstructure:"organizationId,omitempty"`
	URLKey               string  `json:"urlKey,omitempty" mapstructure:"urlKey,omitempty"`
	AccessTokenExpiresAt string  `json:"accessTokenExpiresAt,omitempty" mapstructure:"accessTokenExpiresAt,omitempty"`
	// HostedOAuth is true when this connection uses SuperPlane's Linear OAuth app.
	HostedOAuth bool `json:"hostedOAuth,omitempty" mapstructure:"hostedOAuth,omitempty"`
	// SetupReturnPath is a same-origin path to open after OAuth, such as the intake wizard.
	SetupReturnPath string `json:"setupReturnPath,omitempty" mapstructure:"setupReturnPath,omitempty"`
}

const installationInstructions = `
SuperPlane connects to Linear with OAuth. The person who authorizes the connection must be a member of each private team you want to listen to.

1. Open Linear, then **Settings**, then **Administration**, then **API**.
2. Under **OAuth applications**, click **Create new**.
3. Set the callback URL to the address SuperPlane shows.
4. Set the webhook URL to the SuperPlane address plus ` + "`/api/v1/linear/webhook`" + `.
5. Select **Issues**, **Comments**, and **Issue attachments**.
6. Create the application.
7. Copy the **Client ID** into the **Client ID** field.
8. Copy the **Client Secret** into the **Client Secret** field.
9. Copy the **Webhook signing secret** into the **Webhook signing secret** field.
10. Click **Save**. SuperPlane opens Linear so you can authorize the connection.

You can also leave the fields empty and click **Save**. SuperPlane opens Linear's application form with the callback URL and the webhook URL filled in. Copy the **Client ID**, the **Client Secret**, and the **Webhook signing secret** into the fields above, then click **Save** again.

When the webhook signing secret is set, the connection requests the **read** and **write** scopes. Write covers creating and editing issues, comments, attachments, and reactions. When the secret is empty, the connection also requests the **admin** scope so SuperPlane can create a webhook for each trigger. A workspace admin must authorize that connection.

Linear lets only a comment's own author edit it, so **Update Issue Comment** can only change comments that this same connection posted.
`

const hostedInstallationInstructions = `
Click **Connect** to authorize SuperPlane in your Linear workspace. The person who authorizes the connection must be a member of each private team you want to listen to.

The connection requests the **read**, **write**, and **admin** scopes. Write covers creating and editing issues. Admin lets SuperPlane register webhooks.

To use your own OAuth application instead, open the manual setup and paste its **Client ID** and **Client Secret**.
`

const hostedAppWebhookInstructions = `
Click **Connect** to authorize SuperPlane in your Linear workspace. The person who authorizes the connection must be a member of each private team you want to listen to.

The connection requests the **read** and **write** scopes. Write covers creating and editing issues.

On the Linear OAuth application, set the webhook URL to the SuperPlane address plus ` + "`/api/v1/linear/webhook`" + `. Select **Issues**, **Comments**, and **Issue attachments**. Copy the webhook signing secret into ` + "`SUPERPLANE_LINEAR_OAUTH_WEBHOOK_SECRET`" + `.

To use your own OAuth application instead, open the manual setup and paste its **Client ID**, **Client Secret**, and **Webhook signing secret**.
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
	if UseHostedOAuth() {
		if HostedWebhookSecret() != "" {
			return hostedAppWebhookInstructions
		}
		return hostedInstallationInstructions
	}
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
		{
			Name:        WebhookSecretConfig,
			Label:       "Webhook signing secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Signing secret from the Linear application webhook. Leave this empty to create webhooks with the admin scope.",
		},
	}
}

func (l *Linear) Actions() []core.Action {
	return []core.Action{
		&CreateIssue{},
		&GetIssue{},
		&UpdateIssue{},
		&AddIssueLabel{},
		&AddIssueComment{},
		&UpdateIssueComment{},
		&CreateAttachment{},
		&DeleteAttachment{},
		&RemoveIssueLabel{},
		&AddReaction{},
	}
}

func (l *Linear) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
		&OnIssueComment{},
		&OnIssueLabel{},
		&OnIssueAttachment{},
	}
}

func (l *Linear) Sync(ctx core.SyncContext) error {
	app := resolveOAuthApp(ctx.Integration)
	callbackURL := oauthCallbackURL(ctx.BaseURL, ctx.Integration.ID(), app.Hosted)

	//
	// Without app credentials, guide the user through creating the OAuth app.
	// Linear has no API for this, but its creation form accepts manifest query
	// parameters, so the form opens fully pre-filled. A hosted app skips this
	// step and authorizes SuperPlane's application.
	//
	if !app.configured() {
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
		return l.requestAuthorization(ctx, app, callbackURL)
	}

	//
	// Linear access tokens expire after 24 hours,
	// so refresh on every scheduled resync.
	//
	if err := l.refreshToken(ctx, app.ClientID, app.ClientSecret); err != nil {
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
	params.Set("webhook.enabled", "true")
	params.Set("webhook.url", AppWebhookURL(baseURL))
	for _, resourceType := range appWebhookResourceTypes {
		params.Add("webhook.resourceTypes", resourceType)
	}

	return fmt.Sprintf("%s?%s", AppsNewURL, params.Encode())
}

func (l *Linear) requestAuthorization(ctx core.SyncContext, app oauthApp, callbackURL string) error {
	metadata := readMetadata(ctx.Integration)
	rememberSetupReturnPath(ctx, &metadata)

	if metadata.State == nil {
		state, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		metadata.State = &state
	}
	metadata.HostedOAuth = app.Hosted
	ctx.Integration.SetMetadata(metadata)

	clientID := app.ClientID

	authorizeURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&prompt=consent&actor=user",
		AuthorizeURL,
		url.QueryEscape(clientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape(OAuthScopes(ctx.Integration)),
		url.QueryEscape(*metadata.State),
	)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appConnectDescription,
		URL:         authorizeURL,
		Method:      "GET",
	})

	return nil
}

// refreshToken exchanges the stored refresh token for a fresh token pair once
// the access token is close to expiring. Linear rotates refresh tokens, so both
// secrets are replaced on success, and cleared only when the access token is
// already unusable, to route the user back to the authorize step.
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

	//
	// Connections authorized before expirations were recorded have no
	// expiration stored, so they refresh on the next sync and record one.
	//
	remaining, known := accessTokenValidity(ctx.Integration)
	if known && remaining > tokenRefreshMargin {
		ctx.Logger.Info("Linear access token is still valid, skipping refresh")
		return ctx.Integration.ScheduleResync(remaining - tokenRefreshMargin)
	}

	ctx.Logger.Info("Refreshing Linear token")
	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(clientID, clientSecret, refreshToken)
	if err != nil {
		//
		// A refresh also fails when a concurrent sync already rotated the token,
		// and then the stored pair is the fresh one and has to survive. Keep the
		// credentials while the access token still works and retry before it
		// expires, and only give up on them once it cannot be used at all.
		//
		if known && remaining > 0 {
			ctx.Logger.Errorf("Failed to refresh token, retrying before expiration: %v", err)
			return ctx.Integration.ScheduleResync(refreshRetryInterval(remaining))
		}

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

// accessTokenValidity reports how long the stored access token remains usable.
// The second return is false when no expiration was recorded.
func accessTokenValidity(integration core.IntegrationContext) (time.Duration, bool) {
	metadata := readMetadata(integration)
	if metadata.AccessTokenExpiresAt == "" {
		return 0, false
	}

	expiresAt, err := time.Parse(time.RFC3339, metadata.AccessTokenExpiresAt)
	if err != nil {
		return 0, false
	}

	return time.Until(expiresAt), true
}

// refreshRetryInterval spreads a few retries over what is left of the access
// token lifetime, so a transient token endpoint failure recovers on its own.
func refreshRetryInterval(remaining time.Duration) time.Duration {
	return max(remaining/4, time.Minute)
}

func readMetadata(integration core.IntegrationContext) Metadata {
	metadata := Metadata{}
	_ = mapstructure.Decode(integration.GetMetadata(), &metadata)
	return metadata
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

	//
	// The expiration decides when the next sync refreshes, so it is recorded
	// next to the tokens rather than derived from when the sync last ran.
	//
	metadata := readMetadata(integration)
	metadata.AccessTokenExpiresAt = ""
	if expiresAt := tokenResponse.ExpiresAt(); !expiresAt.IsZero() {
		metadata.AccessTokenExpiresAt = expiresAt.Format(time.RFC3339)
	}
	integration.SetMetadata(metadata)

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

	//
	// The recorded token expiration carries over, since it drives when the next
	// sync refreshes and is unrelated to the workspace data loaded here.
	//
	previous := readMetadata(ctx.Integration)
	organizationID := viewer.Organization.ID
	if organizationID == "" {
		organizationID = previous.OrganizationID
	}
	ctx.Integration.SetMetadata(Metadata{
		User:                 viewer.User,
		Teams:                teams,
		Organization:         viewer.Organization.Name,
		OrganizationID:       organizationID,
		URLKey:               viewer.Organization.URLKey,
		AccessTokenExpiresAt: previous.AccessTokenExpiresAt,
		HostedOAuth:          previous.HostedOAuth,
		SetupReturnPath:      previous.SetupReturnPath,
	})

	return nil
}

func (l *Linear) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	metadata := readMetadata(ctx.Integration)
	app := resolveOAuthApp(ctx.Integration)
	if metadata.HostedOAuth {
		app.Hosted = true
	}
	if !app.configured() {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	expectedState := ""
	if metadata.State != nil {
		expectedState = *metadata.State
	}

	settingsURL := fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID())
	redirectURL := callbackRedirectURL(ctx, settingsURL)
	redirectURI := oauthCallbackURL(ctx.BaseURL, ctx.Integration.ID(), app.Hosted && metadata.HostedOAuth)

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.HandleCallback(ctx.Request, app.ClientID, app.ClientSecret, expectedState, redirectURI)
	if err != nil {
		ctx.Logger.Errorf("Callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request, redirectURL, http.StatusSeeOther)
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
		http.Redirect(ctx.Response, ctx.Request, redirectURL, http.StatusSeeOther)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request, redirectURL, http.StatusSeeOther)
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
