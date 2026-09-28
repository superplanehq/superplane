package github

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

var (
	newInstallationClient       = newClientForAppInstallation
	newAppJWTClient             = newClientForApp
	listInstallationRepos       = listInstallationRepositories
	listAppInstallationRequests = listAppInstallationRequestsFromGitHub
)

// hostedAdoptClockSkew tolerates small clock differences between GitHub and
// this server when adopt compares an installation creation time against the
// recorded request time.
const hostedAdoptClockSkew = 2 * time.Minute

// hostedInstallRequestGraceWindow keeps a recorded install request alive even
// when GitHub does not list it as open yet. Past the window, a request that
// GitHub closed without an installation was declined or cancelled, so the
// connect stops waiting for it.
const hostedInstallRequestGraceWindow = 10 * time.Minute

func (g *GitHub) bindHostedInstallation(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	return g.bindHostedInstallationWith(ctx.Integration, ctx.Logger, metadata, installationID)
}

func (g *GitHub) bindHostedInstallationWith(
	integration core.IntegrationContext,
	logger *logrus.Entry,
	metadata common.Metadata,
	installationID string,
) error {
	// A rebind moves the connection to another account, so the owner below
	// comes from the new installation.
	if metadata.InstallationID != "" && metadata.InstallationID != installationID {
		metadata.Owner = ""
	}
	metadata.InstallationID = installationID
	client, err := newInstallationClient(integration, metadata.GitHubApp.ID, installationID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if metadata.Owner == "" && !metadata.HostedApp {
		ghApp, _, err := client.Apps.Get(context.Background(), metadata.GitHubApp.Slug)
		if err != nil {
			return fmt.Errorf("failed to get app: %w", err)
		}

		metadata.Owner = ghApp.Owner.GetLogin()
	}

	repos, err := listInstallationRepos(context.Background(), client)
	if err != nil {
		return fmt.Errorf("failed to list repos: %w", err)
	}

	if metadata.Owner == "" {
		appClient, err := newAppJWTClient(integration, metadata.GitHubApp.ID)
		if err != nil {
			logger.Errorf("failed to create app client: %v", err)
		}
		metadata.Owner = resolveInstallationOwner(context.Background(), appClient, installationID, repos)
	}

	if metadata.Owner == "" {
		return fmt.Errorf("installation owner is empty for installation %s", installationID)
	}

	// State and PendingInstallations stay after bind. Onboarding reopens the
	// account picker with them, so the member can move the connection to
	// another GitHub account.
	metadata.Repositories = repos
	remainingRequests := slices.DeleteFunc(metadata.CurrentInstallRequests(), func(request common.InstallRequest) bool {
		return request.AccountLogin == "" || strings.EqualFold(request.AccountLogin, metadata.Owner)
	})
	metadata.SetInstallRequests(remainingRequests)

	integration.SetMetadata(metadata)
	integration.RemoveBrowserAction()
	integration.Ready()

	logger.Infof("Successfully installed GitHub App %s - installation=%s", metadata.GitHubApp.Slug, metadata.InstallationID)
	logger.Infof("Repositories: %v", metadata.Repositories)
	return nil
}

// adoptRequestedInstallation resolves a pending install request once an owner
// approved it on GitHub. The approve callback carries no CSRF state, so Sync
// looks up the requested account in the webhook-fed installations table. An
// approved installation joins the account picker; the member still picks the
// account, a silent bind must not happen.
//
// Identity goes first: when GitHub vouches that the starter's login can use
// the approved installation, another organization holding it does not block
// adoption. On an identity miss the installation must postdate the recorded
// request — a forged request callback that names an account with an old
// installation must not adopt it — and the cross-organization rule applies
// unchanged.
func (g *GitHub) adoptRequestedInstallation(ctx core.SyncContext, app common.HostedApp, metadata *common.Metadata) error {
	githubInstallations := g.listAppInstallationsForAdopt(ctx, app)
	login := g.adoptIdentityLogin(*metadata)

	metadata.SetPendingInstallations(metadata.PendingInstallations)
	unresolved, err := g.adoptMatchingInstallations(ctx, app, login, metadata.CurrentInstallRequests(), githubInstallations, metadata)
	if err != nil {
		return err
	}

	// The request callback from GitHub names no account, so a stored request
	// can carry only the requester. GitHub's open request list fills the
	// account in, and a second pass adopts an installation the enriched
	// request now matches.
	if confirmed, ok := g.confirmRequestsOnGitHub(ctx, app, login, unresolved); ok {
		unresolved, err = g.adoptMatchingInstallations(ctx, app, login, confirmed, githubInstallations, metadata)
		if err != nil {
			return err
		}
	}

	metadata.SetInstallRequests(unresolved)
	return nil
}

// adoptMatchingInstallations moves every approved installation into the
// account picker and returns the requests that stay unresolved.
func (g *GitHub) adoptMatchingInstallations(
	ctx core.SyncContext,
	app common.HostedApp,
	login string,
	requests []common.InstallRequest,
	githubInstallations []hostedInstallationSnapshot,
	metadata *common.Metadata,
) ([]common.InstallRequest, error) {
	unresolved := make([]common.InstallRequest, 0, len(requests))
	for _, request := range requests {
		installation, found := lookupRequestedInstallation(request.AccountLogin, githubInstallations)
		if found {
			adopted, err := g.adoptFoundInstallation(ctx, app, login, request, installation, metadata)
			if err != nil {
				return nil, err
			}
			if adopted {
				continue
			}
		}
		unresolved = append(unresolved, request)
	}
	return unresolved, nil
}

// confirmRequestsOnGitHub reconciles unresolved requests with the open
// install requests GitHub lists for the requester. The open list carries the
// account name the request callback omitted, and its absence past the grace
// window means the request was declined or cancelled. The second return is
// false when no request needed GitHub or the lookup failed; the caller then
// keeps the requests unchanged.
func (g *GitHub) confirmRequestsOnGitHub(
	ctx core.SyncContext,
	app common.HostedApp,
	login string,
	requests []common.InstallRequest,
) ([]common.InstallRequest, bool) {
	if !requestsNeedGitHubConfirmation(requests) || strings.TrimSpace(login) == "" {
		return requests, false
	}

	client, err := newAppJWTClient(ctx.Integration, app.ID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to create app client for install requests: %v", err)
		}
		return requests, false
	}
	open, err := listAppInstallationRequests(context.Background(), client, login)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to list app install requests: %v", err)
		}
		return requests, false
	}

	confirmed := make([]common.InstallRequest, 0, len(requests)+len(open))
	for _, request := range requests {
		if strings.TrimSpace(request.AccountLogin) == "" {
			// The open list is the complete set of this requester's open
			// requests, so it replaces the account-less placeholder. Keep
			// the placeholder only while GitHub can lag behind a very
			// fresh callback.
			if len(open) == 0 && withinInstallRequestGraceWindow(request) {
				confirmed = append(confirmed, request)
			}
			continue
		}
		if withinInstallRequestGraceWindow(request) || containsRequestForAccount(open, request.AccountLogin) {
			confirmed = append(confirmed, request)
		}
	}
	for _, request := range open {
		if !containsRequestForAccount(confirmed, request.AccountLogin) {
			confirmed = append(confirmed, request)
		}
	}
	return confirmed, true
}

