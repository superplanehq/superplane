package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
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
	logger := reviewBotLogger(ctx.Logger).WithField("repository", repository)
	if repository == "" {
		logger.Info("review bot catalog skipped: repository parameter is empty")
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return listReviewBotResourcesFromClient(context.Background(), client, repository, logger)
}

func listReviewBotResourcesFromClient(ctx context.Context, client githubReviewBotAPI, repository string, logger *log.Entry) ([]core.IntegrationResource, error) {
	logger = reviewBotLogger(logger).WithField("repository", repository)
	pulls, err := recentReviewBotPullRequests(ctx, client, repository)
	if err != nil {
		return nil, err
	}

	pullNumbers := make([]int, 0, len(pulls))
	bots := map[string]core.IntegrationResource{}
	for _, pull := range pulls {
		if pull == nil {
			continue
		}
		number := pull.GetNumber()
		if number <= 0 {
			continue
		}
		pullNumbers = append(pullNumbers, number)
		collectReviewBotResourcesFromPull(ctx, client, repository, number, bots, logger)
	}

	out := make([]core.IntegrationResource, 0, len(bots))
	botIDs := make([]string, 0, len(bots))
	for _, bot := range bots {
		out = append(out, bot)
		botIDs = append(botIDs, bot.ID)
	}
	logger.WithFields(log.Fields{
		"pull_count":   len(pullNumbers),
		"pull_numbers": pullNumbers,
		"bot_count":    len(botIDs),
		"bot_ids":      botIDs,
	}).Info("listed review bot resources")
	return out, nil
}

func recentReviewBotPullRequests(ctx context.Context, client githubReviewBotAPI, repository string) ([]*github.PullRequest, error) {
	page, _, err := client.ListPullRequests(ctx, repository, &github.PullRequestListOptions{
		State:     "all",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: reviewBotRecentPRLimit,
		},
	})
	if err != nil {
		return nil, err
	}

	pulls := make([]*github.PullRequest, 0, reviewBotRecentPRLimit)
	seen := map[int]struct{}{}
	for _, pull := range page {
		if pull == nil || len(pulls) >= reviewBotRecentPRLimit {
			break
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
	return pulls, nil
}

func collectReviewBotResourcesFromPull(
	ctx context.Context,
	client githubReviewBotAPI,
	repository string,
	number int,
	bots map[string]core.IntegrationResource,
	logger *log.Entry,
) {
	logger = reviewBotLogger(logger).WithFields(log.Fields{
		"repository":  repository,
		"pull_number": number,
	})
	if reviews, err := client.ListReviews(ctx, repository, number); err != nil {
		logger.WithError(err).Warn("failed to list pull request reviews for review bot catalog")
	} else {
		for _, review := range reviews {
			if review == nil {
				continue
			}
			collectReviewBotResource(bots, review.GetUser())
		}
	}
	if comments, err := client.ListIssueComments(ctx, repository, number); err != nil {
		logger.WithError(err).Warn("failed to list issue comments for review bot catalog")
	} else {
		for _, comment := range comments {
			if comment == nil {
				continue
			}
			collectReviewBotResource(bots, comment.GetUser())
		}
	}
	if comments, err := client.ListPullRequestComments(ctx, repository, number); err != nil {
		logger.WithError(err).Warn("failed to list pull request comments for review bot catalog")
	} else {
		for _, comment := range comments {
			if comment == nil {
				continue
			}
			collectReviewBotResource(bots, comment.GetUser())
		}
	}
}

func reviewBotLogger(logger *log.Entry) *log.Entry {
	if logger != nil {
		return logger
	}
	return log.NewEntry(log.StandardLogger())
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
