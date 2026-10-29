package common

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
)

// ListInstallationBranchNames returns branch names visible to a GitHub App
// installation. Callers pass the repository ID so the token can read that
// repository before a binding grant is committed.
func ListInstallationBranchNames(
	httpCtx core.HTTPContext,
	appID int64,
	privateKey string,
	installationID, repositoryID int64,
	repository string,
) ([]string, error) {
	privateKey = strings.TrimSpace(privateKey)
	repository = strings.TrimSpace(repository)
	if httpCtx == nil || appID == 0 || privateKey == "" || installationID == 0 || repositoryID == 0 || repository == "" {
		return nil, fmt.Errorf("github branch list is unavailable")
	}

	client, err := newInstallationGitHubClient(httpCtx, appID, privateKey, installationID, repositoryID)
	if err != nil {
		return nil, err
	}
	return client.branchNames(context.Background(), repository)
}

// InstallationRepositoryDefaultBranch returns the default branch GitHub
// reports for one repository. An empty string means GitHub did not report one.
func InstallationRepositoryDefaultBranch(
	httpCtx core.HTTPContext,
	appID int64,
	privateKey string,
	installationID, repositoryID int64,
	repository string,
) (string, error) {
	privateKey = strings.TrimSpace(privateKey)
	repository = strings.TrimSpace(repository)
	if httpCtx == nil || appID == 0 || privateKey == "" || installationID == 0 || repositoryID == 0 || repository == "" {
		return "", fmt.Errorf("github repository default branch is unavailable")
	}

	client, err := newInstallationGitHubClient(httpCtx, appID, privateKey, installationID, repositoryID)
	if err != nil {
		return "", err
	}
	repo, err := client.FindRepository(repository)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(repo.GetDefaultBranch()), nil
}

func newInstallationGitHubClient(
	httpCtx core.HTTPContext,
	appID int64,
	privateKey string,
	installationID, repositoryID int64,
) (*Client, error) {
	installationTransport, err := ghinstallation.New(&transport{http: httpCtx}, appID, installationID, []byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	if err := RestrictHostedAppTransport(installationTransport, []int64{repositoryID}); err != nil {
		return nil, err
	}
	return &Client{
		authMethod: AuthMethodApp,
		underlying: github.NewClient(&http.Client{Transport: installationTransport}),
	}, nil
}

func (c *Client) branchNames(ctx context.Context, repository string) ([]string, error) {
	names := []string{}
	opts := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		branches, resp, err := c.ListBranches(ctx, repository, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list branches: %w", err)
		}
		for _, branch := range branches {
			name := strings.TrimSpace(branch.GetName())
			if name == "" {
				continue
			}
			names = append(names, name)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return names, nil
}
