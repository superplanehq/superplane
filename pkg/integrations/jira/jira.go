package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	AuthType string `json:"authType"`
	// API Token fields
	BaseURL  string `json:"baseUrl"`
	Email    string `json:"email"`
	APIToken string `json:"apiToken"`
	// OAuth fields
	ClientID     *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
}

type Metadata struct {
	Projects []Project `json:"projects"`
	// OAuth fields
	State   string `json:"state,omitempty"`
	CloudID string `json:"cloudId,omitempty"`
}

// WebhookConfiguration represents the configuration for a Jira webhook.
type WebhookConfiguration struct {
	EventType string `json:"eventType"`
	Project   string `json:"project"`
}

// WebhookMetadata stores the webhook ID for cleanup.
type WebhookMetadata struct {
	ID int64 `json:"id"`
}

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
	return `Jira supports two authentication methods. Choose the one that fits your use case:

## API Token

Use this method for simple setups that only need issue management (no webhooks).

1. Go to **Atlassian API Tokens** (https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click **Create API token** and copy the token
3. Select **API Token** as auth type below and enter your Jira base URL, email, and token

## OAuth 2.0

Use this method if you need webhook support (e.g. On Issue Created trigger).

1. Go to **Atlassian Developer Console** (https://developer.atlassian.com/console/myapps/)
2. Click **Create** > **OAuth 2.0 integration**, name your app and agree to the terms
3. Go to **Permissions** and add these scopes under **Jira API**:
   - read:jira-work - Read issues and projects
   - write:jira-work - Create issues
   - read:jira-user - Read user info
   - manage:jira-webhook - Register and delete webhooks
4. Go to **Distribution**, click **Edit**, and set the status to **Sharing** to allow other users in your organization to connect
5. Go to **Settings** to find the **Client ID** and create a **Secret**
6. Select **OAuth 2.0** as auth type below and enter the Client ID and Client Secret
7. After creating the integration, go to the Atlassian app **Authorization** > **OAuth 2.0 (3LO)** and set the callback URL to your SuperPlane integration callback URL
8. Click **Connect** to authorize with Atlassian`
}

func (j *Jira) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "authType",
			Label:    "Auth Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AuthTypeAPIToken,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "API Token", Value: AuthTypeAPIToken},
						{Label: "OAuth 2.0", Value: AuthTypeOAuth},
					},
				},
			},
		},
		{
			Name:        "baseUrl",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jira Cloud instance URL (e.g. https://your-domain.atlassian.net)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAPIToken}},
			},
		},
		{
			Name:        "email",
			Label:       "Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Email address for API authentication",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAPIToken}},
			},
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Jira API token",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAPIToken}},
			},
		},
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OAuth 2.0 Client ID from Atlassian Developer Console",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeOAuth}},
			},
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "OAuth 2.0 Client Secret from Atlassian Developer Console",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeOAuth}},
			},
		},
	}
}

func (j *Jira) Components() []core.Component {
	return []core.Component{
		&CreateIssue{},
	}
}

func (j *Jira) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueCreated{},
	}
}

func (j *Jira) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jira) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.AuthType == AuthTypeOAuth {
		return j.oauthSync(ctx, config)
	}

	return j.apiTokenSync(ctx, config)
}

func (j *Jira) apiTokenSync(ctx core.SyncContext, config Configuration) error {
	if config.BaseURL == "" {
		return fmt.Errorf("baseUrl is required")
	}

	if config.Email == "" {
		return fmt.Errorf("email is required")
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("error listing projects: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Projects: projects})
	ctx.Integration.Ready()
	return nil
}

func (j *Jira) oauthSync(ctx core.SyncContext, config Configuration) error {
	if config.ClientID == nil || *config.ClientID == "" {
		return fmt.Errorf("clientId is required")
	}

	metadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)

	accessToken, _ := findOAuthSecret(ctx.Integration, OAuthAccessToken)

	if accessToken != "" && metadata.CloudID != "" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			_, err = client.GetCurrentUser()
			if err == nil {
				projects, err := client.ListProjects()
				if err == nil {
					metadata.Projects = projects
					ctx.Integration.SetMetadata(metadata)
					ctx.Integration.Ready()
					return ctx.Integration.ScheduleResync(30 * time.Minute)
				}
				ctx.Logger.Errorf("oauthSync: failed to list projects: %v", err)
			} else {
				ctx.Logger.Errorf("oauthSync: failed to get current user: %v", err)
			}
		} else {
			ctx.Logger.Errorf("oauthSync: failed to create client: %v", err)
		}

		// Tokens invalid, try to refresh
		refreshToken, _ := findOAuthSecret(ctx.Integration, OAuthRefreshToken)
		if refreshToken != "" {
			clientSecret, err := ctx.Integration.GetConfig("clientSecret")
			if err == nil {
				tokenResponse, err := refreshOAuthToken(ctx.HTTP, *config.ClientID, string(clientSecret), refreshToken)
				if err == nil {
					// Store new tokens
					_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken))
					if tokenResponse.RefreshToken != "" {
						_ = ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken))
					}

					// Retry with new tokens
					client, err := NewClient(ctx.HTTP, ctx.Integration)
					if err == nil {
						projects, err := client.ListProjects()
						if err == nil {
							metadata.Projects = projects
							ctx.Integration.SetMetadata(metadata)
							ctx.Integration.Ready()
							return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
						}
					}
				}
			}
		}
	}

	// No valid tokens, need to authorize.
	// Reuse existing state if an OAuth flow is already in progress to avoid
	// invalidating a callback the user has not completed yet.
	state := metadata.State
	if state == "" {
		var err error
		state, err = crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}

		metadata.State = state
		ctx.Integration.SetMetadata(metadata)
	}

	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.WebhooksBaseURL, ctx.Integration.ID().String())
	authURL := buildAuthorizationURL(*config.ClientID, redirectURI, state)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: "Authorize with Atlassian",
		URL:         authURL,
		Method:      "GET",
	})

	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		j.handleOAuthCallback(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/actions/getFailedWebhooks") {
		j.handleGetFailedWebhooks(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/actions/getFailedWebhooks") {
		j.handleGetFailedWebhooks(ctx)
		return
	}
}

