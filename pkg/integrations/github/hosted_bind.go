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
	listAppInstallations        = listAppInstallationsFromGitHub
	listAppInstallationRequests = listAppInstallationRequestsFromGitHub
)

func (g *GitHub) bindHostedInstallation(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	return g.bindHostedInstallationWith(ctx.Integration, ctx.Logger, metadata, installationID)
}

func (g *GitHub) bindHostedInstallationRepositories(
	ctx core.HTTPRequestContext,
	metadata common.Metadata,
	installation common.PendingInstallation,
	repositories []common.Repository,
) error {
	if len(repositories) == 0 {
		return fmt.Errorf("at least one repository is required")
	}

	metadata.InstallationID = installation.ID
	metadata.Owner = installation.AccountLogin
	metadata.Repositories = slices.Clone(repositories)
	metadata.SelectedRepositories = slices.Clone(repositories)
	metadata.RepositoryScoped = true
	remainingRequests := slices.DeleteFunc(metadata.CurrentInstallRequests(), func(request common.InstallRequest) bool {
		return request.AccountLogin == "" || strings.EqualFold(request.AccountLogin, metadata.Owner)
	})
	metadata.SetInstallRequests(remainingRequests)

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	ctx.Logger.Infof(
		"Successfully connected GitHub App %s - installation=%s repositories=%d",
		metadata.GitHubApp.Slug,
		metadata.InstallationID,
		len(metadata.Repositories),
	)
	return nil
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
	metadata.RepositoryScoped = false
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

// reconcileInstallRequests resolves a pending request only after repository
// discovery verifies that the member can use the requested installation. The
// approve callback carries no CSRF state and the installation webhook cannot
// find a connection without an installation ID, so Sync performs this check.
//
// The request callback from GitHub also does not name the requested account,
// so when it is unknown Sync finds the member's open install request on
// GitHub and records the account on the metadata for the next sync and the
// waiting screen.
func (g *GitHub) reconcileInstallRequests(ctx core.SyncContext, app common.HostedApp, metadata *common.Metadata) error {
	trackedRequests := metadata.CurrentInstallRequests()
	requester := strings.TrimSpace(metadata.StartedByGitHubLogin)
	localUnverifiedDiscovery := useDevelopmentGitHubDiscovery() && requester == "development"
	if requester != "" && !localUnverifiedDiscovery {
		trackedRequests = slices.DeleteFunc(trackedRequests, func(request common.InstallRequest) bool {
			return request.RequesterLogin != "" && !strings.EqualFold(request.RequesterLogin, requester)
		})
	}
	openRequests := trackedRequests
	if requester != "" {
		requestContext := ctx.Context
		if requestContext == nil {
			requestContext = context.Background()
		}
		client, err := newAppJWTClient(ctx.Integration, app.ID)
		if err != nil {
			return fmt.Errorf("failed to create app client: %w", err)
		}
		lookupRequester := requester
		if localUnverifiedDiscovery {
			// Local discovery has no linked GitHub identity. Query all App
			// requests, then retain only requests already tracked by this
			// connection. This keeps an open organization request from being
			// mistaken for an approved installation without attaching another
			// developer's request to this connection.
			lookupRequester = ""
		}
		openRequests, err = listAppInstallationRequests(requestContext, client, lookupRequester)
		if err != nil {
			return fmt.Errorf("failed to list app installation requests: %w", err)
		}
		if localUnverifiedDiscovery {
			openRequests = trackedOpenInstallRequests(trackedRequests, openRequests)
		}
	}

	// Put GitHub's current records first so their request IDs and timestamps
	// replace callback placeholders for the same account during deduplication.
	candidates := append(slices.Clone(openRequests), trackedRequests...)
	unresolved := make([]common.InstallRequest, 0, len(openRequests))
	now := time.Now().UTC()
	followUpDiscovery := false
	for _, request := range candidates {
		if installRequestIsOpen(request, openRequests) {
			unresolved = append(unresolved, request)
			continue
		}
		if installRequestHasVerifiedInstallation(request, metadata.PendingInstallations) {
			continue
		}
		if installRequestMayStillResolve(request, now) {
			unresolved = append(unresolved, request)
			continue
		}
		followUpDiscovery = true
	}
	metadata.SetInstallRequests(unresolved)
	if followUpDiscovery {
		metadata.InstallRequestDiscoveryUntil = now.Add(installRequestFollowUpDiscoveryPeriod).Format(time.RFC3339Nano)
	} else if len(unresolved) == 0 {
		metadata.InstallRequestDiscoveryUntil = ""
	}
	return nil
}

func pendingInstallationIDs(installations []common.PendingInstallation) []string {
	ids := make([]string, 0, len(installations))
	for _, installation := range installations {
		if installation.ID != "" {
			ids = append(ids, installation.ID)
		}
	}
	return ids
}

func installRequestHasVerifiedInstallation(
	request common.InstallRequest,
	installations []common.PendingInstallation,
) bool {
	if strings.TrimSpace(request.AccountLogin) != "" {
		installation, found := installationForAccount(installations, request.AccountLogin)
		return found && len(installation.Repositories) > 0
	}
	if request.ExistingInstallationIDs == nil {
		return false
	}

	return slices.ContainsFunc(installations, func(installation common.PendingInstallation) bool {
		return len(installation.Repositories) > 0 && !slices.Contains(request.ExistingInstallationIDs, installation.ID)
	})
}

func installRequestMayStillResolve(request common.InstallRequest, now time.Time) bool {
	createdAt, err := time.Parse(time.RFC3339Nano, request.CreatedAt)
	if err != nil {
		return false
	}
	return now.Before(createdAt.Add(installRequestResolutionGracePeriod))
}

func installationForAccount(installations []common.PendingInstallation, account string) (common.PendingInstallation, bool) {
	if strings.TrimSpace(account) == "" {
		return common.PendingInstallation{}, false
	}
	for _, installation := range installations {
		if strings.EqualFold(installation.AccountLogin, account) {
			return installation, true
		}
	}
	return common.PendingInstallation{}, false
}

func installRequestIsOpen(request common.InstallRequest, open []common.InstallRequest) bool {
	return slices.ContainsFunc(open, func(candidate common.InstallRequest) bool {
		if request.ID != "" && candidate.ID != "" {
			return request.ID == candidate.ID
		}
		return request.AccountLogin != "" && strings.EqualFold(request.AccountLogin, candidate.AccountLogin)
	})
}

func trackedOpenInstallRequests(tracked, open []common.InstallRequest) []common.InstallRequest {
	return slices.DeleteFunc(slices.Clone(open), func(candidate common.InstallRequest) bool {
		return !installRequestIsOpen(candidate, tracked)
	})
}

// listAppInstallationRequestsFromGitHub returns open App install requests. A
// non-empty requester limits the result to requests made by that GitHub user.
func listAppInstallationRequestsFromGitHub(ctx context.Context, client *github.Client, requesterLogin string) ([]common.InstallRequest, error) {
	requesterLogin = strings.TrimSpace(requesterLogin)
	result := []common.InstallRequest{}
	opts := &github.ListOptions{PerPage: 100}
	for {
		requests, response, err := client.Apps.ListInstallationRequests(ctx, opts)
		if err != nil {
			return nil, err
		}

		for _, request := range requests {
			if requesterLogin == "" || strings.EqualFold(request.GetRequester().GetLogin(), requesterLogin) {
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

func listAppInstallationsFromGitHub(ctx context.Context, client *github.Client) ([]common.PendingInstallation, error) {
	result := []common.PendingInstallation{}
	opts := &github.ListOptions{PerPage: 100}
	for {
		installations, response, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return nil, err
		}

		for _, installation := range installations {
			result = append(result, common.PendingInstallation{
				ID:           strconv.FormatInt(installation.GetID(), 10),
				AccountLogin: installation.GetAccount().GetLogin(),
				AccountType:  installation.GetAccount().GetType(),
			})
		}

		if response == nil || response.NextPage == 0 {
			return result, nil
		}
		opts.Page = response.NextPage
	}
}