// requestsNeedGitHubConfirmation reports whether any request misses its
// account name or waited past the grace window, so only those cases spend a
// GitHub lookup.
func requestsNeedGitHubConfirmation(requests []common.InstallRequest) bool {
	return slices.ContainsFunc(requests, func(request common.InstallRequest) bool {
		return strings.TrimSpace(request.AccountLogin) == "" || !withinInstallRequestGraceWindow(request)
	})
}

// withinInstallRequestGraceWindow reports whether the request is too young to
// judge against GitHub's open request list. Requests without a recorded time
// never leave the window, so legacy rows keep waiting until they resolve.
func withinInstallRequestGraceWindow(request common.InstallRequest) bool {
	createdAt, err := time.Parse(time.RFC3339Nano, request.CreatedAt)
	if err != nil {
		return true
	}
	return time.Since(createdAt) < hostedInstallRequestGraceWindow
}

func containsRequestForAccount(requests []common.InstallRequest, account string) bool {
	if strings.TrimSpace(account) == "" {
		return false
	}
	return slices.ContainsFunc(requests, func(request common.InstallRequest) bool {
		return strings.EqualFold(request.AccountLogin, account)
	})
}

func (g *GitHub) adoptFoundInstallation(
	ctx core.SyncContext,
	app common.HostedApp,
	login string,
	request common.InstallRequest,
	installation hostedInstallationSnapshot,
	metadata *common.Metadata,
) (bool, error) {
	if g.adoptVerifiedByIdentity(ctx, app, login, installation) {
		addAdoptedInstallation(metadata, installation)
		return true, nil
	}

	if !installationPostdatesRequest(request, installation) {
		return false, nil
	}

	used, err := installationUsedByOtherOrg(ctx.OrganizationID, installation.ID)
	if err != nil {
		return false, err
	}
	if used {
		return false, nil
	}

	addAdoptedInstallation(metadata, installation)
	return true, nil
}

