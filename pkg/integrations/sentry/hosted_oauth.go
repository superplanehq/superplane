package sentry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSetupReturnCookie = "sp_integration_setup_return"
const sentryAppSetupStateCookie = "sentry_app_setup_state"
const sentryAppJWTGrantType = "urn:sentry:params:oauth:grant-type:jwt-bearer"

type sentryAppAuthorizationRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// InstallationGrant is the one-time install payload Sentry sends on
// installation.created and on the Redirect URL.
type InstallationGrant struct {
	Code    string
	UUID    string
	OrgSlug string
}

type sentryAppAuthorizationResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

func (s *Sentry) afterHostedAppSetup(ctx core.HTTPRequestContext) {
	metadata, ok := decodeHostedMetadata(ctx)
	if !ok {
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if metadata.InstallationUUID != "" {
		redirectToSetupReturn(ctx)
		return
	}

	if errParam := ctx.Request.URL.Query().Get("error"); errParam != "" {
		ctx.Logger.Errorf("Sentry app install error: %s", errParam)
		http.Error(ctx.Response, "authorization was denied", http.StatusBadRequest)
		return
	}

	if !hostedCallbackStateMatches(ctx.Request, metadata.State) {
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	code, installationUUID, orgSlug := sentryAppSetupQuery(ctx.Request)
	if installationUUID != "" {
		app, ok := HostedAppFromEnv()
		if !ok {
			http.Error(ctx.Response, "hosted Sentry app is not configured", http.StatusNotFound)
			return
		}
		if err := s.completeHostedAppInstall(ctx, app, metadata, installationUUID, orgSlug, code); err != nil {
			ctx.Logger.Errorf("failed to complete Sentry app install: %v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
		}
		clearSentryAppSetupStateCookie(ctx.Response)
		redirectToSetupReturn(ctx)
		return
	}

	bound, err := s.bindReadyHostedInstallIfPresent(core.SyncContext{
		Logger:          ctx.Logger,
		HTTP:            ctx.HTTP,
		Integration:     ctx.Integration,
		BaseURL:         ctx.BaseURL,
		WebhooksBaseURL: ctx.WebhooksBaseURL,
		OrganizationID:  ctx.OrganizationID,
	}, metadata)
	if err != nil {
		ctx.Logger.Errorf("failed to bind existing Sentry install: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}
	if bound {
		redirectToSetupReturn(ctx)
		return
	}
	http.Error(ctx.Response, "missing installation", http.StatusBadRequest)
}

func (s *Sentry) completeHostedAppInstall(
	ctx core.HTTPRequestContext,
	app HostedApp,
	metadata Metadata,
	installationUUID, orgSlug, code string,
) error {
	tokens, rememberedOrgSlug, err := resolveHostedAppTokens(ctx.HTTP, app, installationUUID, code, metadata.InstallationUUID)
	if err != nil {
		return err
	}
	if orgSlug == "" {
		orgSlug = rememberedOrgSlug
	}
	return s.finishHostedInstall(ctx, metadata, tokens, installationUUID, orgSlug)
}

func (s *Sentry) finishHostedInstall(
	ctx core.HTTPRequestContext,
	metadata Metadata,
	tokens *sentryAppAuthorizationResponse,
	installationUUID, orgSlug string,
) error {
	return s.adoptHostedInstall(core.SyncContext{
		HTTP:            ctx.HTTP,
		Logger:          ctx.Logger,
		Integration:     ctx.Integration,
		BaseURL:         ctx.BaseURL,
		WebhooksBaseURL: ctx.WebhooksBaseURL,
		OrganizationID:  ctx.OrganizationID,
	}, metadata, tokens, installationUUID, orgSlug)
}

func (s *Sentry) adoptHostedInstall(
	ctx core.SyncContext,
	metadata Metadata,
	tokens *sentryAppAuthorizationResponse,
	installationUUID, orgSlug string,
) error {
	if err := ctx.Integration.SetSecret(SecretAccessToken, []byte(tokens.Token)); err != nil {
		return fmt.Errorf("store Sentry access token: %w", err)
	}
	if tokens.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(SecretRefreshToken, []byte(tokens.RefreshToken)); err != nil {
			return fmt.Errorf("store Sentry refresh token: %w", err)
		}
	}

	metadata.HostedApp = true
	metadata.InstallationUUID = installationUUID
	metadata.TokenExpiresAt = tokens.ExpiresAt
	ctx.Integration.SetMetadata(metadata)

	client := NewAPIClient(ctx.HTTP, DefaultBaseURL, tokens.Token)
	if orgSlug != "" {
		client.orgSlug = orgSlug
	} else if inferred := orgSlugFromBaseURL(optionalConfig(ctx.Integration, "baseUrl")); inferred != "" {
		client.orgSlug = inferred
	}

	if client.orgSlug == "" {
		organizations, err := client.ListOrganizations()
		if err != nil {
			return fmt.Errorf("list Sentry organizations after install: %w", err)
		}
		if len(organizations) == 0 {
			return fmt.Errorf("list Sentry organizations after install: no organizations")
		}
		client.orgSlug = organizations[0].Slug
	}

	if err := s.populateMetadataFromOrg(ctx, client); err != nil {
		return fmt.Errorf("load Sentry organization after install: %w", err)
	}

	bound := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &bound)
	bound.HostedApp = true
	bound.State = metadata.State
	bound.InstallationUUID = installationUUID
	bound.StartedByUserID = metadata.StartedByUserID
	bound.SetupReturnPath = metadata.SetupReturnPath
	bound.TokenExpiresAt = tokens.ExpiresAt
	ctx.Integration.SetMetadata(bound)
	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func tokensForKnownHostedInstall(httpCtx core.HTTPContext, app HostedApp, install hostedSentryInstall) (*sentryAppAuthorizationResponse, error) {
	if strings.TrimSpace(install.AccessToken) != "" && !hostedTokenExpired(install.TokenExpiresAt) {
		return &sentryAppAuthorizationResponse{
			Token:        install.AccessToken,
			RefreshToken: install.RefreshToken,
			ExpiresAt:    install.TokenExpiresAt,
		}, nil
	}
	return mintSentryAppToken(httpCtx, app, install.InstallationUUID)
}

func exchangeSentryAppCode(httpCtx core.HTTPContext, app HostedApp, installationUUID, code string) (*sentryAppAuthorizationResponse, error) {
	return postSentryAppAuthorization(httpCtx, installationUUID, sentryAppAuthorizationRequest{
		GrantType:    "authorization_code",
		Code:         code,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
	}, nil)
}

func refreshSentryAppToken(httpCtx core.HTTPContext, app HostedApp, installationUUID, refreshToken string) (*sentryAppAuthorizationResponse, error) {
	return postSentryAppAuthorization(httpCtx, installationUUID, sentryAppAuthorizationRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
	}, nil)
}

func mintSentryAppToken(httpCtx core.HTTPContext, app HostedApp, installationUUID string) (*sentryAppAuthorizationResponse, error) {
	signed, err := sentryAppJWT(app)
	if err != nil {
		return nil, err
	}
	return postSentryAppAuthorization(
		httpCtx,
		installationUUID,
		sentryAppAuthorizationRequest{GrantType: sentryAppJWTGrantType},
		map[string]string{"Authorization": "Bearer " + signed},
	)
}

func sentryAppJWT(app HostedApp) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": app.ClientID,
		"sub": app.ClientID,
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		"jti": uuid.NewString(),
	})
	return token.SignedString([]byte(app.ClientSecret))
}

