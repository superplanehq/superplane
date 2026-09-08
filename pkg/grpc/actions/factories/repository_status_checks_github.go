package factories

import (
	"context"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const repositoryStatusCheckRecentPRLimit = 3

type githubRepositoryStatusAPI interface {
	FindRepository(repository string) (*github.Repository, error)
	GetBranchProtection(ctx context.Context, repository, branch string) (*github.Protection, error)
	ListPullRequests(ctx context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListCheckRunsForRef(ctx context.Context, repository, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
	GetCombinedStatus(ctx context.Context, repository, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
}

type repositoryStatusCheckSnapshot struct {
	Required []string
	Observed []observedRepositoryStatusCheck
}

func loadGitHubRepositoryStatusCheckSnapshot(ctx context.Context, client githubRepositoryStatusAPI, repository string) (repositoryStatusCheckSnapshot, error) {
	repo, err := client.FindRepository(repository)
	if err != nil {
		return repositoryStatusCheckSnapshot{}, err
	}

	branch := strings.TrimSpace(repo.GetDefaultBranch())
	if branch == "" {
		branch = "main"
	}

	required, err := requiredStatusChecksForBranch(ctx, client, repository, branch)
	if err != nil {
		required = nil
	}

	refs, err := recentRepositoryStatusCheckRefs(ctx, client, repository, branch)
	if err != nil {
		refs = []string{branch}
	}

	observed, err := observedStatusChecksForRefs(ctx, client, repository, refs)
	if err != nil {
		observed = nil
	}

	return repositoryStatusCheckSnapshot{Required: required, Observed: observed}, nil
}

func requiredStatusChecksForBranch(ctx context.Context, client githubRepositoryStatusAPI, repository, branch string) ([]string, error) {
	protection, err := client.GetBranchProtection(ctx, repository, branch)
	if err != nil {
		return nil, err
	}
	if protection == nil || protection.RequiredStatusChecks == nil {
		return nil, nil
	}

	required := protection.RequiredStatusChecks
	if required.Checks != nil {
		names := make([]string, 0, len(*required.Checks))
		for _, check := range *required.Checks {
			if check == nil {
				continue
			}
			names = append(names, check.Context)
		}
		if len(names) > 0 {
			return names, nil
		}
	}
	if required.Contexts != nil {
		return append([]string{}, *required.Contexts...), nil
	}
	return nil, nil
}

func recentRepositoryStatusCheckRefs(ctx context.Context, client githubRepositoryStatusAPI, repository, defaultBranch string) ([]string, error) {
	refs := make([]string, 0, repositoryStatusCheckRecentPRLimit)
	seen := map[string]struct{}{}
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, exists := seen[ref]; exists {
			return
		}
		if len(refs) >= repositoryStatusCheckRecentPRLimit {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}

	addPullSHAs := func(state string) error {
		if len(refs) >= repositoryStatusCheckRecentPRLimit {
			return nil
		}
		pulls, _, err := client.ListPullRequests(ctx, repository, &github.PullRequestListOptions{
			State:     state,
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: repositoryStatusCheckRecentPRLimit,
			},
		})
		if err != nil {
			return err
		}
		for _, pull := range pulls {
			if pull == nil || pull.Head == nil {
				continue
			}
			add(pull.Head.GetSHA())
		}
		return nil
	}

	closedErr := addPullSHAs("closed")
	openErr := addPullSHAs("open")
	if len(refs) == 0 {
		if closedErr != nil {
			return nil, closedErr
		}
		if openErr != nil {
			return nil, openErr
		}
		add(defaultBranch)
	}
	return refs, nil
}

func observedStatusChecksForRefs(ctx context.Context, client githubRepositoryStatusAPI, repository string, refs []string) ([]observedRepositoryStatusCheck, error) {
	observed := make([]observedRepositoryStatusCheck, 0)
	for _, ref := range refs {
		checkRuns, _, err := client.ListCheckRunsForRef(ctx, repository, ref, &github.ListCheckRunsOptions{
			Filter:      github.Ptr("latest"),
			ListOptions: github.ListOptions{PerPage: 100},
		})
		if err == nil {
			observed = append(observed, observedChecksFromCheckRuns(checkRuns)...)
		}

		combined, _, err := client.GetCombinedStatus(ctx, repository, ref, &github.ListOptions{PerPage: 100})
		if err == nil {
			observed = append(observed, observedChecksFromCombinedStatus(combined)...)
		}
	}
	return observed, nil
}

func observedChecksFromCheckRuns(result *github.ListCheckRunsResults) []observedRepositoryStatusCheck {
	if result == nil {
		return nil
	}
	observed := make([]observedRepositoryStatusCheck, 0, len(result.CheckRuns))
	for _, run := range result.CheckRuns {
		if run == nil {
			continue
		}
		observed = append(observed, observedRepositoryStatusCheck{
			Name:       run.GetName(),
			DetailsURL: firstNonEmpty(run.GetDetailsURL, run.GetHTMLURL),
		})
	}
	return observed
}

func observedChecksFromCombinedStatus(combined *github.CombinedStatus) []observedRepositoryStatusCheck {
	if combined == nil {
		return nil
	}
	observed := make([]observedRepositoryStatusCheck, 0, len(combined.Statuses))
	for _, status := range combined.Statuses {
		if status == nil {
			continue
		}
		observed = append(observed, observedRepositoryStatusCheck{
			Name:       status.GetContext(),
			DetailsURL: status.GetTargetURL(),
		})
	}
	return observed
}

func firstNonEmpty(getters ...func() string) string {
	for _, getter := range getters {
		if value := strings.TrimSpace(getter()); value != "" {
			return value
		}
	}
	return ""
}

func loadSnapshotFromGitHubClient(ctx context.Context, client *common.Client, repository string) (repositoryStatusCheckSnapshot, error) {
	return loadGitHubRepositoryStatusCheckSnapshot(ctx, client, repository)
}
