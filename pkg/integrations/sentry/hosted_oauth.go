package sentry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const integrationSetupReturnCookie = "sp_integration_setup_return"
const sentryAppSetupStateCookie = "sentry_app_setup_state"

type sentryAppAuthorizationRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
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

	code := strings.TrimSpace(ctx.Request.URL.Query().Get("code"))
	installationUUID := strings.TrimSpace(ctx.Request.URL.Query().Get("installationId"))
	orgSlug := strings.TrimSpace(ctx.Request.URL.Query().Get("orgSlug"))
	if code == "" || installationUUID == "" {
		http.Error(ctx.Response, "missing installation", http.StatusBadRequest)
		return
	}

	app, ok := HostedAppFromEnv()
	if !ok {
		http.Error(ctx.Response, "hosted Sentry app is not configured", http.StatusNotFound)
		return
	}

	tokens, err := exchangeSentryAppCode(ctx.HTTP, app, installationUUID, code)
	if err != nil {
		ctx.Logger.Errorf("failed to exchange Sentry app code: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := ctx.Integration.SetSecret(SecretAccessToken, []byte(tokens.Token)); err != nil {
		ctx.Logger.Errorf("failed to store Sentry access token: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}
	if tokens.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(SecretRefreshToken, []byte(tokens.RefreshToken)); err != nil {
			ctx.Logger.Errorf("failed to store Sentry refresh token: %v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
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
		if err != nil || len(organizations) == 0 {
			ctx.Logger.Errorf("failed to list Sentry organizations after install: %v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
		}
		client.orgSlug = organizations[0].Slug
	}

	syncCtx := core.SyncContext{
		HTTP:            ctx.HTTP,
		Logger:          ctx.Logger,
		Integration:     ctx.Integration,
		BaseURL:         ctx.BaseURL,
		WebhooksBaseURL: ctx.WebhooksBaseURL,
	}
	if err := s.populateMetadataFromOrg(syncCtx, client); err != nil {
		ctx.Logger.Errorf("failed to load Sentry organization after install: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
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

	clearSentryAppSetupStateCookie(ctx.Response)
	redirectToSetupReturn(ctx)
}

func exchangeSentryAppCode(httpCtx core.HTTPContext, app HostedApp, installationUUID, code string) (*sentryAppAuthorizationResponse, error) {
	return authorizeSentryApp(httpCtx, installationUUID, sentryAppAuthorizationRequest{
		GrantType:    "authorization_code",
		Code:         code,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
	})
}

func refreshSentryAppToken(httpCtx core.HTTPContext, app HostedApp, installationUUID, refreshToken string) (*sentryAppAuthorizationResponse, error) {
	return authorizeSentryApp(httpCtx, installationUUID, sentryAppAuthorizationRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
	})
}

func authorizeSentryApp(
	httpCtx core.HTTPContext,
	installationUUID string,
	request sentryAppAuthorizationRequest,
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

func persistIntegrationBeforeRedirect(ctx core.HTTPRequestContext) {
	persister, ok := ctx.Integration.(persistentIntegration)
	if !ok {
		return
	}
	if err := persister.Persist(); err != nil {
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