func resolveHostedAppTokens(
	httpCtx core.HTTPContext,
	app HostedApp,
	installationUUID, code, knownInstallationUUID string,
) (*sentryAppAuthorizationResponse, string, error) {
	orgSlug := ""
	if unclaimed := takeUnclaimedHostedInstall(installationUUID, code); unclaimed != nil {
		if unclaimed.Organization != nil {
			orgSlug = unclaimed.Organization.Slug
		}
		if strings.TrimSpace(unclaimed.AccessToken) != "" {
			return &sentryAppAuthorizationResponse{
				Token:        unclaimed.AccessToken,
				RefreshToken: unclaimed.RefreshToken,
				ExpiresAt:    unclaimed.TokenExpiresAt,
			}, orgSlug, nil
		}
	}

	tokens, err := hostedAppTokensFromSentry(httpCtx, app, installationUUID, code, knownInstallationUUID)
	if err != nil {
		return nil, "", err
	}
	return tokens, orgSlug, nil
}

func hostedAppTokensFromSentry(
	httpCtx core.HTTPContext,
	app HostedApp,
	installationUUID, code, knownInstallationUUID string,
) (*sentryAppAuthorizationResponse, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	code = strings.TrimSpace(code)
	alreadyBound := strings.TrimSpace(knownInstallationUUID) != "" && knownInstallationUUID == installationUUID

	if code != "" {
		tokens, err := exchangeSentryAppCode(httpCtx, app, installationUUID, code)
		if err == nil {
			return tokens, nil
		}
		if !alreadyBound {
			return nil, fmt.Errorf("exchange Sentry app code: %w", err)
		}
		minted, mintErr := mintSentryAppToken(httpCtx, app, installationUUID)
		if mintErr != nil {
			return nil, fmt.Errorf("exchange Sentry app code: %w", err)
		}
		return minted, nil
	}

	if alreadyBound {
		return mintSentryAppToken(httpCtx, app, installationUUID)
	}
	return nil, fmt.Errorf("Sentry app authorization code is required")
}

