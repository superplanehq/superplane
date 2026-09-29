package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

var (
	exchangeHostedUserOAuthCode = exchangeHostedUserOAuthCodeOnGitHub
	listUserInstallations       = listUserInstallationsFromGitHub
)

// afterHostedAppUserOAuth finishes the GitHub App user OAuth round trip.
// GitHub redirects here with a code. The code buys a user access token, and
// GET /user/installations lists exactly the installations of this app the
// user can use — GitHub decides access itself, so no organization members
// permission is needed. The handler records the installations, fills the
// account picker, and returns to the setup page. A user without any
// installation continues straight to the GitHub install page.
func (g *GitHub) afterHostedAppUserOAuth(ctx core.HTTPRequestContext) {
	metadata, ok := decodeHostedMetadata(ctx)
	if !ok {
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	app, appOK := common.HostedAppFromEnv()
	state := ctx.Request.URL.Query().Get("state")
	if !appOK || state == "" || state != metadata.State {
		http.Error(ctx.Response, "invalid state", http.StatusBadRequest)
		return
	}

	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		// The user denied the authorization; the connect stays pending.
		redirectToIntegrationSettings(ctx)
		return
	}

	token, err := exchangeHostedUserOAuthCode(app, code)
	if err != nil {
		// The connect must not dead-end on a token error; the setup page
		// offers Connect again.
		ctx.Logger.Errorf("failed to exchange GitHub App user OAuth code: %v", err)
		redirectToIntegrationSettings(ctx)
		return
	}

	login, installations, err := listUserInstallations(token)
	if err != nil {
		ctx.Logger.Errorf("failed to list GitHub App installations for the user: %v", err)
		redirectToIntegrationSettings(ctx)
		return
	}

	g.recordUserInstallations(ctx, login, installations)

	if login != "" {
		metadata.StartedByGitHubLogin = login
	}
	pending := metadata.PendingInstallations
	requests := metadata.CurrentInstallRequests()
	for _, installation := range installations {
		pending = append(pending, common.PendingInstallation{
			ID:           installation.ID,
			AccountLogin: installation.AccountLogin,
			AccountType:  installation.AccountType,
		})
		requests = filterResolvedInstallRequests(requests, installation.AccountLogin)
	}
	metadata.SetPendingInstallations(pending)
	metadata.SetInstallRequests(requests)
	ctx.Integration.SetMetadata(metadata)

	if len(metadata.PendingInstallations) == 0 && !metadata.HasInstallRequests() {
		// Nothing installed and nothing waiting: continue to the install
		// page, exactly where the old flow started.
		persistIntegrationBeforeRedirect(ctx)
		http.Redirect(
			ctx.Response,
			ctx.Request,
			common.HostedAppInstallURL(app.Slug, metadata.State),
			http.StatusSeeOther,
		)
		return
	}

	// The picker has options now, so a stored install action is stale.
	ctx.Integration.RemoveBrowserAction()
	redirectToIntegrationSettings(ctx)
}

// recordUserInstallations stores what the user token proved: the
// installations exist, and this login can use them. Later syncs and other
// connects read these rows without any GitHub lookup.
func (g *GitHub) recordUserInstallations(
	ctx core.HTTPRequestContext,
	login string,
	installations []hostedInstallationSnapshot,
) {
	now := time.Now().UTC()
	for _, installation := range installations {
		if err := saveReconciledInstallation(installation); err != nil {
			ctx.Logger.Errorf("failed to record GitHub App installation %s: %v", installation.ID, err)
			continue
		}
		if login == "" {
			continue
		}
		if err := saveCachedInstallationMember(installation.ID, login, true, false, now); err != nil {
			ctx.Logger.Errorf("failed to record GitHub App installation access for %s: %v", login, err)
		}
	}
}

func exchangeHostedUserOAuthCodeOnGitHub(app common.HostedApp, code string) (string, error) {
	values := url.Values{}
	values.Set("client_id", app.ClientID)
	values.Set("client_secret", app.ClientSecret)
	values.Set("code", code)

	request, err := http.NewRequest(
		http.MethodPost,
		"https://github.com/login/oauth/access_token",
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()

	var body struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("GitHub token exchange failed: %s %s", body.Error, body.ErrorDescription)
	}
	return body.AccessToken, nil
}

func listUserInstallationsFromGitHub(token string) (string, []hostedInstallationSnapshot, error) {
	client := gh.NewClient(nil).WithAuthToken(token)

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return "", nil, err
	}

	result := []hostedInstallationSnapshot{}
	opts := &gh.ListOptions{PerPage: 100}
	for {
		installations, response, err := client.Apps.ListUserInstallations(context.Background(), opts)
		if err != nil {
			return "", nil, err
		}

		for _, installation := range installations {
			if installation == nil || installation.GetAccount() == nil {
				continue
			}
			createdAt := time.Time{}
			if installation.CreatedAt != nil {
				createdAt = installation.CreatedAt.Time
			}
			result = append(result, hostedInstallationSnapshot{
				ID:           strconv.FormatInt(installation.GetID(), 10),
				AccountLogin: installation.GetAccount().GetLogin(),
				AccountType:  installation.GetAccount().GetType(),
				CreatedAt:    createdAt,
				LastEventAt:  createdAt,
			})
		}

		if response == nil || response.NextPage == 0 {
			return user.GetLogin(), result, nil
		}
		opts.Page = response.NextPage
	}
}

func resetHostedUserOAuthHooks() {
	exchangeHostedUserOAuthCode = exchangeHostedUserOAuthCodeOnGitHub
	listUserInstallations = listUserInstallationsFromGitHub
}
