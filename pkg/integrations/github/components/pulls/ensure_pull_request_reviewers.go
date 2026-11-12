package pulls

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type EnsurePullRequestReviewers struct{}

type ensureReviewersConfiguration struct {
	Repository      string             `json:"repository" mapstructure:"repository"`
	PullNumber      any                `json:"pullNumber" mapstructure:"pullNumber"`
	Reviewers       []ReviewerIdentity `json:"reviewers" mapstructure:"reviewers"`
	ExpectedHeadSHA string             `json:"expectedHeadSha" mapstructure:"expectedHeadSha"`
}

type ensureReviewersOutcome struct {
	Status         string             `json:"status"`
	Reason         string             `json:"reason,omitempty"`
	HeadSHA        string             `json:"headSha"`
	Requested      []ReviewerIdentity `json:"requested"`
	AlreadyCovered []ReviewerIdentity `json:"alreadyCovered"`
}

func (c *EnsurePullRequestReviewers) Name() string  { return "github.ensurePullRequestReviewers" }
func (c *EnsurePullRequestReviewers) Label() string { return "Ensure Pull Request Reviewers" }
func (c *EnsurePullRequestReviewers) Description() string {
	return "Request missing reviews after checking reviewer eligibility and the pull request head"
}
func (c *EnsurePullRequestReviewers) Documentation() string {
	return `Ensure desired human reviewers have a review request or an approval for the current commit. Recheck IDs and repository write access. Preserve unrelated requests. Read the pull request again immediately before mutation. Closed pull requests and unexpected commits emit a skipped outcome. Provider failures and ineligible identities fail the execution. Output includes requested and already-covered identities. The final read and mutation are not atomic.`
}
func (c *EnsurePullRequestReviewers) Icon() string  { return "github" }
func (c *EnsurePullRequestReviewers) Color() string { return "gray" }
func (c *EnsurePullRequestReviewers) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *EnsurePullRequestReviewers) Configuration() []configuration.Field {
	fields := (&ListPullRequestReviewCandidates{}).Configuration()
	return append(fields,
		configuration.Field{Name: "reviewers", Label: "Reviewers", Type: configuration.FieldTypeList, Required: true, TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{ItemLabel: "Reviewer", ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeObject, Schema: []configuration.Field{
			{Name: "id", Label: "GitHub user ID", Type: configuration.FieldTypeNumber, Required: true},
			{Name: "login", Label: "GitHub login", Type: configuration.FieldTypeString, Required: true},
		}}}}},
		configuration.Field{Name: "expectedHeadSha", Label: "Expected Head SHA", Type: configuration.FieldTypeString, Description: "Skip the request if the current head differs from this commit."},
	)
}
func (c *EnsurePullRequestReviewers) Setup(ctx core.SetupContext) error {
	fields, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return fmt.Errorf("configuration must be an object")
	}
	if expression, ok := fields["reviewers"].(string); !ok || !common.IsExpression(expression) {
		var config ensureReviewersConfiguration
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return err
		}
		if err := validateDesiredReviewers(config.Reviewers); err != nil {
			return err
		}
	}
	return (&ListPullRequestReviewCandidates{}).Setup(ctx)
}
func validateDesiredReviewers(reviewers []ReviewerIdentity) error {
	if len(reviewers) == 0 {
		return fmt.Errorf("at least one reviewer is required")
	}
	seenIDs := map[int64]bool{}
	seenLogins := map[string]bool{}
	for _, reviewer := range reviewers {
		login := strings.ToLower(strings.TrimSpace(reviewer.Login))
		if reviewer.ID <= 0 || login == "" || seenIDs[reviewer.ID] || seenLogins[login] {
			return fmt.Errorf("reviewers require distinct IDs and logins")
		}
		seenIDs[reviewer.ID], seenLogins[login] = true, true
	}
	return nil
}
func (c *EnsurePullRequestReviewers) Execute(ctx core.ExecutionContext) error {
	var config ensureReviewersConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if err := validateDesiredReviewers(config.Reviewers); err != nil {
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
	read := func() (*github.PullRequest, error) {
		pr, _, err := client.GetPullRequest(context.Background(), config.Repository, number)
		return pr, err
	}
	emit := func(outcome ensureReviewersOutcome) error {
		return ctx.ExecutionState.Emit("default", "github.pullRequestReviewersEnsured", []any{outcome})
	}
	skipped := func(pr *github.PullRequest) string {
		if pr.GetState() != "open" || pr.GetMerged() {
			return "closed"
		}
		if config.ExpectedHeadSHA != "" && pr.GetHead().GetSHA() != config.ExpectedHeadSHA {
			return "head_mismatch"
		}
		return ""
	}
	pr, err := read()
	if err != nil {
		return fmt.Errorf("read pull request: %w", err)
	}
	outcome := ensureReviewersOutcome{Requested: []ReviewerIdentity{}, AlreadyCovered: []ReviewerIdentity{}}
	if reason := skipped(pr); reason != "" {
		outcome.Status, outcome.Reason, outcome.HeadSHA = "skipped", reason, pr.GetHead().GetSHA()
		return emit(outcome)
	}
	users, err := client.ListHumanReviewers(context.Background(), config.Repository)
	if err != nil {
		return err
	}
	if pr.GetUser().GetLogin() == "" {
		return fmt.Errorf("pull request author is missing")
	}
	for _, reviewer := range config.Reviewers {
		if strings.EqualFold(reviewer.Login, pr.GetUser().GetLogin()) || !slices.ContainsFunc(users, func(user *github.User) bool {
			return user.GetID() == reviewer.ID && strings.EqualFold(user.GetLogin(), reviewer.Login)
		}) {
			return fmt.Errorf("reviewer is not eligible: %s", reviewer.Login)
		}
		permission, _, err := client.GetRepositoryPermissionLevel(context.Background(), config.Repository, reviewer.Login)
		if err != nil {
			return fmt.Errorf("read reviewer permission: %w", err)
		}
		if !slices.Contains([]string{"write", "maintain", "admin"}, permission.GetPermission()) {
			return fmt.Errorf("reviewer requires repository write access: %s", reviewer.Login)
		}
	}
	reviews, err := client.ListReviews(context.Background(), config.Repository, number)
	if err != nil {
		return fmt.Errorf("read pull request reviews: %w", err)
	}
	// Keep this as the last provider read: eligibility checks can outlive a push.
	pr, err = read()
	if err != nil {
		return fmt.Errorf("recheck pull request: %w", err)
	}
	outcome.HeadSHA = pr.GetHead().GetSHA()
	if reason := skipped(pr); reason != "" {
		outcome.Status, outcome.Reason = "skipped", reason
		return emit(outcome)
	}
	logins := []string{}
	for _, reviewer := range config.Reviewers {
		requested := slices.ContainsFunc(pr.RequestedReviewers, func(user *github.User) bool {
			return user.GetID() == reviewer.ID && strings.EqualFold(user.GetLogin(), reviewer.Login)
		})
		if requested || hasCurrentApproval(reviews, reviewer, outcome.HeadSHA) {
			outcome.AlreadyCovered = append(outcome.AlreadyCovered, reviewer)
			continue
		}
		logins = append(logins, reviewer.Login)
		outcome.Requested = append(outcome.Requested, reviewer)
	}
	outcome.Status = "already_covered"
	if len(logins) > 0 {
		if _, _, err := client.AddPullRequestReviewers(context.Background(), config.Repository, number, github.ReviewersRequest{Reviewers: logins}); err != nil {
			return fmt.Errorf("request pull request reviewers: %w", err)
		}
		outcome.Status = "requested"
	}
	return emit(outcome)
}
func hasCurrentApproval(reviews []*github.PullRequestReview, reviewer ReviewerIdentity, sha string) bool {
	var latest *github.PullRequestReview
	for _, review := range reviews {
		if review.GetUser().GetID() != reviewer.ID || !strings.EqualFold(review.GetUser().GetLogin(), reviewer.Login) || review.GetState() == "COMMENTED" || review.GetState() == "PENDING" {
			continue
		}
		if latest == nil || review.GetSubmittedAt().After(latest.GetSubmittedAt().Time) || (review.GetSubmittedAt().Time.Equal(latest.GetSubmittedAt().Time) && review.GetID() > latest.GetID()) {
			latest = review
		}
	}
	return latest != nil && latest.GetState() == "APPROVED" && latest.GetCommitID() == sha
}
func (c *EnsurePullRequestReviewers) Hooks() []core.Hook                      { return nil }
func (c *EnsurePullRequestReviewers) HandleHook(core.ActionHookContext) error { return nil }
func (c *EnsurePullRequestReviewers) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *EnsurePullRequestReviewers) Cancel(core.ExecutionContext) error { return nil }
func (c *EnsurePullRequestReviewers) Cleanup(core.SetupContext) error    { return nil }

func (c *EnsurePullRequestReviewers) ExampleOutput() map[string]any {
	return map[string]any{"type": "github.pullRequestReviewersEnsured", "timestamp": "2026-01-01T09:00:00Z", "data": map[string]any{
		"status": "requested", "headSha": "abc", "requested": []any{map[string]any{"id": 123, "login": "octocat"}}, "alreadyCovered": []any{},
	}}
}
