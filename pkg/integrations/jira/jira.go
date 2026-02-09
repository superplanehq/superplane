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
	registry.RegisterIntegration("jira", &Jira{})
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
	return ""
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
		&ListWebhooks{},
		&DeleteWebhooks{},
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
	ctx.Logger.Infof("Sync: starting, configuration=%+v", ctx.Configuration)

	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		ctx.Logger.Errorf("Sync: failed to decode config: %v", err)
		return fmt.Errorf("failed to decode config: %v", err)
	}

	ctx.Logger.Infof("Sync: authType=%s", config.AuthType)

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
	ctx.Logger.Infof("oauthSync: starting")

	if config.ClientID == nil || *config.ClientID == "" {
		ctx.Logger.Errorf("oauthSync: clientId is required")
		return fmt.Errorf("clientId is required")
	}

	// Get existing metadata
	metadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	ctx.Logger.Infof("oauthSync: existing metadata cloudID=%s, state=%s", metadata.CloudID, metadata.State)

	// Check if we already have valid tokens
	accessToken, _ := findOAuthSecret(ctx.Integration, OAuthAccessToken)
	ctx.Logger.Infof("oauthSync: hasAccessToken=%v, hasCloudID=%v", accessToken != "", metadata.CloudID != "")

	if accessToken != "" && metadata.CloudID != "" {
		ctx.Logger.Infof("oauthSync: attempting to use existing tokens")
		// Try to use existing tokens
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			_, err = client.GetCurrentUser()
			if err == nil {
				// Tokens are still valid, refresh projects and mark ready
				projects, err := client.ListProjects()
				if err == nil {
					ctx.Logger.Infof("oauthSync: tokens valid, found %d projects", len(projects))
					metadata.Projects = projects
					ctx.Integration.SetMetadata(metadata)
					ctx.Logger.Infof("oauthSync: calling Ready()")
					ctx.Integration.Ready()
					ctx.Logger.Infof("oauthSync: Ready() called, scheduling resync")

					// Schedule token refresh (30 minutes)
					resyncErr := ctx.Integration.ScheduleResync(30 * time.Minute)
					ctx.Logger.Infof("oauthSync: ScheduleResync returned: %v", resyncErr)
					return resyncErr
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

	// No valid tokens, need to authorize
	state, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("failed to generate state: %v", err)
	}

	metadata.State = state
	ctx.Integration.SetMetadata(metadata)

	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.WebhooksBaseURL, ctx.Integration.ID().String())
	ctx.Logger.Infof("oauthSync: WebhooksBaseURL=%s, redirectURI=%s", ctx.WebhooksBaseURL, redirectURI)
	authURL := buildAuthorizationURL(*config.ClientID, redirectURI, state)
	ctx.Logger.Infof("oauthSync: authURL=%s", authURL)

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

	if strings.HasSuffix(ctx.Request.URL.Path, "/actions/listWebhooks") {
		j.handleListWebhooks(ctx)
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

func (j *Jira) handleListWebhooks(ctx core.HTTPRequestContext) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("handleListWebhooks: failed to create client: %v", err)
		http.Error(ctx.Response, fmt.Sprintf("failed to create client: %v", err), http.StatusInternalServerError)
		return
	}

	webhooks, err := client.ListWebhooks()
	if err != nil {
		ctx.Logger.Errorf("handleListWebhooks: error: %v", err)
		http.Error(ctx.Response, fmt.Sprintf("error listing webhooks: %v", err), http.StatusInternalServerError)
		return
	}

	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(webhooks)
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
	ctx.Logger.Infof("handleOAuthCallback: WebhooksBaseURL=%s, redirectURI=%s", ctx.WebhooksBaseURL, redirectURI)

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

	// Remove browser action
	ctx.Integration.RemoveBrowserAction()

	ctx.Logger.Infof("Successfully authenticated with Jira Cloud %s", cloudID)

	// Redirect to integration settings page
	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
		http.StatusSeeOther,
	)
}

func (j *Jira) CompareWebhookConfig(a, b any) (bool, error) {
	var configA, configB WebhookConfiguration

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, fmt.Errorf("failed to decode config a: %w", err)
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, fmt.Errorf("failed to decode config b: %w", err)
	}

	return configA.EventType == configB.EventType && configA.Project == configB.Project, nil
}

