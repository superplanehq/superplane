package factories

import (
	"context"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	repositoryReviewBotsRecentPRLimit = 5
	superplaneAgentBotLogin           = "superplaneagent"
)

type githubReviewBotsAPI interface {
	ListPullRequests(ctx context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListReviews(ctx context.Context, repository string, pullNumber int) ([]*github.PullRequestReview, error)
	ListIssueComments(ctx context.Context, repository string, issueNumber int) ([]*github.IssueComment, error)
	ListPullRequestComments(ctx context.Context, repository string, pullNumber int) ([]*github.PullRequestComment, error)
}

type repositoryReviewBot struct {
	Login       string
	DisplayName string
}

func listRepositoryReviewBotsFromClient(ctx context.Context, client githubReviewBotsAPI, repository string) ([]repositoryReviewBot, error) {
	pulls, err := recentReviewBotPullRequests(ctx, client, repository)
	if err != nil {
		return nil, err
	}

	bots := map[string]repositoryReviewBot{}
	for _, pull := range pulls {
		if pull == nil {
			continue
		}
		number := pull.GetNumber()
		if number <= 0 {
			continue
		}
		collectReviewBotsFromPull(ctx, client, repository, number, bots)
	}
	return serializedReviewBots(bots), nil
}

func recentReviewBotPullRequests(ctx context.Context, client githubReviewBotsAPI, repository string) ([]*github.PullRequest, error) {
	pulls := make([]*github.PullRequest, 0, repositoryReviewBotsRecentPRLimit)
	seen := map[int]struct{}{}
	add := func(candidates []*github.PullRequest) {
		for _, pull := range candidates {
			if pull == nil || len(pulls) >= repositoryReviewBotsRecentPRLimit {
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
		if len(pulls) >= repositoryReviewBotsRecentPRLimit {
			break
		}
		page, _, err := client.ListPullRequests(ctx, repository, &github.PullRequestListOptions{
			State:     state,
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: repositoryReviewBotsRecentPRLimit,
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

func collectReviewBotsFromPull(
	ctx context.Context,
	client githubReviewBotsAPI,
	repository string,
	number int,
	bots map[string]repositoryReviewBot,
) {
	if reviews, err := client.ListReviews(ctx, repository, number); err == nil {
		for _, review := range reviews {
			if review == nil {
				continue
			}
			collectReviewBotUser(bots, review.GetUser())
		}
	}
	if comments, err := client.ListIssueComments(ctx, repository, number); err == nil {
		for _, comment := range comments {
			if comment == nil {
				continue
			}
			collectReviewBotUser(bots, comment.GetUser())
		}
	}
	if comments, err := client.ListPullRequestComments(ctx, repository, number); err == nil {
		for _, comment := range comments {
			if comment == nil {
				continue
			}
			collectReviewBotUser(bots, comment.GetUser())
		}
	}
}

func collectReviewBotUser(bots map[string]repositoryReviewBot, user *github.User) {
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
	bots[login] = repositoryReviewBot{Login: login, DisplayName: displayName}
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

func serializedReviewBots(bots map[string]repositoryReviewBot) []repositoryReviewBot {
	out := make([]repositoryReviewBot, 0, len(bots))
	for _, bot := range bots {
		out = append(out, bot)
	}
	return out
}

func loadReviewBotsFromGitHubClient(ctx context.Context, client *common.Client, repository string) ([]repositoryReviewBot, error) {
	return listRepositoryReviewBotsFromClient(ctx, client, repository)
}
