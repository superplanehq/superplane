package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errFactoryGitHubNotConnected = errors.New("github is not connected")

type factoryGitHubAPI interface {
	GetPullRequest(ctx context.Context, repository string, pullNumber int) (*github.PullRequest, *github.Response, error)
	GetCombinedStatus(ctx context.Context, repository string, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
	ListCheckRunsForRef(ctx context.Context, repository string, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
	FindRepository(repository string) (*github.Repository, error)
	MergePullRequest(ctx context.Context, repository string, pullNumber int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error)
}

var newFactoryGitHubAPI = githubClientForFactory

func githubClientForFactory(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (factoryGitHubAPI, error) {
	if deps.Registry == nil || deps.Encryptor == nil {
		return nil, errFactoryGitHubNotConnected
	}

	integrationID := strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID)
	if integrationID == "" {
		return nil, errFactoryGitHubNotConnected
	}

	integration, err := findReadyOnboardingIntegration(db, factory.OrganizationID, integrationID)
	if err != nil {
		return nil, errFactoryGitHubNotConnected
	}
	if integration.AppName != "github" {
		return nil, errFactoryGitHubNotConnected
	}

	client, err := common.NewClient(
		contexts.NewIntegrationContext(db, nil, integration, deps.Encryptor, deps.Registry, nil),
		deps.Registry.HTTPContextInTransaction(db),
	)
	if err != nil {
		return nil, errFactoryGitHubNotConnected
	}
	return client, nil
}