func (j *Jira) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	logger.Infof("SetupWebhook: called")

	var config WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		logger.Errorf("SetupWebhook: failed to decode configuration: %v", err)
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}
	logger.Infof("SetupWebhook: config=%+v", config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		logger.Errorf("SetupWebhook: failed to create client: %v", err)
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	jqlFilter := fmt.Sprintf("project = %q", config.Project)
	events := []string{config.EventType}
	webhookURL := ctx.Webhook.GetURL()
	logger.Infof("SetupWebhook: webhookURL=%s, jqlFilter=%s, events=%v", webhookURL, jqlFilter, events)

	response, err := client.RegisterWebhook(webhookURL, jqlFilter, events)
	if err != nil {
		logger.Errorf("SetupWebhook: error registering webhook: %v", err)
		return nil, fmt.Errorf("error registering webhook: %v", err)
	}

	logger.Infof("SetupWebhook: response=%+v", response)

	if len(response.WebhookRegistrationResult) == 0 {
		logger.Errorf("SetupWebhook: no webhook registration result returned")
		return nil, fmt.Errorf("no webhook registration result returned")
	}

	result := response.WebhookRegistrationResult[0]
	if len(result.Errors) > 0 {
		logger.Errorf("SetupWebhook: webhook registration failed: %v", result.Errors)
		return nil, fmt.Errorf("webhook registration failed: %v", result.Errors)
	}

	logger.Infof("SetupWebhook: webhook created with ID=%d", result.CreatedWebhookID)
	return WebhookMetadata{ID: result.CreatedWebhookID}, nil
}

func (j *Jira) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	var metadata WebhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.ID == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	return client.DeleteWebhook([]int64{metadata.ID})
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "listWebhooks",
			Description:    "List all webhooks registered with Jira for this OAuth app",
			UserAccessible: true,
		},
		{
			Name:           "deleteWebhook",
			Description:    "Delete a single webhook by its Jira ID",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:     "webhookId",
					Label:    "Webhook ID",
					Type:     configuration.FieldTypeNumber,
					Required: true,
				},
			},
		},
		{
			Name:           "deleteAllWebhooks",
			Description:    "Delete all webhooks registered with Jira for this OAuth app",
			UserAccessible: true,
		},
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
	case "listWebhooks":
		webhooks, err := client.ListWebhooks()
		if err != nil {
			return fmt.Errorf("error listing webhooks: %v", err)
		}
		logger.Infof("HandleAction listWebhooks: found %d webhooks", webhooks.Total)
		for _, w := range webhooks.Values {
			logger.Infof("  Webhook ID=%d, JQL=%s, Events=%v", w.ID, w.JQLFilter, w.Events)
		}
		return nil

	case "deleteWebhook":
		params, ok := ctx.Parameters.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid parameters")
		}
		webhookID, ok := params["webhookId"].(float64)
		if !ok {
			return fmt.Errorf("webhookId parameter is required")
		}
		err := client.DeleteWebhookByID(int64(webhookID))
		if err != nil {
			return fmt.Errorf("error deleting webhook: %v", err)
		}
		logger.Infof("HandleAction deleteWebhook: deleted webhook %d", int64(webhookID))
		return nil

	case "deleteAllWebhooks":
		err := client.DeleteAllWebhooks()
		if err != nil {
			return fmt.Errorf("error deleting webhooks: %v", err)
		}
		logger.Infof("HandleAction deleteAllWebhooks: all webhooks deleted")
		return nil

	case "getFailedWebhooks":
		failed, err := client.GetFailedWebhooks()
		if err != nil {
			return fmt.Errorf("error getting failed webhooks: %v", err)
		}
		logger.Infof("HandleAction getFailedWebhooks: found %d failed webhooks", len(failed.Values))
		for _, f := range failed.Values {
			logger.Infof("  Failed webhook ID=%s, URL=%s, Reason=%s, Time=%s", f.ID, f.URL, f.FailureReason, f.LatestFailureTime)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
