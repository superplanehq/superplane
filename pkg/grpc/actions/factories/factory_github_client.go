package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errFactoryGitHubNotConnected = errors.New("github is not connected")

// errFactoryVCSSetupFailed marks a configured VCS connection that SuperPlane
// could not load, for example after a database or credential read failure.
// These errors also match the provider "not connected" error, so request
// handlers keep their current response. Background publishers retry them.
var errFactoryVCSSetupFailed = errors.New("could not load the VCS connection")

func factoryVCSSetupError(notConnected, err error) error {
	return fmt.Errorf("%w: %w: %w", notConnected, errFactoryVCSSetupFailed, err)
}

// findFactoryVCSIntegration returns notConnected when the workspace has no
// ready VCS integration. Other lookup failures are setup failures.
func findFactoryVCSIntegration(db *gorm.DB, factory *models.Factory, appName string, notConnected error) (*models.Integration, error) {
	id, err := uuid.Parse(strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID))
	if err != nil {
		return nil, notConnected
	}
	integration, err := models.FindIntegrationInTransaction(db, factory.OrganizationID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notConnected
	}
	if err != nil {
		return nil, factoryVCSSetupError(notConnected, err)
	}
	if integration.State != models.IntegrationStateReady || integration.AppName != appName {
		return nil, notConnected
	}
	return integration, nil
}

type factoryGitHubAPI interface {
	GetPullRequest(ctx context.Context, repository string, pullNumber int) (*github.PullRequest, *github.Response, error)
	GetCombinedStatus(ctx context.Context, repository string, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
	ListCheckRunsForRef(ctx context.Context, repository string, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
	EditPullRequest(ctx context.Context, repository string, pullNumber int, pullRequest *github.PullRequest) (*github.PullRequest, *github.Response, error)
	FindRepository(repository string) (*github.Repository, error)
	MergePullRequest(ctx context.Context, repository string, pullNumber int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error)
}

var newFactoryGitHubAPI = githubClientForFactory

func githubClientForFactory(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (factoryGitHubAPI, error) {
	return githubIntegrationClient(db, deps, factory)
}

// githubIntegrationClient opens a client on the workspace GitHub integration.
func githubIntegrationClient(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (*common.Client, error) {
	if deps.Registry == nil || deps.Encryptor == nil {
		return nil, errFactoryGitHubNotConnected
	}

	integration, err := findFactoryVCSIntegration(db, factory, models.ProviderGitHub, errFactoryGitHubNotConnected)
	if err != nil {
		return nil, err
	}

	client, err := common.NewClient(
		contexts.NewIntegrationContext(db, nil, integration, deps.Encryptor, deps.Registry, nil),
		deps.Registry.HTTPContextInTransaction(db),
	)
	if err != nil {
		return nil, factoryVCSSetupError(errFactoryGitHubNotConnected, err)
	}
	return client, nil
}
