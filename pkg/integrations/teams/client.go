package teams

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	botFrameworkOpenIDURL = "https://login.botframework.com/v1/.well-known/openidconfiguration"
	graphAPIBase          = "https://graph.microsoft.com/v1.0"
)

// Client handles Bot Framework REST API and Graph API interactions.
type Client struct {
	AppID       string
	AppPassword string
	TenantID    string
	httpClient  *http.Client
}

// NewClient creates a new Teams client from integration context.
func NewClient(ctx core.IntegrationContext) (*Client, error) {
	appID, err := ctx.GetConfig("appId")
	if err != nil {
		return nil, fmt.Errorf("failed to get appId: %w", err)
	}

	appPassword, err := ctx.GetConfig("appPassword")
	if err != nil {
		return nil, fmt.Errorf("failed to get appPassword: %w", err)
	}

	if string(appID) == "" || string(appPassword) == "" {
		return nil, fmt.Errorf("appId and appPassword are required")
	}

	var tenantID string
	tenantIDBytes, err := ctx.GetConfig("tenantId")
	if err == nil && tenantIDBytes != nil {
		tenantID = string(tenantIDBytes)
	}

	return &Client{
		AppID:       string(appID),
		AppPassword: string(appPassword),
		TenantID:    tenantID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// NewClientFromConfig creates a client directly from config values.
func NewClientFromConfig(appID, appPassword, tenantID string) *Client {
	return &Client{
		AppID:       appID,
		AppPassword: appPassword,
		TenantID:    tenantID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// TokenResponse represents an OAuth2 token response from Azure AD.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetBotToken fetches a Bot Framework access token from Azure AD.
func (c *Client) GetBotToken() (*TokenResponse, error) {
	tokenURL := c.botTokenURL()

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.AppID},
		"client_secret": {c.AppPassword},
		"scope":         {"https://api.botframework.com/.default"},
	}

	resp, err := c.httpClient.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &token, nil
}

// GetGraphToken fetches a Microsoft Graph API access token from Azure AD.
func (c *Client) GetGraphToken() (*TokenResponse, error) {
	tokenURL := c.graphTokenURL()

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.AppID},
		"client_secret": {c.AppPassword},
		"scope":         {"https://graph.microsoft.com/.default"},
	}

	resp, err := c.httpClient.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to request graph token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph token request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &token, nil
}

// Activity represents a Bot Framework Activity object.
type Activity struct {
	Type             string           `json:"type"`
	ID               string           `json:"id,omitempty"`
	Timestamp        string           `json:"timestamp,omitempty"`
	ChannelID        string           `json:"channelId,omitempty"`
	ServiceURL       string           `json:"serviceUrl,omitempty"`
	From             ChannelAccount   `json:"from,omitempty"`
	Conversation     ConversationInfo `json:"conversation,omitempty"`
	Recipient        ChannelAccount   `json:"recipient,omitempty"`
	Text             string           `json:"text,omitempty"`
	Entities         []Entity         `json:"entities,omitempty"`
	ChannelData      map[string]any   `json:"channelData,omitempty"`
	MembersAdded     []ChannelAccount `json:"membersAdded,omitempty"`
	MembersRemoved   []ChannelAccount `json:"membersRemoved,omitempty"`
	ReplyToID        string           `json:"replyToId,omitempty"`
	TextFormat       string           `json:"textFormat,omitempty"`
	AttachmentLayout string           `json:"attachmentLayout,omitempty"`
}

// ChannelAccount represents a user or bot account.
type ChannelAccount struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	AADObjectID string `json:"aadObjectId,omitempty"`
}

// ConversationInfo represents a conversation.
type ConversationInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name,omitempty"`
	IsGroup          bool   `json:"isGroup,omitempty"`
	ConversationType string `json:"conversationType,omitempty"`
	TenantID         string `json:"tenantId,omitempty"`
}

// Entity represents an entity in a message (mention, etc.).
type Entity struct {
	Type      string          `json:"type"`
	Mentioned *ChannelAccount `json:"mentioned,omitempty"`
	Text      string          `json:"text,omitempty"`
}

// SendActivity sends an activity to a conversation via the Bot Framework REST API.
func (c *Client) SendActivity(serviceURL, conversationID string, activity Activity) (*Activity, error) {
	token, err := c.GetBotToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	endpoint := fmt.Sprintf("%sv3/conversations/%s/activities", normalizeServiceURL(serviceURL), conversationID)

	body, err := json.Marshal(activity)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send activity: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("send activity failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	var result Activity
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Team represents a Microsoft Teams team.
type Team struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
}

// Channel represents a Microsoft Teams channel.
type Channel struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
}

