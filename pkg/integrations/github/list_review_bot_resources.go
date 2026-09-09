package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	reviewBotRecentPRLimit  = 5
	superplaneAgentBotLogin = "superplaneagent"
)

type githubReviewBotAPI interface {
	ListPullRequests(ctx context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListReviews(ctx context.Context, repository string, pullNumber int) ([]*github.PullRequestReview, error)
	ListIssueComments(ctx context.Context, repository string, issueNumber int) ([]*github.IssueComment, error)
	ListPullRequestComments(ctx context.Context, repository string, pullNumber int) ([]*github.PullRequestComment, error)
}

func (g *GitHub) listReviewBotResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := strings.TrimSpace(ctx.Parameters["repository"])
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return listReviewBotResourcesFromClient(context.Background(), client, repository)
}

func listReviewBotResourcesFromClient(ctx context.Context, client githubReviewBotAPI, repository string) ([]core.IntegrationResource, error) {
	pulls, err := recentReviewBotPullRequests(ctx, client, repository)
	if err != nil {
		return nil, err
	}

	bots := map[string]core.IntegrationResource{}
	for _, pull := range pulls {
		if pull == nil {
			continue
		}
		number := pull.GetNumber()
		if number <= 0 {
			continue
		}
		collectReviewBotResourcesFromPull(ctx, client, repository, number, bots)
	}

	out := make([]core.IntegrationResource, 0, len(bots))
	for _, bot := range bots {
		out = append(out, bot)
	}
	return out, nil
}

func recentReviewBotPullRequests(ctx context.Context, client githubReviewBotAPI, repository string) ([]*github.PullRequest, error) {
	pulls := make([]*github.PullRequest, 0, reviewBotRecentPRLimit)
	seen := map[int]struct{}{}
	add := func(candidates []*github.PullRequest) {
		for _, pull := range candidates {
			if pull == nil || len(pulls) >= reviewBotRecentPRLimit {
				return
			}
			number := pull.GetNumber()
			if number <= 0 {
				continue
			}
			if _, exists := seen[number]; exists {
				continue
			}
			seen[number] = struct{}{}
			pulls = append(pulls, pull)
		}
	}

	var firstErr error
	for _, state := range []string{"closed", "open"} {
		if len(pulls) >= reviewBotRecentPRLimit {
			break
		}
		page, _, err := client.ListPullRequests(ctx, repository, &github.PullRequestListOptions{
			State:     state,
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: reviewBotRecentPRLimit,
			},
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		add(page)
	}
	if len(pulls) == 0 {
		return nil, firstErr
	}
	return pulls, nil
}

func collectReviewBotResourcesFromPull(
	ctx context.Context,
	client githubReviewBotAPI,
	repository string,
	number int,
	bots map[string]core.IntegrationResource,
) {
	if reviews, err := client.ListReviews(ctx, repository, number); err == nil {
		for _, review := range reviews {
			if review == nil {
				continue
			}
			collectReviewBotResource(bots, review.GetUser())
		}
	}
	if comments, err := client.ListIssueComments(ctx, repository, number); err == nil {
		for _, comment := range comments {
			if comment == nil {
				continue
			}
			collectReviewBotResource(bots, comment.GetUser())
		}
	}
	if comments, err := client.ListPullRequestComments(ctx, repository, number); err == nil {
		for _, comment := range comments {
			if comment == nil {
				continue
			}
			collectReviewBotResource(bots, comment.GetUser())
		}
	}
}

func collectReviewBotResource(bots map[string]core.IntegrationResource, user *github.User) {
	if !isReviewBotUser(user) {
		return
	}
	displayName := strings.TrimSpace(user.GetLogin())
	login := normalizeReviewBotLogin(displayName)
	if login == "" || login == superplaneAgentBotLogin {
		return
	}
	if _, exists := bots[login]; exists {
		return
	}
	bots[login] = core.IntegrationResource{
		Type: "review_bot",
		Name: displayName,
		ID:   login,
	}
}

func isReviewBotUser(user *github.User) bool {
	if user == nil {
		return false
	}
	login := strings.TrimSpace(user.GetLogin())
	if login == "" {
		return false
	}
	if strings.EqualFold(user.GetType(), "Bot") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(login), "[bot]")
}

func normalizeReviewBotLogin(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "@")
	name = strings.TrimSuffix(name, "[bot]")
	return name
}
