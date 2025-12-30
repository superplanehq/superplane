package smtp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	registry.RegisterApplication("smtp", &SMTP{})
}

const (
	SMTPAccessToken  = "accessToken"
	SMTPRefreshToken = "refreshToken"
)

type SMTP struct{}

type Configuration struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	AuthMethod string `json:"authMethod"` // "password" or "oauth2"

	// For password auth (only needed if authMethod == "password")
	User     *string `json:"user,omitempty"`
	Password *string `json:"password,omitempty"`

	// For OAuth 2.0 (only needed if authMethod == "oauth2")
	Provider     *string `json:"provider,omitempty"` // "gmail" or "microsoft"
	ClientID     *string `json:"clientId,omitempty"`
	ClientSecret *string `json:"clientSecret,omitempty"`
}

type Metadata struct {
	Provider    string `json:"provider"`
	TokenExpiry string `json:"tokenExpiry"`
	State       string `json:"state"`
}

func (s *SMTP) Name() string {
	return "smtp"
}

func (s *SMTP) Label() string {
	return "SMTP"
}

func (s *SMTP) Icon() string {
	return "mail"
}

func (s *SMTP) Description() string {
	return "Send emails"
}

func (s *SMTP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "host",
			Label:       "Host",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g. smtp.gmail.com",
			Required:    true,
		},
		{
			Name:     "port",
			Label:    "Port",
			Type:     configuration.FieldTypeNumber,
			Required: true,
			Default:  587,
		},
		{
			Name:        "authMethod",
			Label:       "Authentication Method",
			Type:        configuration.FieldTypeSelect,
			Description: "Choose how to authenticate with the SMTP server",
			Required:    true,
			Default:     "password",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Password",
							Value: "password",
						},
						{
							Label: "OAuth 2.0",
							Value: "oauth2",
						},
					},
				},
			},
		},
		{
			Name:     "user",
			Label:    "User",
			Type:     configuration.FieldTypeString,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authMethod",
					Values: []string{"password"},
				},
			},
		},
		{
			Name:        "password",
			Label:       "Password",
			Type:        configuration.FieldTypeString,
			Description: "Your password or app-specific password",
			Sensitive:   true,
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authMethod",
					Values: []string{"password"},
				},
			},
		},
		{
			Name:        "provider",
			Label:       "Provider",
			Type:        configuration.FieldTypeSelect,
			Description: "OAuth provider",
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authMethod",
					Values: []string{"oauth2"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Gmail",
							Value: "gmail",
						},
						{
							Label: "Microsoft/Outlook",
							Value: "microsoft",
						},
					},
				},
			},
		},
		{
			Name:        "clientId",
			Label:       "OAuth Client ID",
			Type:        configuration.FieldTypeString,
			Description: "OAuth 2.0 Client ID from your provider",
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authMethod",
					Values: []string{"oauth2"},
				},
			},
		},
		{
			Name:        "clientSecret",
			Label:       "OAuth Client Secret",
			Type:        configuration.FieldTypeString,
			Description: "OAuth 2.0 Client Secret from your provider",
			Sensitive:   true,
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authMethod",
					Values: []string{"oauth2"},
				},
			},
		},
	}
}

func (s *SMTP) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	// Handle password auth - just verify connection
	if config.AuthMethod == "password" {
		client, err := NewClient(ctx.AppInstallation)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}

		dialCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := client.DialWithContext(dialCtx); err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %v", err)
		}
		defer client.Close()

		ctx.AppInstallation.SetState("ready", "")
		return nil
	}

	// Handle OAuth 2.0 auth
	if config.AuthMethod == "oauth2" {
		return s.syncOAuth(ctx, config)
	}

	return fmt.Errorf("invalid authMethod: %s", config.AuthMethod)
}

func (s *SMTP) syncOAuth(ctx core.SyncContext, config Configuration) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	// Check if we already have valid tokens
	secrets, err := ctx.AppInstallation.GetSecrets()
	if err != nil {
		return fmt.Errorf("failed to get secrets: %v", err)
	}

	hasAccessToken := false
	for _, secret := range secrets {
		if secret.Name == SMTPAccessToken {
			hasAccessToken = true
			break
		}
	}

	// If we have tokens, check if they need refreshing
	if hasAccessToken {
		// Check if token is expired
		if metadata.TokenExpiry != "" {
			expiryTime, err := time.Parse(time.RFC3339, metadata.TokenExpiry)
			if err == nil && time.Now().After(expiryTime) {
				// Token expired, refresh it
				err = s.refreshAccessToken(ctx.AppInstallation, config)
				if err != nil {
					return fmt.Errorf("failed to refresh access token: %v", err)
				}
			}
		}
		ctx.AppInstallation.SetState("ready", "")
		return nil
	}

	// No tokens - initiate OAuth flow
	state, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("failed to generate state: %v", err)
	}

	authURL := s.getOAuthURL(config, state, ctx.BaseURL, ctx.InstallationID)

	ctx.AppInstallation.NewBrowserAction(core.BrowserAction{
		Description: "Click Continue to authorize Superplane to send emails on your behalf via OAuth 2.0",
		URL:         authURL,
		Method:      "GET",
	})

	metadata.State = state
	metadata.Provider = *config.Provider
	ctx.AppInstallation.SetMetadata(metadata)

	return nil
}

func (s *SMTP) getOAuthURL(config Configuration, state, baseURL, installationID string) string {
	redirectURI := fmt.Sprintf("%s/api/v1/apps/%s/oauth/callback", baseURL, installationID)

	var authEndpoint, scope string
	switch *config.Provider {
	case "gmail":
		authEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
		scope = "https://mail.google.com/"
	case "microsoft":
		authEndpoint = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
		scope = "https://outlook.office.com/SMTP.Send offline_access"
	default:
		return ""
	}

	return fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline&prompt=consent",
		authEndpoint, *config.ClientID, redirectURI, scope, state)
}

