package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	AuthorizationCodeTTL = 10 * time.Minute
	RefreshTokenTTL      = 30 * 24 * time.Hour
)

type Client struct {
	ID           string
	Name         string
	RedirectURIs []string
}

type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	ResponseType        string
}

type TokenIssue struct {
	UserID    uuid.UUID
	OrgID     uuid.UUID
	FactoryID uuid.UUID
	ClientID  string
	Resource  string
	Scopes    []string
}

type OAuthError struct {
	Code        string
	Description string
	Status      int
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return e.Description
	}
	return e.Code
}

func ParseAuthorizeRequest(values url.Values, resource string) (*AuthorizeRequest, *OAuthError) {
	req := &AuthorizeRequest{
		ClientID:            strings.TrimSpace(values.Get("client_id")),
		RedirectURI:         strings.TrimSpace(values.Get("redirect_uri")),
		State:               values.Get("state"),
		CodeChallenge:       strings.TrimSpace(values.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(values.Get("code_challenge_method")),
		Resource:            strings.TrimSpace(values.Get("resource")),
		ResponseType:        strings.TrimSpace(values.Get("response_type")),
	}
	if req.ResponseType == "" {
		req.ResponseType = "code"
	}
	if req.ClientID == "" {
		return nil, &OAuthError{Code: "invalid_request", Description: "client_id is required", Status: http.StatusBadRequest}
	}
	if req.RedirectURI == "" {
		return nil, &OAuthError{Code: "invalid_request", Description: "redirect_uri is required", Status: http.StatusBadRequest}
	}
	if req.ResponseType != "code" {
		return nil, &OAuthError{Code: "unsupported_response_type", Description: "response_type must be code", Status: http.StatusBadRequest}
	}
	if !strings.EqualFold(req.CodeChallengeMethod, "S256") || req.CodeChallenge == "" {
		return nil, &OAuthError{Code: "invalid_request", Description: "PKCE S256 is required", Status: http.StatusBadRequest}
	}
	req.CodeChallengeMethod = "S256"
	if req.Resource != "" && req.Resource != resource {
		return nil, &OAuthError{Code: "invalid_target", Description: "resource does not match this server", Status: http.StatusBadRequest}
	}
	if req.Resource == "" {
		req.Resource = resource
	}
	return req, nil
}

func ResolveClient(ctx context.Context, tx *gorm.DB, httpClient mcp.HTTPDoer, clientID, redirectURI string) (*Client, *OAuthError) {
	client, err := lookupClient(ctx, tx, httpClient, clientID)
	if err != nil {
		return nil, err
	}
	if !redirectAllowed(client, redirectURI) {
		return nil, &OAuthError{Code: "invalid_request", Description: "redirect_uri is not allowed", Status: http.StatusBadRequest}
	}
	return client, nil
}

func ClientDisplayName(tx *gorm.DB, clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return models.DefaultMCPClientName
	}
	if clientID == LocalClientID {
		return LocalClientName
	}
	stored := ""
	if row, err := models.FindMCPOAuthClient(tx, clientID); err == nil {
		stored = row.ClientName
	}
	return mcp.ClientDisplayName(stored, clientID, models.DefaultMCPClientName)
}

func lookupClient(ctx context.Context, tx *gorm.DB, httpClient mcp.HTTPDoer, clientID string) (*Client, *OAuthError) {
	if clientID == LocalClientID {
		return &Client{ID: LocalClientID, Name: LocalClientName, RedirectURIs: append([]string{}, CursorRedirectURIs...)}, nil
	}
	if stored, err := models.FindMCPOAuthClient(tx, clientID); err == nil {
		return &Client{ID: stored.ClientID, Name: stored.ClientName, RedirectURIs: []string(stored.RedirectURIs)}, nil
	} else if !errors.Is(err, models.ErrMCPOAuthClientNotFound) {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load client", Status: http.StatusInternalServerError}
	}
	if strings.HasPrefix(clientID, "https://") {
		return fetchClientMetadata(ctx, httpClient, clientID)
	}
	return nil, &OAuthError{Code: "invalid_client", Description: "client is not registered", Status: http.StatusBadRequest}
}

