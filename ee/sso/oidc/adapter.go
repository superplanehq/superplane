package oidc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/superplanehq/superplane/ee/sso/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"golang.org/x/oauth2"
)

const providerCacheTTL = time.Hour

// Doer performs one outbound HTTP request. Production uses the SSRF-safe client.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter is the OIDC protocol implementation.
type Adapter struct {
	BaseURL   string
	Doer      Doer
	Encryptor crypto.Encryptor
	AllowHTTP func(context.Context) bool
	Now       func() time.Time

	mu    sync.Mutex
	cache map[string]cachedProvider
}

type cachedProvider struct {
	provider *oidc.Provider
	until    time.Time
}

type doerTransport struct {
	doer Doer
}

func (t doerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.doer.Do(req)
}

func (a *Adapter) Protocol() string { return models.SSOProtocolOIDC }

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Adapter) allowHTTP(ctx context.Context) bool {
	return a.AllowHTTP != nil && a.AllowHTTP(ctx)
}

func (a *Adapter) clientContext(ctx context.Context) context.Context {
	client := &http.Client{Transport: doerTransport{doer: a.Doer}}
	ctx = oidc.ClientContext(ctx, client)
	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

// ValidateConfig checks the issuer URL. Discovery runs when a sign-in starts.
func (a *Adapter) ValidateConfig(ctx context.Context, conn *models.SSOConnection) error {
	cfg, err := conn.OIDCConfig()
	if err != nil {
		return err
	}
	return validateIssuer(cfg.Issuer, a.allowHTTP(ctx))
}

func validateIssuer(issuer string, allowHTTP bool) error {
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("the issuer URL is not valid")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && allowHTTP {
		return nil
	}
	return fmt.Errorf("the issuer URL must use HTTPS")
}

// BeginLogin starts Authorization Code with PKCE.
func (a *Adapter) BeginLogin(ctx context.Context, conn *models.SSOConnection, req core.LoginRequest) (string, core.LoginState, error) {
	cfg, secret, err := a.client(ctx, conn)
	if err != nil {
		return "", core.LoginState{}, err
	}
	provider, err := a.provider(ctx, conn, cfg.Issuer)
	if err != nil {
		return "", core.LoginState{}, err
	}
	state, err := randomToken()
	if err != nil {
		return "", core.LoginState{}, err
	}
	nonce, err := randomToken()
	if err != nil {
		return "", core.LoginState{}, err
	}
	verifier := oauth2.GenerateVerifier()
	oauth := a.oauthConfig(conn, cfg, secret, provider)
	options := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	}
	if req.LoginHint != "" {
		options = append(options, oauth2.SetAuthURLParam("login_hint", req.LoginHint))
	}
	login := core.LoginState{
		ConnectionID: conn.ID,
		State:        state,
		Nonce:        nonce,
		Verifier:     verifier,
		Redirect:     req.Redirect,
		Test:         req.Test,
	}
	return oauth.AuthCodeURL(state, options...), login, nil
}