func (s *SMTP) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/oauth/callback") {
		s.handleOAuthCallback(ctx)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (s *SMTP) handleOAuthCallback(ctx core.HTTPRequestContext) {
	code := ctx.Request.URL.Query().Get("code")
	state := ctx.Request.URL.Query().Get("state")

	if code == "" || state == "" {
		ctx.Logger.Errorf("missing code or state")
		http.Error(ctx.Response, "missing code or state", http.StatusBadRequest)
		return
	}

	metadata := Metadata{}
	err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		ctx.Logger.Errorf("failed to decode metadata: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	// Verify state
	if state != metadata.State {
		ctx.Logger.Errorf("invalid state")
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	// Get OAuth configuration fields
	provider, err := ctx.AppInstallation.GetConfig("provider")
	if err != nil {
		ctx.Logger.Errorf("failed to get provider: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	clientID, err := ctx.AppInstallation.GetConfig("clientId")
	if err != nil {
		ctx.Logger.Errorf("failed to get clientId: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	clientSecret, err := ctx.AppInstallation.GetConfig("clientSecret")
	if err != nil {
		ctx.Logger.Errorf("failed to get clientSecret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	// Exchange code for tokens
	tokens, err := s.exchangeCodeForTokens(code, string(provider), string(clientID), string(clientSecret), ctx.BaseURL, ctx.AppInstallation.ID().String())
	if err != nil {
		ctx.Logger.Errorf("failed to exchange code for tokens: %v", err)
		http.Error(ctx.Response, "failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Store tokens as secrets
	err = ctx.AppInstallation.SetSecret(SMTPAccessToken, []byte(tokens.AccessToken))
	if err != nil {
		ctx.Logger.Errorf("failed to save access token: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.AppInstallation.SetSecret(SMTPRefreshToken, []byte(tokens.RefreshToken))
	if err != nil {
		ctx.Logger.Errorf("failed to save refresh token: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	// Update metadata with token expiry
	metadata.TokenExpiry = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).Format(time.RFC3339)
	metadata.State = ""
	ctx.AppInstallation.SetMetadata(metadata)

	// Remove browser action and set state to ready
	ctx.AppInstallation.RemoveBrowserAction()
	ctx.AppInstallation.SetState("ready", "")

	ctx.Logger.Infof("Successfully authenticated SMTP via OAuth 2.0")

	// Redirect back to the app installation page
	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf("%s/%s/settings/applications/%s", ctx.BaseURL, ctx.OrganizationID, ctx.AppInstallation.ID().String()),
		http.StatusSeeOther,
	)
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (s *SMTP) exchangeCodeForTokens(code, provider, clientID, clientSecret, baseURL, installationID string) (*TokenResponse, error) {
	redirectURI := fmt.Sprintf("%s/api/v1/apps/%s/oauth/callback", baseURL, installationID)

	var tokenEndpoint string
	switch provider {
	case "gmail":
		tokenEndpoint = "https://oauth2.googleapis.com/token"
	case "microsoft":
		tokenEndpoint = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokens TokenResponse
	err = json.Unmarshal(body, &tokens)
	if err != nil {
		return nil, err
	}

	return &tokens, nil
}

func (s *SMTP) refreshAccessToken(ctx core.AppInstallationContext, config Configuration) error {
	// Get refresh token from secrets
	refreshToken, err := findSecretFromContext(ctx, SMTPRefreshToken)
	if err != nil {
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Get provider and client credentials
	provider, err := ctx.GetConfig("provider")
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	clientID, err := ctx.GetConfig("clientId")
	if err != nil {
		return fmt.Errorf("failed to get clientId: %w", err)
	}

	clientSecret, err := ctx.GetConfig("clientSecret")
	if err != nil {
		return fmt.Errorf("failed to get clientSecret: %w", err)
	}

	// Determine token endpoint
	var tokenEndpoint string
	switch string(provider) {
	case "gmail":
		tokenEndpoint = "https://oauth2.googleapis.com/token"
	case "microsoft":
		tokenEndpoint = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	default:
		return fmt.Errorf("unknown provider: %s", string(provider))
	}

	// Prepare refresh request
	data := url.Values{}
	data.Set("client_id", string(clientID))
	data.Set("client_secret", string(clientSecret))
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokens TokenResponse
	err = json.Unmarshal(body, &tokens)
	if err != nil {
		return err
	}

	// Update access token secret
	err = ctx.SetSecret(SMTPAccessToken, []byte(tokens.AccessToken))
	if err != nil {
		return fmt.Errorf("failed to update access token: %w", err)
	}

	// Update refresh token if a new one was provided
	if tokens.RefreshToken != "" {
		err = ctx.SetSecret(SMTPRefreshToken, []byte(tokens.RefreshToken))
		if err != nil {
			return fmt.Errorf("failed to update refresh token: %w", err)
		}
	}

	// Update metadata with new expiry
	metadata := Metadata{}
	err = mapstructure.Decode(ctx.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	metadata.TokenExpiry = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).Format(time.RFC3339)
	ctx.SetMetadata(metadata)

	return nil
}

func findSecretFromContext(ctx core.AppInstallationContext, secretName string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return string(secret.Value), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", secretName)
}

func (s *SMTP) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (s *SMTP) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	return nil, nil
}

func (s *SMTP) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	return nil
}

func (s *SMTP) Components() []core.Component {
	return []core.Component{
		&SendEmail{},
	}
}

func (s *SMTP) Triggers() []core.Trigger {
	return []core.Trigger{}
}