func redirectAllowed(client *Client, uri string) bool {
	if slices.Contains(CursorRedirectURIs, uri) {
		return true
	}
	if slices.Contains(client.RedirectURIs, uri) {
		return true
	}
	return ephemeralLoopbackRedirectAllowed(client.RedirectURIs, uri)
}

func ephemeralLoopbackRedirectAllowed(registered []string, requested string) bool {
	requestedRedirect, ok := parseLoopbackHTTPRedirect(requested)
	if !ok {
		return false
	}
	for _, candidate := range registered {
		if slices.Contains(CursorRedirectURIs, candidate) {
			continue
		}
		registeredRedirect, ok := parseLoopbackHTTPRedirect(candidate)
		if !ok {
			continue
		}
		if requestedRedirect == registeredRedirect {
			return true
		}
	}
	return false
}

type loopbackHTTPRedirect struct {
	host  string
	path  string
	query string
}

func parseLoopbackHTTPRedirect(raw string) (loopbackHTTPRedirect, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return loopbackHTTPRedirect{}, false
	}
	if parsed.User != nil || redirectHasFragment(raw, parsed) {
		return loopbackHTTPRedirect{}, false
	}
	host := parsed.Hostname()
	if !exactLoopbackHostname(host) || !numericOrDefaultHTTPPort(parsed.Host) {
		return loopbackHTTPRedirect{}, false
	}
	return loopbackHTTPRedirect{
		host:  strings.ToLower(host),
		path:  parsed.EscapedPath(),
		query: parsed.RawQuery,
	}, true
}

func redirectHasFragment(raw string, parsed *url.URL) bool {
	return parsed.Fragment != "" || parsed.RawFragment != "" || strings.Contains(raw, "#")
}