func addAdoptedInstallation(metadata *common.Metadata, installation hostedInstallationSnapshot) {
	if metadata.AllowsPendingInstallation(installation.ID) {
		return
	}
	metadata.PendingInstallations = append(metadata.PendingInstallations, common.PendingInstallation{
		ID:           installation.ID,
		AccountLogin: installation.AccountLogin,
		AccountType:  installation.AccountType,
	})
}

// installationPostdatesRequest reports whether the installation appeared on
// or after the recorded request time. Requests recorded before timestamps
// existed keep the previous rule.
func installationPostdatesRequest(request common.InstallRequest, installation hostedInstallationSnapshot) bool {
	if request.CreatedAt == "" {
		return true
	}
	requestedAt, err := time.Parse(time.RFC3339Nano, request.CreatedAt)
	if err != nil {
		return true
	}

	installedAt := installation.CreatedAt
	if installedAt.IsZero() {
		installedAt = installation.LastEventAt
	}
	if installedAt.IsZero() {
		return false
	}
	return !installedAt.Before(requestedAt.Add(-hostedAdoptClockSkew))
}

func (g *GitHub) adoptIdentityLogin(metadata common.Metadata) string {
	login, err := findGitHubLoginForUser(metadata.StartedByUserID)
	if err == nil && login != "" {
		return login
	}
	return metadata.StartedByGitHubLogin
}

func (g *GitHub) adoptVerifiedByIdentity(ctx core.SyncContext, app common.HostedApp, login string, installation hostedInstallationSnapshot) bool {
	if strings.TrimSpace(login) == "" {
		return false
	}

	// Adoption is privileged, so the check is live: a cached answer from
	// discovery must not authorize an adopt after GitHub revoked the access.
	allowed, err := g.userCanAccessInstallationLive(ctx.Integration, app.ID, login, installation)
	return err == nil && allowed
}

func (g *GitHub) listAppInstallationsForAdopt(ctx core.SyncContext, app common.HostedApp) []hostedInstallationSnapshot {
	installations, err := listAppInstallationsDetailed(ctx.Integration, app.ID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to list app installations: %v", err)
		}
		return nil
	}
	return installations
}

func lookupRequestedInstallation(accountLogin string, githubInstallations []hostedInstallationSnapshot) (hostedInstallationSnapshot, bool) {
	if snapshot, err := findHostedInstallationByAccount(context.Background(), accountLogin); err == nil && snapshot != nil && !snapshot.Deleted {
		return *snapshot, true
	}
	return installationForAccount(githubInstallations, accountLogin)
}

func installationForAccount(installations []hostedInstallationSnapshot, account string) (hostedInstallationSnapshot, bool) {
	if strings.TrimSpace(account) == "" {
		return hostedInstallationSnapshot{}, false
	}
	for _, installation := range installations {
		if strings.EqualFold(installation.AccountLogin, account) {
			return installation, true
		}
	}
	return hostedInstallationSnapshot{}, false
}

// listAppInstallationRequestsFromGitHub returns every open App install request
// made by the requester.
func listAppInstallationRequestsFromGitHub(ctx context.Context, client *github.Client, requesterLogin string) ([]common.InstallRequest, error) {
	if strings.TrimSpace(requesterLogin) == "" {
		return nil, nil
	}

	result := []common.InstallRequest{}
	opts := &github.ListOptions{PerPage: 100}
	for {
		requests, response, err := client.Apps.ListInstallationRequests(ctx, opts)
		if err != nil {
			return nil, err
		}

		for _, request := range requests {
			if strings.EqualFold(request.GetRequester().GetLogin(), requesterLogin) {
				createdAt := ""
				if request.CreatedAt != nil {
					createdAt = request.CreatedAt.Time.UTC().Format(time.RFC3339Nano)
				}
				result = append(result, common.InstallRequest{
					ID:             strconv.FormatInt(request.GetID(), 10),
					AccountLogin:   request.GetAccount().GetLogin(),
					RequesterLogin: request.GetRequester().GetLogin(),
					CreatedAt:      createdAt,
				})
			}
		}

		if response == nil || response.NextPage == 0 {
			return result, nil
		}
		opts.Page = response.NextPage
	}
}
