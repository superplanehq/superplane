package github

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func (g *GitHub) listReviewerResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := strings.TrimSpace(ctx.Parameters["repository"])
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	repo, err := client.FindRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	reviewers := map[string]core.IntegrationResource{}
	// The owner of a personal repository can review even when GitHub does
	// not include them in the collaborator response.
	collectReviewerResource(reviewers, repo.GetOwner())

	opts := &github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		collaborators, response, err := client.ListCollaborators(context.Background(), repository, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repository reviewers: %w", err)
		}
		for _, collaborator := range collaborators {
			if collaborator == nil {
				continue
			}
			permissions := collaborator.Permissions
			if !permissions.GetPush() && !permissions.GetMaintain() && !permissions.GetAdmin() {
				continue
			}
			collectReviewerResource(reviewers, collaborator)
		}
		if response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}

	resources := make([]core.IntegrationResource, 0, len(reviewers))
	for _, reviewer := range reviewers {
		resources = append(resources, reviewer)
	}
	slices.SortFunc(resources, func(a, b core.IntegrationResource) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return resources, nil
}

func collectReviewerResource(reviewers map[string]core.IntegrationResource, user *github.User) {
	if user == nil || !strings.EqualFold(user.GetType(), "User") || isReviewBotUser(user) {
		return
	}
	login := strings.TrimSpace(user.GetLogin())
	if login == "" {
		return
	}
	key := strings.ToLower(login)
	if _, exists := reviewers[key]; exists {
		return
	}
	reviewers[key] = core.IntegrationResource{Type: "reviewer", ID: login, Name: login}
}