func exactLoopbackHostname(host string) bool {
	switch strings.ToLower(host) {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
}

func numericOrDefaultHTTPPort(host string) bool {
	_, port, err := net.SplitHostPort(host)
	if err != nil {
		return hostHasNoPort(host)
	}
	return allDigits(port)
}

func hostHasNoPort(host string) bool {
	if strings.HasPrefix(host, "[") {
		return strings.HasSuffix(host, "]")
	}
	return host != "" && !strings.Contains(host, ":")
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func fetchClientMetadata(ctx context.Context, httpClient mcp.HTTPDoer, metadataURL string) (*Client, *OAuthError) {
	if err := mcp.ValidatePublicHTTPSURL(metadataURL); err != nil {
		return nil, &OAuthError{Code: "invalid_client", Description: "client metadata URL is not valid", Status: http.StatusBadRequest}
	}
	var document struct {
		ClientID     string   `json:"client_id"`
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := mcp.GetJSON(ctx, httpClient, metadataURL, &document); err != nil {
		return nil, &OAuthError{Code: "invalid_client", Description: "failed to load client metadata", Status: http.StatusBadRequest}
	}
	if document.ClientID != "" && document.ClientID != metadataURL {
		return nil, &OAuthError{Code: "invalid_client", Description: "client_id does not match the metadata URL", Status: http.StatusBadRequest}
	}
	return &Client{
		ID:           metadataURL,
		Name:         strings.TrimSpace(document.ClientName),
		RedirectURIs: document.RedirectURIs,
	}, nil
}

func CreateAuthorizationCode(tx *gorm.DB, req *AuthorizeRequest, userID, orgID, factoryID uuid.UUID) (string, error) {
	raw, err := mcp.RandomToken()
	if err != nil {
		return "", err
	}
	err = models.CreateMCPOAuthCode(tx, &models.MCPOAuthCode{
		CodeHash:            crypto.HashToken(raw),
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Resource:            req.Resource,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		UserID:              userID,
		OrganizationID:      orgID,
		FactoryID:           factoryID,
		Scopes:              datatypes.NewJSONSlice(GrantedScopes),
		ExpiresAt:           time.Now().Add(AuthorizationCodeTTL),
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func ExchangeAuthorizationCode(tx *gorm.DB, values url.Values, resource string) (*TokenIssue, string, *OAuthError) {
	if values.Get("grant_type") != "authorization_code" {
		return nil, "", &OAuthError{Code: "unsupported_grant_type", Description: "grant_type must be authorization_code", Status: http.StatusBadRequest}
	}
	code := strings.TrimSpace(values.Get("code"))
	verifier := strings.TrimSpace(values.Get("code_verifier"))
	redirectURI := strings.TrimSpace(values.Get("redirect_uri"))
	clientID := strings.TrimSpace(values.Get("client_id"))
	requestedResource := strings.TrimSpace(values.Get("resource"))
	if code == "" || verifier == "" || redirectURI == "" || clientID == "" {
		return nil, "", &OAuthError{Code: "invalid_request", Description: "code, code_verifier, client_id, and redirect_uri are required", Status: http.StatusBadRequest}
	}
	if requestedResource != "" && requestedResource != resource {
		return nil, "", &OAuthError{Code: "invalid_target", Description: "resource does not match this server", Status: http.StatusBadRequest}
	}

	stored, err := models.FindMCPOAuthCode(tx, crypto.HashToken(code), time.Now())
	if err != nil {
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "authorization code is not valid", Status: http.StatusBadRequest}
	}
	if stored.ClientID != clientID || stored.RedirectURI != redirectURI || stored.Resource != resource {
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "authorization code is not valid", Status: http.StatusBadRequest}
	}
	if stored.CodeChallengeMethod != "S256" || !mcp.VerifyS256(verifier, stored.CodeChallenge) {
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "PKCE verification failed", Status: http.StatusBadRequest}
	}
	if err := models.DeleteMCPOAuthCode(tx, stored); err != nil {
		return nil, "", &OAuthError{Code: "server_error", Description: "failed to issue tokens", Status: http.StatusInternalServerError}
	}
	if UserAccountBlocked(tx, stored.UserID) {
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "authorization code is not valid", Status: http.StatusBadRequest}
	}

	refresh, err := storeRefreshToken(tx, TokenIssue{
		UserID:    stored.UserID,
		OrgID:     stored.OrganizationID,
		FactoryID: stored.FactoryID,
		ClientID:  stored.ClientID,
		Resource:  stored.Resource,
		Scopes:    []string(stored.Scopes),
	})
	if err != nil {
		return nil, "", &OAuthError{Code: "server_error", Description: "failed to issue refresh token", Status: http.StatusInternalServerError}
	}
	return &TokenIssue{
		UserID:    stored.UserID,
		OrgID:     stored.OrganizationID,
		FactoryID: stored.FactoryID,
		ClientID:  stored.ClientID,
		Resource:  stored.Resource,
		Scopes:    []string(stored.Scopes),
	}, refresh, nil
}

func RefreshTokens(tx *gorm.DB, values url.Values, resource string) (*TokenIssue, string, *OAuthError) {
	if values.Get("grant_type") != "refresh_token" {
		return nil, "", &OAuthError{Code: "unsupported_grant_type", Description: "grant_type must be refresh_token", Status: http.StatusBadRequest}
	}
	raw := strings.TrimSpace(values.Get("refresh_token"))
	clientID := strings.TrimSpace(values.Get("client_id"))
	requestedResource := strings.TrimSpace(values.Get("resource"))
	if raw == "" || clientID == "" {
		return nil, "", &OAuthError{Code: "invalid_request", Description: "refresh_token and client_id are required", Status: http.StatusBadRequest}
	}
	if requestedResource != "" && requestedResource != resource {
		return nil, "", &OAuthError{Code: "invalid_target", Description: "resource does not match this server", Status: http.StatusBadRequest}
	}

	stored, err := models.FindMCPOAuthRefreshToken(tx, crypto.HashToken(raw), time.Now())
	if err != nil {
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "refresh token is not valid", Status: http.StatusBadRequest}
	}
	if stored.ClientID != clientID || stored.Resource != resource {
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "refresh token is not valid", Status: http.StatusBadRequest}
	}
	if UserAccountBlocked(tx, stored.UserID) {
		if err := models.DeleteMCPOAuthRefreshToken(tx, stored); err != nil {
			return nil, "", &OAuthError{Code: "server_error", Description: "failed to rotate refresh token", Status: http.StatusInternalServerError}
		}
		return nil, "", &OAuthError{Code: "invalid_grant", Description: "refresh token is not valid", Status: http.StatusBadRequest}
	}
	if err := models.DeleteMCPOAuthRefreshToken(tx, stored); err != nil {
		return nil, "", &OAuthError{Code: "server_error", Description: "failed to rotate refresh token", Status: http.StatusInternalServerError}
	}
	issue := TokenIssue{
		UserID:    stored.UserID,
		OrgID:     stored.OrganizationID,
		FactoryID: stored.FactoryID,
		ClientID:  stored.ClientID,
		Resource:  stored.Resource,
		Scopes:    []string(stored.Scopes),
	}
	refresh, err := storeRefreshToken(tx, issue)
	if err != nil {
		return nil, "", &OAuthError{Code: "server_error", Description: "failed to rotate refresh token", Status: http.StatusInternalServerError}
	}
	return &issue, refresh, nil
}