func (j *Jira) handleGetFailedWebhooks(ctx core.HTTPRequestContext) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("handleGetFailedWebhooks: failed to create client: %v", err)
		http.Error(ctx.Response, fmt.Sprintf("failed to create client: %v", err), http.StatusInternalServerError)
		return
	}

	failed, err := client.GetFailedWebhooks()
	if err != nil {
		ctx.Logger.Errorf("handleGetFailedWebhooks: error: %v", err)
		http.Error(ctx.Response, fmt.Sprintf("error getting failed webhooks: %v", err), http.StatusInternalServerError)
		return
	}

	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(failed)
}

func (j *Jira) handleOAuthCallback(ctx core.HTTPRequestContext) {
	code := ctx.Request.URL.Query().Get("code")
	state := ctx.Request.URL.Query().Get("state")

	if code == "" || state == "" {
		ctx.Logger.Errorf("missing code or state")
		http.Error(ctx.Response, "missing code or state", http.StatusBadRequest)
		return
	}

	// Validate state
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("failed to decode metadata: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if state != metadata.State {
		ctx.Logger.Errorf("invalid state")
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	// Get client credentials
	clientID, err := ctx.Integration.GetConfig("clientId")
	if err != nil {
		ctx.Logger.Errorf("failed to get clientId: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		ctx.Logger.Errorf("failed to get clientSecret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.WebhooksBaseURL, ctx.Integration.ID().String())

	// Exchange code for tokens
	tokenResponse, err := exchangeCodeForTokens(ctx.HTTP, string(clientID), string(clientSecret), code, redirectURI)
	if err != nil {
		ctx.Logger.Errorf("failed to exchange code for tokens: %v", err)
		http.Error(ctx.Response, "failed to exchange code for tokens", http.StatusInternalServerError)
		return
	}

	// Get accessible resources to find cloud ID
	resources, err := getAccessibleResources(ctx.HTTP, tokenResponse.AccessToken)
	if err != nil {
		ctx.Logger.Errorf("failed to get accessible resources: %v", err)
		http.Error(ctx.Response, "failed to get accessible resources", http.StatusInternalServerError)
		return
	}

	if len(resources) == 0 {
		ctx.Logger.Errorf("no accessible Jira resources found")
		http.Error(ctx.Response, "no accessible Jira resources found", http.StatusBadRequest)
		return
	}

	// Use first resource (most users have only one)
	cloudID := resources[0].ID

	// Store tokens as secrets
	if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
		ctx.Logger.Errorf("failed to store access token: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if tokenResponse.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			ctx.Logger.Errorf("failed to store refresh token: %v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Update metadata with cloud ID and clear state
	metadata.CloudID = cloudID
	metadata.State = ""
	ctx.Integration.SetMetadata(metadata)

	// Load projects and mark integration as ready so the user does not
	// have to wait for the next sync cycle.
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("failed to create client after OAuth: %v", err)
	} else {
		projects, err := client.ListProjects()
		if err != nil {
			ctx.Logger.Errorf("failed to list projects after OAuth: %v", err)
		} else {
			metadata.Projects = projects
			ctx.Integration.SetMetadata(metadata)
		}
	}

	ctx.Integration.Ready()
	_ = ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())

	// Remove browser action
	ctx.Integration.RemoveBrowserAction()

	// Redirect to integration settings page
	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
		http.StatusSeeOther,
	)
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "getFailedWebhooks",
			Description:    "Get webhooks that failed to be delivered in the last 72 hours",
			UserAccessible: true,
		},
	}
}

func (j *Jira) HandleAction(ctx core.IntegrationActionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	switch ctx.Name {
	case "getFailedWebhooks":
		_, err := client.GetFailedWebhooks()
		if err != nil {
			return fmt.Errorf("error getting failed webhooks: %v", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
