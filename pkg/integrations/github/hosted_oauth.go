package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	githubOAuthTokenURL        = "https://github.com/login/oauth/access_token"
	githubUserInstallationsURL = "https://api.github.com/user/installations"
)

type githubOAuthTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type githubUserInstallationsResponse struct {
	Installations []githubUserInstallation `json:"installations"`
}

type githubUserInstallation struct {
	ID      int64 `json:"id"`
	AppID   int64 `json:"app_id"`
	Account *struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
}

func (g *GitHub) afterHostedAppOAuth(ctx core.HTTPRequestContext) {
	metadata, ok := decodeHostedMetadata(ctx)
	if !ok {
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if metadata.InstallationID != "" {
		redirectToIntegrationSettings(ctx)
		return
	}

	state := ctx.Request.URL.Query().Get("state")
	if state == "" || state != metadata.State {
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	if errParam := ctx.Request.URL.Query().Get("error"); errParam != "" {
		ctx.Logger.Errorf("GitHub OAuth error: %s", errParam)
		http.Error(ctx.Response, "authorization was denied", http.StatusBadRequest)
		return
	}

	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		http.Error(ctx.Response, "missing code", http.StatusBadRequest)
		return
	}

	app, ok := common.HostedAppFromEnv()
	if !ok || !app.UserOAuthEnabled() {
		http.Error(ctx.Response, "hosted GitHub App OAuth is not configured", http.StatusNotFound)
		return
	}

	token, err := exchangeGitHubUserOAuthToken(
		ctx.HTTP,
		app.ClientID,
		app.ClientSecret,
		code,
		common.HostedAppOAuthCallbackURL(ctx.BaseURL),
	)
	if err != nil {
		ctx.Logger.Errorf("failed to exchange GitHub OAuth code: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	installations, err := listUserAppInstallations(ctx.HTTP, token, app.ID)
	if err != nil {
		ctx.Logger.Errorf("failed to list user GitHub App installations: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	switch {
	case len(installations) == 0:
		g.redirectToHostedInstall(ctx, metadata, app.Slug)
	case len(installations) == 1 && !metadata.InstallRequested:
		if err := g.bindHostedInstallation(ctx, metadata, installations[0].ID); err != nil {
			ctx.Logger.Errorf("%v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
		}
		redirectToIntegrationSettings(ctx)
	default:
		metadata.PendingInstallations = installations
		ctx.Integration.SetMetadata(metadata)
		ctx.Integration.RemoveBrowserAction()
		redirectToIntegrationSettings(ctx)
	}
}

func (g *GitHub) afterHostedAppBind(ctx core.HTTPRequestContext) {
	metadata, ok := decodeHostedMetadata(ctx)
	if !ok {
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if metadata.InstallationID != "" {
		redirectToIntegrationSettings(ctx)
		return
	}

	state := ctx.Request.URL.Query().Get("state")
	installationID := ctx.Request.URL.Query().Get("installation_id")
	if state == "" || state != metadata.State || installationID == "" {
		http.Error(ctx.Response, "invalid installation ID or state", http.StatusBadRequest)
		return
	}

	if !metadata.AllowsPendingInstallation(installationID) {
		http.Error(ctx.Response, "installation is not allowed", http.StatusBadRequest)
		return
	}

	if err := g.bindHostedInstallation(ctx, metadata, installationID); err != nil {
		ctx.Logger.Errorf("%v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	redirectToIntegrationSettings(ctx)
}

func (g *GitHub) redirectToHostedInstall(ctx core.HTTPRequestContext, metadata common.Metadata, slug string) {
	metadata.PendingInstallations = nil
	installURL := common.HostedAppInstallURL(slug, metadata.State)
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: hostedInstallDescription,
		URL:         installURL,
		Method:      "GET",
	})
	ctx.Integration.SetMetadata(metadata)
	http.Redirect(ctx.Response, ctx.Request, installURL, http.StatusSeeOther)
}

func decodeHostedMetadata(ctx core.HTTPRequestContext) (common.Metadata, bool) {
	metadata := common.Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("failed to decode metadata: %v", err)
		return metadata, false
	}
	return metadata, true
}

func exchangeGitHubUserOAuthToken(httpCtx core.HTTPContext, clientID, clientSecret, code, redirectURI string) (string, error) {
	if httpCtx == nil {
		return "", fmt.Errorf("HTTP context is required")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest(http.MethodPost, githubOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := httpCtx.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub OAuth token exchange failed: status %d", response.StatusCode)
	}

	var token githubOAuthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", err
	}
	if token.Error != "" || token.AccessToken == "" {
		return "", fmt.Errorf("GitHub OAuth token exchange failed")
	}

	return token.AccessToken, nil
}

func listUserAppInstallations(httpCtx core.HTTPContext, token string, appID int64) ([]common.PendingInstallation, error) {
	if httpCtx == nil {
		return nil, fmt.Errorf("HTTP context is required")
	}

	var all []common.PendingInstallation
	page := 1
	for {
		endpoint, err := url.Parse(githubUserInstallationsURL)
		if err != nil {
			return nil, err
		}
		query := endpoint.Query()
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()

		req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+token)

		response, err := httpCtx.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			return nil, err
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("GitHub user installations request failed: status %d", response.StatusCode)
		}

		var payload githubUserInstallationsResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		for _, installation := range payload.Installations {
			if appID != 0 && installation.AppID != 0 && installation.AppID != appID {
				continue
			}
			pending := common.PendingInstallation{ID: strconv.FormatInt(installation.ID, 10)}
			if installation.Account != nil {
				pending.AccountLogin = installation.Account.Login
				pending.AccountType = installation.Account.Type
			}
			all = append(all, pending)
		}

		if len(payload.Installations) < 100 {
			break
		}
		page++
	}

	return all, nil
}
