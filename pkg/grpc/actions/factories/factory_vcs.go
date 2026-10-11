package factories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func openFactoryVCS(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) vcs.Provider {
	if factory.OnboardingConfigValue().EffectiveVCSProvider() == models.ProviderBitbucket {
		return &bitbucketProvider{
			db:      db,
			deps:    deps,
			factory: factory,
		}
	}
	return vcs.Select(factory.OnboardingConfigValue().VCSProvider, &githubProvider{
		db:      db,
		deps:    deps,
		factory: factory,
	})
}

// githubProvider runs the current GitHub pull request calls.
// Velocity listing uses its own GitHub provider in the sync worker.
type githubProvider struct {
	db      *gorm.DB
	deps    IntakeDependencies
	factory *models.Factory
	client  factoryGitHubAPI
}

// ListMergedPullRequests is unused on this client. The velocity worker
// lists merged pull requests through its own GitHub provider.
func (g *githubProvider) ListMergedPullRequests(context.Context, string, time.Time, time.Time) ([]vcs.MergedPullRequest, error) {
	return nil, errors.New("merged pull request listing is not available on this client")
}

func (g *githubProvider) ReadMergeability(ctx context.Context, pullRequest *models.FactoryPullRequest) (vcs.Mergeability, error) {
	result, err := evaluateFactoryPullRequestMergeability(ctx, g.db, g.deps, g.factory, pullRequest)
	if err != nil {
		return vcs.Mergeability{}, err
	}
	g.client = result.Client
	return mergeabilityToVCS(result), nil
}

func (g *githubProvider) MergePullRequest(ctx context.Context, repository string, number int, method, expectedSHA string) error {
	if err := g.ensureClient(); err != nil {
		return err
	}
	_, _, err := g.client.MergePullRequest(ctx, repository, number, "", &github.PullRequestOptions{
		MergeMethod: method,
		SHA:         expectedSHA,
	})
	return err
}

func (g *githubProvider) ClosePullRequest(ctx context.Context, ref vcs.PullRequestRef) error {
	if ref.Provider == models.FactoryPullRequestProviderBitbucket {
		return errCannotCloseBitbucketPullRequest
	}
	if ref.Provider != models.FactoryPullRequestProviderGitHub {
		return errCannotClosePullRequest
	}
	if err := g.ensureClient(); err != nil {
		return err
	}
	_, _, err := g.client.EditPullRequest(ctx, ref.Repository, int(ref.Number), &github.PullRequest{
		State: github.Ptr("closed"),
	})
	if err != nil {
		return errors.Join(errCannotClosePullRequest, err)
	}
	return nil
}

func (g *githubProvider) ensureClient() error {
	if g.client != nil {
		return nil
	}
	client, err := newFactoryGitHubAPI(g.db, g.deps, g.factory)
	if err != nil {
		return err
	}
	g.client = client
	return nil
}

func connectFactoryVCS(provider vcs.Provider) error {
	github, ok := provider.(*githubProvider)
	if !ok {
		return nil
	}
	return github.ensureClient()
}

func readFactoryPullRequestMergeability(
	ctx context.Context,
	db *gorm.DB,
	provider vcs.Provider,
	pullRequest *models.FactoryPullRequest,
) (*factoryPullRequestMergeability, error) {
	read, err := provider.ReadMergeability(ctx, pullRequest)
	if err != nil {
		return nil, err
	}
	result := mergeabilityFromVCS(pullRequest, read)
	if github, ok := provider.(*githubProvider); ok {
		result.Client = github.client
	}
	if err := persistFactoryPullRequestMergeability(db, pullRequest, result); err != nil {
		return nil, err
	}
	return result, nil
}

func mergeabilityToVCS(result *factoryPullRequestMergeability) vcs.Mergeability {
	if result == nil {
		return vcs.Mergeability{}
	}
	return vcs.Mergeability{
		CanMerge:       result.CanMerge,
		BlockedReason:  mergeabilityBlockedReasonName(result.BlockedReason),
		Message:        result.Message,
		AllowedMethods: allowedMethodNames(result.AllowedMethods),
		HeadSHA:        result.HeadSHA,
		CanRetry:       result.canRetry,
	}
}

func mergeabilityFromVCS(pullRequest *models.FactoryPullRequest, read vcs.Mergeability) *factoryPullRequestMergeability {
	return &factoryPullRequestMergeability{
		CanMerge:       read.CanMerge,
		BlockedReason:  mergeabilityBlockedReasonFromName(read.BlockedReason),
		Message:        read.Message,
		AllowedMethods: mergeMethodsFromNames(read.AllowedMethods),
		HeadSHA:        read.HeadSHA,
		PullRequest:    pullRequest,
		canRetry:       read.CanRetry,
	}
}

func allowedMethodNames(methods []pb.FactoryPullRequestMergeability_MergeMethod) []string {
	joined := mergeMethodNames(methods)
	if joined == "" {
		return nil
	}
	return strings.Split(joined, ",")
}
