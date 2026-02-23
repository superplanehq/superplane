package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

	linearScopes = "read,write,issues:create"

	appSetupDescription = `
- Click the **Continue** button to go to the Linear API settings page
- Click **Create new** under OAuth2 Applications:
  - **Application name**: SuperPlane
  - **Redirect callback URLs**: Use the Callback URL shown above
  - Enable **Webhooks** and set the webhook URL to the Webhook URL shown above
  - Under webhook events, check **Issues**
- Copy the **Client ID**, **Client Secret**, and **Webhook signing secret** and paste them in the fields below.
- Click **Save** to complete the setup.
`

	appConnectDescription = `Click **Continue** to authorize SuperPlane to access your Linear workspace.`
)

func init() {
	registry.RegisterIntegration("linear", &Linear{})
}

type Linear struct{}

type Metadata struct {
	State       *string `json:"state,omitempty" mapstructure:"state,omitempty"`
	Teams       []Team  `json:"teams" mapstructure:"teams"`
	Labels      []Label `json:"labels" mapstructure:"labels"`
	WebhookURL  string  `json:"webhookUrl,omitempty" mapstructure:"webhookUrl,omitempty"`
	CallbackURL string  `json:"callbackUrl,omitempty" mapstructure:"callbackUrl,omitempty"`
}

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
	return ""
}

func (l *Linear) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Description: "OAuth Client ID from your Linear app",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "OAuth Client Secret from your Linear app",
		},
		{
			Name:        "webhookSecret",
			Label:       "Webhook Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Webhook signing secret from your Linear app",
		},
	}
}

func (l *Linear) Components() []core.Component {
	return []core.Component{
		&CreateIssue{},
	}
}

func (l *Linear) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (l *Linear) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *Linear) Sync(ctx core.SyncContext) error {
	baseURL := ctx.WebhooksBaseURL
	if baseURL == "" {
		baseURL = ctx.BaseURL
	}
	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", baseURL, ctx.Integration.ID())
	webhookURL := fmt.Sprintf("%s/api/v1/integrations/%s/webhook", baseURL, ctx.Integration.ID())

	clientID, _ := ctx.Integration.GetConfig("clientId")
	clientSecret, _ := ctx.Integration.GetConfig("clientSecret")

	// No credentials yet — show setup instructions with URLs in metadata.
	if string(clientID) == "" || string(clientSecret) == "" {
		ctx.Integration.SetMetadata(Metadata{
			WebhookURL:  webhookURL,
			CallbackURL: callbackURL,
		})
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appSetupDescription,
			URL:         "https://linear.app/settings/api/applications/new",
			Method:      "GET",
		})

		return nil
	}

	// No access token — ask user to authorize.
	accessToken, _ := findSecret(ctx.Integration, OAuthAccessToken)
	if accessToken == "" {
		return l.handleOAuthNoAccessToken(ctx, callbackURL, string(clientID))
	}

	// Refresh token and update metadata.
	err := l.refreshToken(ctx, string(clientID), string(clientSecret))
	if err != nil {
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

func (l *Linear) handleOAuthNoAccessToken(ctx core.SyncContext, callbackURL, clientID string) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("Failed to decode metadata while setting state: %v", err)
	}

	if metadata.State == nil {
		s, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		metadata.State = &s
		ctx.Integration.SetMetadata(metadata)
	}

	authURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&actor=%s",
		linearAuthorizeURL,
		url.QueryEscape(clientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape(linearScopes),
		url.QueryEscape(*metadata.State),
		url.QueryEscape("app"),
	)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appConnectDescription,
		URL:         authURL,
		Method:      "GET",
	})

	return nil
}

