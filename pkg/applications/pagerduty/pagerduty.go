package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("pagerduty", &PagerDuty{})
}

type PagerDuty struct{}

const (
	PagerDutyAppClientID     = "clientId"
	PagerDutyAppClientSecret = "clientSecret"
	PagerDutyAppAccessToken  = "accessToken"
	PagerDutyAppRefreshToken = "refreshToken"
)

type Configuration struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type Metadata struct {
	State     string `json:"state,omitempty"`
	AccountID string `json:"accountId,omitempty"`
	SubDomain string `json:"subDomain,omitempty"`
}

type WebhookConfiguration struct {
	Events    []string `json:"events"`    // Specific event types, e.g., ["incident.resolved", "incident.triggered"]
	ServiceID string   `json:"serviceId"` // Optional: filter by service
	TeamID    string   `json:"teamId"`    // Optional: filter by team
}

type WebhookMetadata struct {
	SubscriptionID string `json:"subscriptionId"` // PagerDuty subscription ID for cleanup
}

func (p *PagerDuty) Name() string {
	return "pagerduty"
}

func (p *PagerDuty) Label() string {
	return "PagerDuty"
}

func (p *PagerDuty) Icon() string {
	return "alert-triangle"
}

func (p *PagerDuty) Description() string {
	return "Manage and react to incidents in PagerDuty"
}

func (p *PagerDuty) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OAuth Client ID from your PagerDuty App",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "OAuth Client Secret from your PagerDuty App",
		},
	}
}

func (p *PagerDuty) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (p *PagerDuty) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
		&OnIncidentNote{},
		&OnIncidentResponder{},
		&OnIncidentStatusUpdate{},
		&OnIncidentFieldValues{},
		&OnService{},
		&OnServiceFieldValues{},
	}
}

func (p *PagerDuty) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	// Check if OAuth flow is already completed
	secrets, err := ctx.AppInstallation.GetSecrets()
	if err != nil {
		return fmt.Errorf("failed to get secrets: %v", err)
	}

	hasAccessToken := false
	for _, secret := range secrets {
		if secret.Name == PagerDutyAppAccessToken {
			hasAccessToken = true
			break
		}
	}

	if hasAccessToken {
		// Already authenticated, verify token is still valid
		client, err := NewClient(ctx.AppInstallation)
		if err != nil {
			return fmt.Errorf("error creating client: %v", err)
		}

		_, err = client.GetCurrentUser()
		if err != nil {
			// Token might be expired, clear state to trigger re-auth
			ctx.AppInstallation.SetState("error", "OAuth token expired or invalid")
			metadata.State = ""
			ctx.AppInstallation.SetMetadata(metadata)
			return fmt.Errorf("OAuth token is invalid: %v", err)
		}

		ctx.AppInstallation.SetState("ready", "")
		return nil
	}

	// Generate state for OAuth flow
	state, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("failed to generate state: %v", err)
	}

	// Construct OAuth authorization URL
	redirectURI := fmt.Sprintf("%s/api/v1/apps/%s/oauth/callback", ctx.BaseURL, ctx.InstallationID)
	authURL := fmt.Sprintf(
		"https://app.pagerduty.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		config.ClientID,
		redirectURI,
		state,
	)

	ctx.AppInstallation.NewBrowserAction(core.BrowserAction{
		Description: "To complete the PagerDuty app setup, click the button below to authorize access.",
		URL:         authURL,
		Method:      "GET",
	})

	metadata.State = state
	ctx.AppInstallation.SetMetadata(metadata)

	return nil
}