// ListTeams lists all teams accessible by the app.
func (c *Client) ListTeams() ([]Team, error) {
	token, err := c.GetGraphToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get graph token: %w", err)
	}

	responseBody, err := c.graphRequest(http.MethodGet, "/teams", nil, token.AccessToken)
	if err != nil {
		return nil, err
	}

	var result struct {
		Value []Team `json:"value"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse teams response: %w", err)
	}

	return result.Value, nil
}

// ListTeamChannels lists channels in a specific team.
func (c *Client) ListTeamChannels(teamID string) ([]Channel, error) {
	token, err := c.GetGraphToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get graph token: %w", err)
	}

	endpoint := fmt.Sprintf("/teams/%s/channels", teamID)
	responseBody, err := c.graphRequest(http.MethodGet, endpoint, nil, token.AccessToken)
	if err != nil {
		return nil, err
	}

	var result struct {
		Value []Channel `json:"value"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse channels response: %w", err)
	}

	return result.Value, nil
}

// GetChannel retrieves a specific channel by ID.
func (c *Client) GetChannel(teamID, channelID string) (*Channel, error) {
	token, err := c.GetGraphToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get graph token: %w", err)
	}

	endpoint := fmt.Sprintf("/teams/%s/channels/%s", teamID, channelID)
	responseBody, err := c.graphRequest(http.MethodGet, endpoint, nil, token.AccessToken)
	if err != nil {
		return nil, err
	}

	var channel Channel
	if err := json.Unmarshal(responseBody, &channel); err != nil {
		return nil, fmt.Errorf("failed to parse channel response: %w", err)
	}

	return &channel, nil
}

// ChannelInfo contains channel information resolved from the Graph API.
type ChannelInfo struct {
	ID          string
	DisplayName string
	TeamID      string
	TeamName    string
}

// FindChannelByID searches all accessible teams for a channel matching the given ID.
func (c *Client) FindChannelByID(channelID string) (*ChannelInfo, error) {
	teams, err := c.ListTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	for _, team := range teams {
		channels, err := c.ListTeamChannels(team.ID)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			if channel.ID == channelID {
				return &ChannelInfo{
					ID:          channel.ID,
					DisplayName: channel.DisplayName,
					TeamID:      team.ID,
					TeamName:    team.DisplayName,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("channel %s not found in any accessible team", channelID)
}

func (c *Client) graphRequest(method, endpoint string, body io.Reader, accessToken string) ([]byte, error) {
	fullURL := graphAPIBase + endpoint

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("graph request failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func (c *Client) botTokenURL() string {
	tenantID := c.TenantID
	if tenantID == "" {
		tenantID = "botframework.com"
	}

	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
}

func (c *Client) graphTokenURL() string {
	tenantID := c.TenantID
	if tenantID == "" {
		// client_credentials requires a specific tenant.
		// This should not happen since tenantId is required.
		tenantID = "organizations"
	}

	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
}

func normalizeServiceURL(serviceURL string) string {
	if serviceURL == "" {
		return "https://smba.trafficmanager.net/teams/"
	}

	if serviceURL[len(serviceURL)-1] != '/' {
		return serviceURL + "/"
	}

	return serviceURL
}

// JWTValidator validates Bot Framework JWT tokens.
type JWTValidator struct {
	appID     string
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	lastFetch time.Time
}

// NewJWTValidator creates a new JWT validator for a given app ID.
func NewJWTValidator(appID string) *JWTValidator {
	return &JWTValidator{
		appID: appID,
		keys:  make(map[string]*rsa.PublicKey),
	}
}

// ValidateToken validates a Bot Framework JWT token.
func (v *JWTValidator) ValidateToken(tokenString string) (*jwt.Token, error) {
	if err := v.ensureKeys(); err != nil {
		return nil, fmt.Errorf("failed to fetch signing keys: %w", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		v.mu.RLock()
		key, exists := v.keys[kid]
		v.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("signing key not found for kid: %s", kid)
		}

		return key, nil
	}, jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(v.appID),
		jwt.WithIssuer("https://api.botframework.com"),
	)

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return token, nil
}

func (v *JWTValidator) ensureKeys() error {
	v.mu.RLock()
	if len(v.keys) > 0 && time.Since(v.lastFetch) < 24*time.Hour {
		v.mu.RUnlock()
		return nil
	}
	v.mu.RUnlock()

	return v.fetchKeys()
}

// OpenIDConfiguration represents the OpenID Connect discovery document.
type OpenIDConfiguration struct {
	JWKSURI string `json:"jwks_uri"`
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key.
type JWK struct {
	KID string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Kty string `json:"kty"`
}

func (v *JWTValidator) fetchKeys() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check after acquiring write lock
	if len(v.keys) > 0 && time.Since(v.lastFetch) < 24*time.Hour {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch OpenID configuration
	resp, err := client.Get(botFrameworkOpenIDURL)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenID config: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read OpenID config: %w", err)
	}

	var config OpenIDConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return fmt.Errorf("failed to parse OpenID config: %w", err)
	}

	// Fetch JWKS
	resp, err = client.Get(config.JWKSURI)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Parse RSA public keys
	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}

		key, err := parseRSAPublicKey(jwk)
		if err != nil {
			continue
		}

		keys[jwk.KID] = key
	}

	v.keys = keys
	v.lastFetch = time.Now()
	return nil
}

func parseRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := jwt.NewParser().DecodeSegment(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := jwt.NewParser().DecodeSegment(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