// HandleCallback exchanges the code and verifies the ID token.
func (a *Adapter) HandleCallback(ctx context.Context, conn *models.SSOConnection, r *http.Request, state core.LoginState) (*core.ExternalIdentity, error) {
	if r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != state.State {
		return nil, fmt.Errorf("state does not match")
	}
	if message := r.URL.Query().Get("error"); message != "" {
		return nil, fmt.Errorf("identity provider returned %s", message)
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("authorization code is missing")
	}

	cfg, secret, err := a.client(ctx, conn)
	if err != nil {
		return nil, err
	}
	provider, err := a.provider(ctx, conn, cfg.Issuer)
	if err != nil {
		return nil, err
	}
	oauth := a.oauthConfig(conn, cfg, secret, provider)
	token, err := oauth.Exchange(a.clientContext(ctx), code, oauth2.VerifierOption(state.Verifier))
	if err != nil {
		return nil, err
	}
	raw, _ := token.Extra("id_token").(string)
	if raw == "" {
		return nil, fmt.Errorf("identity provider did not return an ID token")
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(a.clientContext(ctx), raw)
	if err != nil {
		return nil, err
	}
	if idToken.Nonce != state.Nonce {
		return nil, fmt.Errorf("nonce does not match")
	}

	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	identity := identityFromClaims(cfg, idToken.Subject, raw, claims)
	if identity.Email == "" {
		if err := a.fillEmailFromUserInfo(a.clientContext(ctx), provider, token, cfg, idToken.Subject, identity); err != nil {
			return nil, err
		}
	}
	if identity.Subject == "" || identity.Email == "" {
		return nil, fmt.Errorf("the ID token has no subject or email")
	}
	return identity, nil
}

// BeginLogout builds an RP-initiated logout URL when the provider publishes one.
func (a *Adapter) BeginLogout(ctx context.Context, conn *models.SSOConnection, idTokenHint string) (string, bool, error) {
	if strings.TrimSpace(idTokenHint) == "" || !conn.IDPLogoutEnabled {
		return "", false, nil
	}
	cfg, err := conn.OIDCConfig()
	if err != nil {
		return "", false, err
	}
	provider, err := a.provider(ctx, conn, cfg.Issuer)
	if err != nil {
		return "", false, err
	}
	var metadata struct {
		EndSession string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&metadata); err != nil {
		return "", false, err
	}
	if metadata.EndSession == "" {
		return "", false, nil
	}
	endpoint, err := url.Parse(metadata.EndSession)
	if err != nil {
		return "", false, err
	}
	state, err := randomToken()
	if err != nil {
		return "", false, err
	}
	query := endpoint.Query()
	query.Set("id_token_hint", idTokenHint)
	query.Set("post_logout_redirect_uri", strings.TrimRight(a.BaseURL, "/")+"/login")
	query.Set("state", state)
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), true, nil
}

// VerifyLogoutToken checks an OIDC back-channel logout token.
func (a *Adapter) VerifyLogoutToken(ctx context.Context, conn *models.SSOConnection, raw string) (string, string, error) {
	cfg, err := conn.OIDCConfig()
	if err != nil {
		return "", "", err
	}
	provider, err := a.provider(ctx, conn, cfg.Issuer)
	if err != nil {
		return "", "", err
	}
	// A logout token may omit exp. Signature, issuer, and audience stay checked.
	// A present exp is checked below. iat limits how old the token can be.
	idToken, err := provider.Verifier(&oidc.Config{
		ClientID:        cfg.ClientID,
		SkipExpiryCheck: true,
	}).Verify(a.clientContext(ctx), raw)
	if err != nil {
		return "", "", err
	}
	now := a.now()
	if now.Sub(idToken.IssuedAt) > 5*time.Minute || idToken.IssuedAt.After(now.Add(time.Minute)) {
		return "", "", fmt.Errorf("logout token is too old")
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return "", "", err
	}
	if rawExp, ok := claims["exp"]; ok {
		if err := requireLogoutUnexpired(rawExp, now); err != nil {
			return "", "", err
		}
	}
	if _, hasNonce := claims["nonce"]; hasNonce {
		return "", "", fmt.Errorf("logout token must not include a nonce")
	}
	events, _ := claims["events"].(map[string]any)
	if _, ok := events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		return "", "", fmt.Errorf("logout token is missing the back-channel event")
	}
	sessionID, _ := claims["sid"].(string)
	subject := idToken.Subject
	if subject == "" {
		subject, _ = claims["sub"].(string)
	}
	if subject == "" && sessionID == "" {
		return "", "", fmt.Errorf("logout token needs a subject or a session id")
	}
	return subject, sessionID, nil
}

func requireLogoutUnexpired(raw any, now time.Time) error {
	seconds, ok := raw.(float64)
	if !ok {
		return fmt.Errorf("logout token is expired")
	}
	if time.Unix(int64(seconds), 0).Before(now) {
		return fmt.Errorf("logout token is expired")
	}
	return nil
}