func RegisterClient(tx *gorm.DB, body []byte) (*Client, *OAuthError) {
	var payload struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &OAuthError{Code: "invalid_client_metadata", Description: "request body is not valid JSON", Status: http.StatusBadRequest}
	}
	if len(payload.RedirectURIs) == 0 {
		return nil, &OAuthError{Code: "invalid_client_metadata", Description: "redirect_uris is required", Status: http.StatusBadRequest}
	}
	clientID, err := mcp.RandomToken()
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to create client", Status: http.StatusInternalServerError}
	}
	name := strings.TrimSpace(payload.ClientName)
	if name == "" {
		name = "MCP client"
	}
	redirectURIs := append([]string{}, payload.RedirectURIs...)
	for _, uri := range CursorRedirectURIs {
		if !slices.Contains(redirectURIs, uri) {
			redirectURIs = append(redirectURIs, uri)
		}
	}
	stored, err := models.CreateMCPOAuthClient(tx, clientID, name, redirectURIs)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to create client", Status: http.StatusInternalServerError}
	}
	return &Client{ID: stored.ClientID, Name: stored.ClientName, RedirectURIs: []string(stored.RedirectURIs)}, nil
}

func storeRefreshToken(tx *gorm.DB, issue TokenIssue) (string, error) {
	raw, err := mcp.RandomToken()
	if err != nil {
		return "", err
	}
	err = models.CreateMCPOAuthRefreshToken(tx, &models.MCPOAuthRefreshToken{
		TokenHash:      crypto.HashToken(raw),
		ClientID:       issue.ClientID,
		UserID:         issue.UserID,
		OrganizationID: issue.OrgID,
		FactoryID:      issue.FactoryID,
		Resource:       issue.Resource,
		Scopes:         datatypes.NewJSONSlice(issue.Scopes),
		ExpiresAt:      time.Now().Add(RefreshTokenTTL),
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func AccessGrantIsActive(tx *gorm.DB, claims *AccessClaims, now time.Time) bool {
	if claims == nil {
		return false
	}
	ok, err := models.HasMCPOAuthRefreshTokenForClient(
		tx,
		claims.OrgID,
		claims.FactoryID,
		claims.UserID,
		claims.ClientID,
		now,
	)
	return err == nil && ok
}

func AuthorizationRedirect(redirectURI, code, state string) (string, error) {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("code", code)
	if state != "" {
		query.Set("state", state)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