// RememberHostedInstallGrant exchanges a webhook grant and stores tokens until
// the authenticated setup callback presents the same code.
func RememberHostedInstallGrant(httpCtx core.HTTPContext, app HostedApp, grant InstallationGrant) error {
	if strings.TrimSpace(grant.UUID) == "" {
		return fmt.Errorf("installation is required")
	}
	code := strings.TrimSpace(grant.Code)
	if code == "" {
		return fmt.Errorf("installation grant code is required")
	}

	install := hostedSentryInstall{
		InstallationUUID: grant.UUID,
		Code:             code,
	}
	if grant.OrgSlug != "" {
		install.Organization = &OrganizationSummary{Slug: grant.OrgSlug}
	}

	tokens, err := exchangeSentryAppCode(httpCtx, app, grant.UUID, code)
	if err == nil {
		install.AccessToken = tokens.Token
		install.RefreshToken = tokens.RefreshToken
		install.TokenExpiresAt = tokens.ExpiresAt
	}
	rememberUnclaimedHostedInstall(install)
	return nil
}

func hostedCallbackState(request *http.Request) string {
	if request == nil {
		return ""
	}
	if state := strings.TrimSpace(request.URL.Query().Get("state")); state != "" {
		return state
	}
	return sentryAppSetupStateFromCookie(request)
}

func hostedCallbackStateMatches(request *http.Request, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	return hostedCallbackState(request) == expected
}

func postSentryAppAuthorization(
	httpCtx core.HTTPContext,
	installationUUID string,
	request sentryAppAuthorizationRequest,
	headers map[string]string,
) (*sentryAppAuthorizationResponse, error) {
	if httpCtx == nil {
		return nil, fmt.Errorf("HTTP context is required")
	}
	if strings.TrimSpace(installationUUID) == "" {
		return nil, fmt.Errorf("installation is required")
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(
		"%s/api/0/sentry-app-installations/%s/authorizations/",
		DefaultBaseURL,
		url.PathEscape(installationUUID),
	)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(encoded)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	response, err := httpCtx.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("Sentry app authorization failed: status %d", response.StatusCode)
	}

	var tokens sentryAppAuthorizationResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tokens.Token) == "" {
		return nil, fmt.Errorf("Sentry app authorization returned no token")
	}
	return &tokens, nil
}

func sentryAppSetupQuery(request *http.Request) (code, installationUUID, orgSlug string) {
	if request == nil {
		return "", "", ""
	}
	query := request.URL.Query()
	code = strings.TrimSpace(query.Get("code"))
	installationUUID = firstNonEmpty(
		strings.TrimSpace(query.Get("installationId")),
		strings.TrimSpace(query.Get("installation_id")),
	)
	orgSlug = firstNonEmpty(
		strings.TrimSpace(query.Get("orgSlug")),
		strings.TrimSpace(query.Get("org_slug")),
	)
	return code, installationUUID, orgSlug
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// ParseInstallationCreatedGrant reads the grant Sentry sends on
// installation.created. SuperPlane uses it when the browser Redirect URL
// never completed.
func ParseInstallationCreatedGrant(resource string, body []byte) (InstallationGrant, bool) {
	if strings.TrimSpace(resource) != "installation" {
		return InstallationGrant{}, false
	}

	var payload struct {
		Action string `json:"action"`
		Data   struct {
			Installation struct {
				Code         string `json:"code"`
				UUID         string `json:"uuid"`
				Organization struct {
					Slug string `json:"slug"`
				} `json:"organization"`
			} `json:"installation"`
		} `json:"data"`
		Installation struct {
			UUID string `json:"uuid"`
		} `json:"installation"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return InstallationGrant{}, false
	}
	if strings.TrimSpace(payload.Action) != "created" {
		return InstallationGrant{}, false
	}

	grant := InstallationGrant{
		Code:    strings.TrimSpace(payload.Data.Installation.Code),
		UUID:    firstNonEmpty(strings.TrimSpace(payload.Data.Installation.UUID), strings.TrimSpace(payload.Installation.UUID)),
		OrgSlug: strings.TrimSpace(payload.Data.Installation.Organization.Slug),
	}
	if grant.UUID == "" {
		return InstallationGrant{}, false
	}
	return grant, true
}

// ParseInstallationDeletedUUID reads the installation UUID Sentry sends on
// installation.deleted.
func ParseInstallationDeletedUUID(resource string, body []byte) (string, bool) {
	if strings.TrimSpace(resource) != "installation" {
		return "", false
	}

	var payload struct {
		Action       string `json:"action"`
		Installation struct {
			UUID string `json:"uuid"`
		} `json:"installation"`
		Data struct {
			Installation struct {
				UUID string `json:"uuid"`
			} `json:"installation"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	if strings.TrimSpace(payload.Action) != "deleted" {
		return "", false
	}
	uuid := firstNonEmpty(
		strings.TrimSpace(payload.Data.Installation.UUID),
		strings.TrimSpace(payload.Installation.UUID),
	)
	if uuid == "" {
		return "", false
	}
	return uuid, true
}

func decodeHostedMetadata(ctx core.HTTPRequestContext) (Metadata, bool) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("failed to decode Sentry metadata: %v", err)
		return metadata, false
	}
	return metadata, true
}