func (l *Linear) refreshToken(ctx core.SyncContext, clientID, clientSecret string) error {
	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshToken)
	if refreshToken == "" {
		ctx.Logger.Warn("Linear integration has no refresh token - not refreshing token")
		return nil
	}

	ctx.Logger.Info("Refreshing Linear token")
	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(clientID, clientSecret, refreshToken)

	if err != nil {
		_ = ctx.Integration.SetSecret(OAuthRefreshToken, []byte(""))
		_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(""))
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	if tokenResponse.AccessToken != "" {
		ctx.Logger.Info("Saving access token")
		if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
			return fmt.Errorf("failed to save access token: %v", err)
		}
	}

	if tokenResponse.RefreshToken != "" {
		ctx.Logger.Info("Saving refresh token")
		if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			return fmt.Errorf("failed to save refresh token: %v", err)
		}
	}

	ctx.Logger.Info("Token refreshed successfully")
	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

func (l *Linear) updateMetadata(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	_, err = client.GetViewer()
	if err != nil {
		return fmt.Errorf("verify credentials: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	labels, err := client.ListLabels()
	if err != nil {
		return fmt.Errorf("list labels: %w", err)
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}

	metadata.Teams = teams
	metadata.Labels = labels
	metadata.State = nil
	metadata.WebhookURL = ""
	metadata.CallbackURL = ""
	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (l *Linear) HandleRequest(ctx core.HTTPRequestContext) {
	switch {
	case strings.HasSuffix(ctx.Request.URL.Path, "/webhook"):
		l.handleWebhookEvent(ctx)
	case strings.HasSuffix(ctx.Request.URL.Path, "/callback"):
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

		l.handleCallback(ctx, string(clientID), string(clientSecret))
	default:
		ctx.Response.WriteHeader(http.StatusNotFound)
	}
}

func (l *Linear) handleWebhookEvent(ctx core.HTTPRequestContext) {
	defer ctx.Request.Body.Close()
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("error reading webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	webhookSecret, _ := ctx.Integration.GetConfig("webhookSecret")
	if len(webhookSecret) == 0 {
		ctx.Logger.Errorf("webhook secret not configured - refusing to accept unverified webhook")
		ctx.Response.WriteHeader(http.StatusUnauthorized)
		return
	}

	signature := ctx.Request.Header.Get("Linear-Signature")
	if signature == "" {
		ctx.Logger.Errorf("missing Linear-Signature header")
		ctx.Response.WriteHeader(http.StatusUnauthorized)
		return
	}

	mac := hmac.New(sha256.New, webhookSecret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		ctx.Logger.Errorf("invalid webhook signature")
		ctx.Response.WriteHeader(http.StatusUnauthorized)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("error unmarshaling webhook payload: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if err := subscription.SendMessage(payload); err != nil {
			ctx.Logger.Errorf("error sending message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (l *Linear) handleCallback(ctx core.HTTPRequestContext, clientID, clientSecret string) {
	redirectBaseURL := ctx.BaseURL
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Use WebhooksBaseURL for the redirect URI to match what was sent during OAuth initiation in Sync().
	webhooksBaseURL := ctx.WebhooksBaseURL
	if webhooksBaseURL == "" {
		webhooksBaseURL = ctx.BaseURL
	}
	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", webhooksBaseURL, ctx.Integration.ID().String())

	if metadata.State == nil {
		ctx.Logger.Errorf("Callback error: missing OAuth state in metadata")
		http.Redirect(ctx.Response, ctx.Request,
			fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
			http.StatusSeeOther)
		return
	}

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.HandleCallback(ctx.Request, clientID, clientSecret, *metadata.State, redirectURI)

	if err != nil {
		ctx.Logger.Errorf("Callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request,
			fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
			http.StatusSeeOther)
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

	if err := ctx.Integration.ScheduleResync(tokenResponse.GetExpiration()); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := l.updateMetadata(core.SyncContext{
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}); err != nil {
		ctx.Logger.Errorf("Callback error: failed to update metadata: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request,
		fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
		http.StatusSeeOther)
}

func (l *Linear) Actions() []core.Action {
	return []core.Action{}
}

func (l *Linear) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