func (a *Adapter) client(ctx context.Context, conn *models.SSOConnection) (models.OIDCConnectionConfig, string, error) {
	cfg, err := conn.OIDCConfig()
	if err != nil {
		return models.OIDCConnectionConfig{}, "", err
	}
	if err := validateIssuer(cfg.Issuer, a.allowHTTP(ctx)); err != nil {
		return models.OIDCConnectionConfig{}, "", err
	}
	if len(conn.ClientSecret) == 0 || a.Encryptor == nil {
		return models.OIDCConnectionConfig{}, "", fmt.Errorf("the client secret is missing")
	}
	plain, err := a.Encryptor.Decrypt(ctx, conn.ClientSecret, []byte(models.SSOClientSecretAssociatedData))
	if err != nil {
		return models.OIDCConnectionConfig{}, "", err
	}
	return cfg, string(plain), nil
}

func (a *Adapter) oauthConfig(conn *models.SSOConnection, cfg models.OIDCConnectionConfig, secret string, provider *oidc.Provider) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: secret,
		RedirectURL:  redirectURI(a.BaseURL, conn.ID.String()),
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}
}

func (a *Adapter) provider(ctx context.Context, conn *models.SSOConnection, issuer string) (*oidc.Provider, error) {
	key := conn.ID.String() + "|" + conn.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + issuer
	a.mu.Lock()
	if a.cache == nil {
		a.cache = map[string]cachedProvider{}
	}
	if cached, ok := a.cache[key]; ok && a.now().Before(cached.until) {
		a.mu.Unlock()
		return cached.provider, nil
	}
	a.mu.Unlock()

	provider, err := oidc.NewProvider(a.clientContext(ctx), issuer)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	prefix := conn.ID.String() + "|"
	for cachedKey := range a.cache {
		if strings.HasPrefix(cachedKey, prefix) {
			delete(a.cache, cachedKey)
		}
	}
	a.cache[key] = cachedProvider{provider: provider, until: a.now().Add(providerCacheTTL)}
	a.mu.Unlock()
	return provider, nil
}

func (a *Adapter) fillEmailFromUserInfo(ctx context.Context, provider *oidc.Provider, token *oauth2.Token, cfg models.OIDCConnectionConfig, subject string, identity *core.ExternalIdentity) error {
	info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return err
	}
	if info.Subject == "" || info.Subject != subject {
		return fmt.Errorf("userinfo subject does not match the ID token")
	}
	claims := map[string]any{}
	if err := info.Claims(&claims); err != nil {
		return err
	}
	if identity.Email == "" {
		identity.Email = claimString(claims, cfg.EmailClaim)
	}
	if identity.Name == "" {
		identity.Name = claimString(claims, cfg.NameClaim)
	}
	return nil
}

func claimString(claims map[string]any, name string) string {
	value, ok := claims[name]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func identityFromClaims(cfg models.OIDCConnectionConfig, verifiedSubject, raw string, claims map[string]any) *core.ExternalIdentity {
	subject := claimString(claims, cfg.SubjectClaim)
	if subject == "" && cfg.SubjectClaim == "sub" {
		subject = verifiedSubject
	}
	authTime := time.Time{}
	if value, ok := claims["auth_time"].(float64); ok {
		authTime = time.Unix(int64(value), 0)
	}
	verified, _ := claims["email_verified"].(bool)
	return &core.ExternalIdentity{
		Subject:       subject,
		IDPSubject:    verifiedSubject,
		Email:         claimString(claims, cfg.EmailClaim),
		EmailVerified: verified,
		Name:          claimString(claims, cfg.NameClaim),
		SessionID:     claimString(claims, "sid"),
		AuthTime:      authTime,
		RawIDToken:    raw,
		Claims:        claims,
	}
}

func redirectURI(baseURL, connectionID string) string {
	return strings.TrimRight(baseURL, "/") + "/auth/sso/" + connectionID + "/callback"
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// RedirectURI is the OIDC redirect URI for a connection.
func RedirectURI(baseURL, connectionID string) string {
	return redirectURI(baseURL, connectionID)
}
