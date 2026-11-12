package pulls

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type ListPullRequestReviewCandidates struct{}

type reviewCandidatesConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	PullNumber any    `json:"pullNumber" mapstructure:"pullNumber"`
}

type ReviewerIdentity struct {
	ID    int64  `json:"id" mapstructure:"id"`
	Login string `json:"login" mapstructure:"login"`
}

func (c *ListPullRequestReviewCandidates) Name() string {
	return "github.listPullRequestReviewCandidates"
}
func (c *ListPullRequestReviewCandidates) Label() string {
	return "List Pull Request Review Candidates"
}
func (c *ListPullRequestReviewCandidates) Description() string {
	return "Find eligible human reviewers for a pull request"
}
func (c *ListPullRequestReviewCandidates) Documentation() string {
	return `List human collaborators with write, maintain, or admin access. Exclude the pull request author and bots. Read every page; incomplete discovery fails the execution. Output includes the pull request author, state, head, base, and candidate IDs and logins.`
}
func (c *ListPullRequestReviewCandidates) Icon() string  { return "github" }
func (c *ListPullRequestReviewCandidates) Color() string { return "gray" }
func (c *ListPullRequestReviewCandidates) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *ListPullRequestReviewCandidates) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "repository", Label: "Repository", Type: configuration.FieldTypeIntegrationResource, Required: true, TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "repository", UseNameAsValue: true}}},
		{Name: "pullNumber", Label: "Pull Request Number", Type: configuration.FieldTypeString, Required: true, Description: "Pull request number. Supports expressions."},
	}
}
func (c *ListPullRequestReviewCandidates) Setup(ctx core.SetupContext) error {
	var config reviewCandidatesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if strings.TrimSpace(config.Repository) == "" {
		return fmt.Errorf("repository is required")
	}
	if !common.IsExpression(pullNumberText(config.PullNumber)) {
		if _, err := parsePullNumber(config.PullNumber); err != nil {
			return err
		}
	}
	return common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration)
}
func (c *ListPullRequestReviewCandidates) Execute(ctx core.ExecutionContext) error {
	var config reviewCandidatesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	number, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return err
	}
	if strings.TrimSpace(config.Repository) == "" {
		return fmt.Errorf("repository is required")
	}
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return err
	}
	pr, _, err := client.GetPullRequest(context.Background(), config.Repository, number)
	if err != nil {
		return fmt.Errorf("read pull request: %w", err)
	}
	if pr.GetUser().GetLogin() == "" {
		return fmt.Errorf("pull request author is missing")
	}
	users, err := client.ListHumanReviewers(context.Background(), config.Repository)
	if err != nil {
		return err
	}
	candidates := []ReviewerIdentity{}
	for _, user := range users {
		if user.GetID() <= 0 {
			return fmt.Errorf("review candidate ID is missing")
		}
		if strings.EqualFold(user.GetLogin(), pr.GetUser().GetLogin()) {
			continue
		}
		candidates = append(candidates, ReviewerIdentity{ID: user.GetID(), Login: user.GetLogin()})
	}
	return ctx.ExecutionState.Emit("default", "github.pullRequestReviewCandidates", []any{map[string]any{
		"repository": config.Repository, "pullNumber": number, "state": pr.GetState(), "merged": pr.GetMerged(), "head": pr.Head, "base": pr.Base, "candidates": candidates,
		"author": ReviewerIdentity{ID: pr.GetUser().GetID(), Login: pr.GetUser().GetLogin()},
	}})
}
func (c *ListPullRequestReviewCandidates) Hooks() []core.Hook                      { return nil }
func (c *ListPullRequestReviewCandidates) HandleHook(core.ActionHookContext) error { return nil }
func (c *ListPullRequestReviewCandidates) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *ListPullRequestReviewCandidates) Cancel(core.ExecutionContext) error { return nil }
func (c *ListPullRequestReviewCandidates) Cleanup(core.SetupContext) error    { return nil }

func (c *ListPullRequestReviewCandidates) ExampleOutput() map[string]any {
	return map[string]any{"type": "github.pullRequestReviewCandidates", "timestamp": "2026-01-01T09:00:00Z", "data": map[string]any{
		"repository": "acme/app", "pullNumber": 42, "state": "open", "merged": false,
		"author": map[string]any{"id": 1, "login": "author"},
		"head":   map[string]any{"sha": "abc"}, "base": map[string]any{"ref": "main", "sha": "base-sha"},
		"candidates": []any{map[string]any{"id": 123, "login": "octocat"}},
	}}
}