type persistentIntegration interface {
	Persist() error
}

func persistIntegration(integration core.IntegrationContext) error {
	persister, ok := integration.(persistentIntegration)
	if !ok {
		return nil
	}
	return persister.Persist()
}

func persistIntegrationBeforeRedirect(ctx core.HTTPRequestContext) {
	if err := persistIntegration(ctx.Integration); err != nil {
		ctx.Logger.Errorf("failed to persist Sentry integration before redirect: %v", err)
	}
}

func redirectToSetupReturn(ctx core.HTTPRequestContext) {
	persistIntegrationBeforeRedirect(ctx)
	location := hostedCallbackLocation(ctx)
	if hostedCallbackReturnPath(ctx) != "" {
		clearIntegrationSetupReturnCookie(ctx.Response)
	}
	http.Redirect(ctx.Response, ctx.Request, location, http.StatusSeeOther)
}

func hostedCallbackLocation(ctx core.HTTPRequestContext) string {
	settings := fmt.Sprintf(
		"%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String(),
	)
	returnPath := hostedCallbackReturnPath(ctx)
	if returnPath == "" {
		return settings
	}
	return strings.TrimRight(ctx.BaseURL, "/") + returnPath
}

func hostedCallbackReturnPath(ctx core.HTTPRequestContext) string {
	if path := setupReturnPathFromMetadata(ctx); path != "" {
		return path
	}
	if path := integrationSetupReturnPath(ctx.Request); path != "" {
		return path
	}
	return ""
}

func setupReturnPathFromMetadata(ctx core.HTTPRequestContext) string {
	if ctx.Integration == nil {
		return ""
	}
	metadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	return firstSafeSetupReturnPath(metadata.SetupReturnPath)
}

func firstSafeSetupReturnPath(paths ...string) string {
	for _, path := range paths {
		if isSafeIntegrationSetupReturnPath(path) {
			return path
		}
	}
	return ""
}

func integrationSetupReturnPath(request *http.Request) string {
	if request == nil {
		return ""
	}
	cookie, err := request.Cookie(integrationSetupReturnCookie)
	if err != nil || cookie == nil {
		return ""
	}
	path, err := url.QueryUnescape(strings.TrimSpace(cookie.Value))
	if err != nil || !isSafeIntegrationSetupReturnPath(path) {
		return ""
	}
	return path
}

func isSafeIntegrationSetupReturnPath(path string) bool {
	if path == "" || strings.Contains(path, "://") || strings.ContainsAny(path, "\\\t\r\n ") {
		return false
	}

	pathname, _, _ := strings.Cut(path, "?")
	if pathname == "/onboarding" {
		return true
	}
	if !strings.HasPrefix(pathname, "/") || strings.HasPrefix(pathname, "//") {
		return false
	}

	rest := strings.TrimPrefix(pathname, "/")
	organization, after, ok := strings.Cut(rest, "/")
	return ok && organization != "" && after != ""
}

func clearIntegrationSetupReturnCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     integrationSetupReturnCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func clearSentryAppSetupStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sentryAppSetupStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func setSentryAppSetupStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sentryAppSetupStateCookie,
		Value:    url.QueryEscape(state),
		Path:     "/",
		MaxAge:   int((20 * time.Minute).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func sentryAppSetupStateFromCookie(request *http.Request) string {
	if request == nil {
		return ""
	}
	cookie, err := request.Cookie(sentryAppSetupStateCookie)
	if err != nil || cookie == nil {
		return ""
	}
	state, err := url.QueryUnescape(strings.TrimSpace(cookie.Value))
	if err != nil {
		return ""
	}
	return state
}

func hostedTokenExpired(expiresAt string) bool {
	expiresAt = strings.TrimSpace(expiresAt)
	if expiresAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, expiresAt)
		if err != nil {
			return false
		}
	}
	return !parsed.After(time.Now().Add(5 * time.Minute))
}