func (p *PagerDuty) HandleRequest(ctx core.HTTPRequestContext) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		ctx.Logger.Errorf("Error decoding metadata: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/oauth/callback") {
		p.handleOAuthCallback(ctx, metadata)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (p *PagerDuty) handleOAuthCallback(ctx core.HTTPRequestContext, metadata Metadata) {
	// Verify state parameter
	state := ctx.Request.URL.Query().Get("state")
	if state != metadata.State {
		ctx.Logger.Errorf("Invalid state parameter")
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		ctx.Logger.Errorf("Missing authorization code")
		http.Error(ctx.Response, "missing code", http.StatusBadRequest)
		return
	}

	// Exchange code for access token
	// Get configuration from context (need to query it properly)
	configData, err := ctx.AppInstallation.GetConfig("clientId")
	if err != nil {
		ctx.Logger.Errorf("Error getting client ID: %v", err)
		http.Error(ctx.Response, "internal error", http.StatusInternalServerError)
		return
	}
	clientID := string(configData)

	configData, err = ctx.AppInstallation.GetConfig("clientSecret")
	if err != nil {
		ctx.Logger.Errorf("Error getting client secret: %v", err)
		http.Error(ctx.Response, "internal error", http.StatusInternalServerError)
		return
	}
	clientSecret := string(configData)

	redirectURI := fmt.Sprintf("%s/api/v1/apps/%s/oauth/callback", ctx.BaseURL, ctx.AppInstallation.ID().String())

	tokenResponse, err := exchangeCodeForToken(clientID, clientSecret, code, redirectURI)
	if err != nil {
		ctx.Logger.Errorf("Error exchanging code for token: %v", err)
		http.Error(ctx.Response, "failed to exchange code", http.StatusInternalServerError)
		return
	}

	// Store tokens as secrets
	err = ctx.AppInstallation.SetSecret(PagerDutyAppAccessToken, []byte(tokenResponse.AccessToken))
	if err != nil {
		ctx.Logger.Errorf("Error storing access token: %v", err)
		http.Error(ctx.Response, "internal error", http.StatusInternalServerError)
		return
	}

	if tokenResponse.RefreshToken != "" {
		err = ctx.AppInstallation.SetSecret(PagerDutyAppRefreshToken, []byte(tokenResponse.RefreshToken))
		if err != nil {
			ctx.Logger.Errorf("Error storing refresh token: %v", err)
			http.Error(ctx.Response, "internal error", http.StatusInternalServerError)
			return
		}
	}

	// Get user info to verify token and store metadata
	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		ctx.Logger.Errorf("Error creating client: %v", err)
		http.Error(ctx.Response, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		ctx.Logger.Errorf("Error getting current user: %v", err)
		http.Error(ctx.Response, "failed to verify token", http.StatusInternalServerError)
		return
	}

	// Update metadata
	metadata.AccountID = user.ID
	metadata.State = ""
	ctx.AppInstallation.SetMetadata(metadata)
	ctx.AppInstallation.RemoveBrowserAction()
	ctx.AppInstallation.SetState("ready", "")

	ctx.Logger.Infof("Successfully authenticated PagerDuty app for user %s", user.Email)

	// Redirect to app installation page
	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf("%s/%s/settings/applications/%s", ctx.BaseURL, ctx.OrganizationID, ctx.AppInstallation.ID().String()),
		http.StatusSeeOther,
	)
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func exchangeCodeForToken(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	tokenURL := "https://app.pagerduty.com/oauth/token"

	requestBody := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &tokenResponse, nil
}

func (p *PagerDuty) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	// Service/Team filters must match exactly
	if configA.ServiceID != configB.ServiceID || configA.TeamID != configB.TeamID {
		return false, nil
	}

	// Check if A contains all events from B (A is superset of B)
	// This allows webhook sharing when existing webhook has more events than needed
	for _, eventB := range configB.Events {
		if !slices.Contains(configA.Events, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (p *PagerDuty) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	client, err := NewClient(ctx)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(options.Configuration, &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	// Determine filter type based on configuration
	var filterType, filterID string
	if configuration.ServiceID != "" {
		filterType = "service_reference"
		filterID = configuration.ServiceID
	} else if configuration.TeamID != "" {
		filterType = "team_reference"
		filterID = configuration.TeamID
	} else {
		filterType = "account_reference"
		filterID = ""
	}

	// Create webhook subscription with specific events and filter
	subscription, err := client.CreateWebhookSubscription(options.URL, configuration.Events, filterType, filterID)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook subscription: %v", err)
	}

	// Return metadata containing subscription ID for cleanup
	return WebhookMetadata{
		SubscriptionID: subscription.ID,
	}, nil
}

func (p *PagerDuty) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(options.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	err = client.DeleteWebhookSubscription(metadata.SubscriptionID)
	if err != nil {
		return fmt.Errorf("error deleting webhook subscription: %v", err)
	}

	return nil
}
