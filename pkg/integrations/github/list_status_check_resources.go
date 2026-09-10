package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const statusCheckRecentPRLimit = 3

type githubStatusCheckAPI interface {
	FindRepository(repository string) (*github.Repository, error)
	GetBranchProtection(ctx context.Context, repository, branch string) (*github.Protection, error)
	ListPullRequests(ctx context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListCheckRunsForRef(ctx context.Context, repository, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
	GetCombinedStatus(ctx context.Context, repository, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
}

type observedStatusCheck struct {
	Name string
	URL  string
}

func (g *GitHub) listStatusCheckResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := strings.TrimSpace(ctx.Parameters["repository"])
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return listStatusCheckResourcesFromClient(context.Background(), client, repository)
}

func listStatusCheckResourcesFromClient(ctx context.Context, client githubStatusCheckAPI, repository string) ([]core.IntegrationResource, error) {
	repo, err := client.FindRepository(repository)
	if err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(repo.GetDefaultBranch())
	if branch == "" {
		branch = "main"
	}

	protectionNames, err := statusCheckNamesForBranchProtection(ctx, client, repository, branch)
	if err != nil {
		protectionNames = nil
	}

	refs, err := recentStatusCheckRefs(ctx, client, repository, branch)
	if err != nil {
		refs = []string{branch}
	}

	observed, err := observedStatusChecksForRefs(ctx, client, repository, refs)
	if err != nil {
		observed = nil
	}

	return mergeStatusCheckResources(protectionNames, observed), nil
}

func statusCheckNamesForBranchProtection(ctx context.Context, client githubStatusCheckAPI, repository, branch string) ([]string, error) {
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

func recentStatusCheckRefs(ctx context.Context, client githubStatusCheckAPI, repository, defaultBranch string) ([]string, error) {
	refs := make([]string, 0, statusCheckRecentPRLimit)
	seen := map[string]struct{}{}
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, exists := seen[ref]; exists {
			return
		}
		if len(refs) >= statusCheckRecentPRLimit {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}

	addPullSHAs := func(state string) error {
		if len(refs) >= statusCheckRecentPRLimit {
			return nil
		}
		pulls, _, err := client.ListPullRequests(ctx, repository, &github.PullRequestListOptions{
			State:     state,
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: statusCheckRecentPRLimit,
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

	if err := addPullSHAs("open"); err != nil {
		return nil, err
	}
	closedErr := addPullSHAs("closed")
	if len(refs) == 0 {
		if closedErr != nil {
			return nil, closedErr
		}
		add(defaultBranch)
	}
	return refs, nil
}

func observedStatusChecksForRefs(ctx context.Context, client githubStatusCheckAPI, repository string, refs []string) ([]observedStatusCheck, error) {
	observed := make([]observedStatusCheck, 0)
	for _, ref := range refs {
		checkRuns, _, err := client.ListCheckRunsForRef(ctx, repository, ref, &github.ListCheckRunsOptions{
			Filter:      github.Ptr("latest"),
			ListOptions: github.ListOptions{PerPage: 100},
		})
		if err == nil {
			observed = append(observed, observedStatusChecksFromCheckRuns(checkRuns)...)
		}

		combined, _, err := client.GetCombinedStatus(ctx, repository, ref, &github.ListOptions{PerPage: 100})
		if err == nil {
			observed = append(observed, observedStatusChecksFromCombinedStatus(combined)...)
		}
	}
	return observed, nil
}

func observedStatusChecksFromCheckRuns(result *github.ListCheckRunsResults) []observedStatusCheck {
	if result == nil {
		return nil
	}
	observed := make([]observedStatusCheck, 0, len(result.CheckRuns))
	for _, run := range result.CheckRuns {
		if run == nil {
			continue
		}
		observed = append(observed, observedStatusCheck{
			Name: run.GetName(),
			URL:  firstNonEmptyString(run.GetDetailsURL(), run.GetHTMLURL()),
		})
	}
	return observed
}

func observedStatusChecksFromCombinedStatus(combined *github.CombinedStatus) []observedStatusCheck {
	if combined == nil {
		return nil
	}
	observed := make([]observedStatusCheck, 0, len(combined.Statuses))
	for _, status := range combined.Statuses {
		if status == nil {
			continue
		}
		observed = append(observed, observedStatusCheck{
			Name: status.GetContext(),
			URL:  status.GetTargetURL(),
		})
	}
	return observed
}

func mergeStatusCheckResources(protectionNames []string, observed []observedStatusCheck) []core.IntegrationResource {
	index := map[string]int{}
	resources := make([]core.IntegrationResource, 0, len(protectionNames)+len(observed))

	appendCheck := func(name, checkURL string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if existing, ok := index[key]; ok {
			if resources[existing].URL == "" {
				resources[existing].URL = strings.TrimSpace(checkURL)
			}
			return
		}
		index[key] = len(resources)
		resources = append(resources, core.IntegrationResource{
			Type: "status_check",
			Name: name,
			ID:   name,
			URL:  strings.TrimSpace(checkURL),
		})
	}

	for _, name := range protectionNames {
		appendCheck(name, "")
	}
	for _, item := range observed {
		appendCheck(item.Name, item.URL)
	}
	return resources
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
