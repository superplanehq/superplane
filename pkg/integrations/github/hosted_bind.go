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
// approved it on GitHub. The approve callback carries no CSRF state and the
// installation webhook cannot find a connection without an installation id,
// so Sync asks GitHub whether the requested account has the App installed.
// An approved installation joins the account picker; the member still picks
// the account, a silent bind must not happen.
//
// The request callback from GitHub also does not name the requested account,
// so when it is unknown Sync finds the member's open install request on
// GitHub and records the account on the metadata for the next sync and the
// waiting screen.
func (g *GitHub) adoptRequestedInstallation(ctx core.SyncContext, app common.HostedApp, metadata *common.Metadata) error {
	client, err := newAppJWTClient(ctx.Integration, app.ID)
	if err != nil {
		return fmt.Errorf("failed to create app client: %w", err)
	}

	trackedRequests := metadata.CurrentInstallRequests()
	if requester := strings.TrimSpace(metadata.StartedByGitHubLogin); requester != "" {
		trackedRequests = slices.DeleteFunc(trackedRequests, func(request common.InstallRequest) bool {
			return request.RequesterLogin != "" && !strings.EqualFold(request.RequesterLogin, requester)
		})
	}
	openRequests := trackedRequests
	if strings.TrimSpace(metadata.StartedByGitHubLogin) != "" {
		openRequests, err = listAppInstallationRequests(context.Background(), client, metadata.StartedByGitHubLogin)
		if err != nil {
			return fmt.Errorf("failed to list app installation requests: %w", err)
		}
	}

	installations, err := listAppInstallations(context.Background(), client)
	if err != nil {
		return fmt.Errorf("failed to list app installations: %w", err)
	}

	// Put GitHub's current records first so their request IDs and timestamps
	// replace callback placeholders for the same account during deduplication.
	candidates := append(slices.Clone(openRequests), trackedRequests...)
	metadata.SetPendingInstallations(metadata.PendingInstallations)
	unresolved := make([]common.InstallRequest, 0, len(openRequests))
	for _, request := range candidates {
		installation, installed := installationForAccount(installations, request.AccountLogin)
		if installed {
			if !metadata.AllowsPendingInstallation(installation.ID) {
				metadata.PendingInstallations = append(metadata.PendingInstallations, installation)
			}
			continue
		}
		if installRequestIsOpen(request, openRequests) {
			unresolved = append(unresolved, request)
		}
	}
	metadata.SetInstallRequests(unresolved)
	return nil
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
