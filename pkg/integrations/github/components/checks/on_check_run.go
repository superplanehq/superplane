package checks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type OnCheckRun struct{}

type OnCheckRunConfiguration struct {
	Repository       string                    `json:"repository" mapstructure:"repository"`
	Statuses         []string                  `json:"statuses" mapstructure:"statuses"`
	Conclusions      []string                  `json:"conclusions" mapstructure:"conclusions"`
	Names            []configuration.Predicate `json:"names" mapstructure:"names"`
	Branches         []configuration.Predicate `json:"branches" mapstructure:"branches"`
	PullRequestsOnly bool                      `json:"pullRequestsOnly" mapstructure:"pullRequestsOnly"`
}

func (t *OnCheckRun) Name() string {
	return "github.onCheckRun"
}

func (t *OnCheckRun) Label() string {
	return "On Check Run"
}

func (t *OnCheckRun) Description() string {
	return "Listen to GitHub Checks API check run events"
}

func (t *OnCheckRun) Documentation() string {
	return `The On Check Run trigger starts a workflow execution when a GitHub Checks API check run changes.

GitHub check runs are created by GitHub Apps and power many pull request checks such as Cloudflare Pages, DCO, Sourcery, and GitHub Actions check runs. They are separate from legacy Commit Statuses API statuses. Use On Commit Status for legacy commit status events.

## Use Cases

- **Pull request automation**: React when Checks API checks complete or fail
- **Quality gates**: Continue workflows only after a specific check run reaches the expected state
- **Notifications**: Notify teams when app-provided checks fail or require action
- **Merge orchestration**: Combine this trigger with List Check Runs For Ref to decide whether all checks are green

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Statuses**: Select check run statuses to listen for (queued, in_progress, completed, requested, waiting, pending)
- **Conclusions** *(optional)*: Filter completed check runs by conclusion
- **Names** *(optional)*: Filter check run names using predicates, e.g. equals "DCO" or matches "Cloudflare.*"
- **Branches** *(optional)*: Filter check suite head branches using predicates
- **Pull requests only**: Emit only check runs that GitHub associates with at least one pull request

## Event Data

Each event includes:
- **action**: The check_run webhook action
- **check_run**: Check run details, including name, status, conclusion, head SHA, app, pull requests, and URLs
- **repository**: Repository information
- **sender**: User or app actor associated with the webhook event

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.

## Notes

GitHub may return an empty pull_requests array for check runs created from forked repository pushes. When Pull requests only is enabled, those fork-originated check runs may be filtered out because GitHub does not include PR metadata in the webhook payload.`
}

func (t *OnCheckRun) Icon() string {
	return "github"
}

func (t *OnCheckRun) Color() string {
	return "gray"
}

func (t *OnCheckRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "statuses",
			Label:    "Statuses",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"queued", "in_progress", "completed", "requested", "waiting", "pending"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Queued", Value: "queued"},
						{Label: "In Progress", Value: "in_progress"},
						{Label: "Completed", Value: "completed"},
						{Label: "Requested", Value: "requested"},
						{Label: "Waiting", Value: "waiting"},
						{Label: "Pending", Value: "pending"},
					},
				},
			},
		},
		{
			Name:        "conclusions",
			Label:       "Conclusions",
			Description: "Optional. Filter completed check runs by conclusion.",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: checkRunConclusionOptions(),
				},
			},
		},
		{
			Name:        "names",
			Label:       "Names",
			Description: "Optional. Filter check run names.",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "branches",
			Label:       "Branches",
			Description: "Optional. Filter check suite head branch names.",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "pullRequestsOnly",
			Label:       "Pull requests only",
			Description: "Emit only check runs associated with at least one pull request.",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
		},
	}
}

func (t *OnCheckRun) Setup(ctx core.TriggerContext) error {
	if err := common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration); err != nil {
		return err
	}

	var config OnCheckRunConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventType:  "check_run",
		Repository: config.Repository,
	})
}

func (t *OnCheckRun) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnCheckRun) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnCheckRun) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, t.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	var config OnCheckRunConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		ctx.Logger.Errorf("Missing X-GitHub-Event header")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "check_run" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a check_run event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := common.VerifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if !matchesCheckRunFilters(payload, config) {
		ctx.Logger.Info("Ignoring event - check run did not match configured filters")
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("github.checkRun", payload); err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnCheckRun) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func matchesCheckRunFilters(payload map[string]any, config OnCheckRunConfiguration) bool {
	checkRun, ok := payload["check_run"].(map[string]any)
	if !ok {
		return false
	}

	status := extractStringValue(checkRun, "status")
	if !matchesConfiguredCheckRunStatus(status, config.Statuses) {
		return false
	}

	if !matchesConfiguredCheckRunConclusion(status, extractStringValue(checkRun, "conclusion"), config.Conclusions) {
		return false
	}

	if !matchesCheckRunName(extractStringValue(checkRun, "name"), config.Names) {
		return false
	}

	if !matchesCheckRunBranch(checkRun, config.Branches) {
		return false
	}

	if config.PullRequestsOnly && len(extractCheckRunPullRequestNumbers(checkRun)) == 0 {
		return false
	}

	return true
}

func matchesConfiguredCheckRunStatus(status string, allowedStatuses []string) bool {
	if len(allowedStatuses) == 0 {
		return true
	}

	return slices.Contains(allowedStatuses, status)
}

func matchesConfiguredCheckRunConclusion(status string, conclusion string, allowedConclusions []string) bool {
	if len(allowedConclusions) == 0 {
		return true
	}

	if status != "completed" {
		return true
	}

	if conclusion == "" {
		return false
	}

	return slices.Contains(allowedConclusions, conclusion)
}

func matchesCheckRunName(name string, allowedNames []configuration.Predicate) bool {
	if len(allowedNames) == 0 {
		return true
	}

	return configuration.MatchesAnyPredicate(allowedNames, name)
}

func matchesCheckRunBranch(checkRun map[string]any, allowedBranches []configuration.Predicate) bool {
	if len(allowedBranches) == 0 {
		return true
	}

	branch := extractCheckRunBranch(checkRun)
	if branch == "" {
		return false
	}

	return configuration.MatchesAnyPredicate(allowedBranches, branch)
}

func extractCheckRunBranch(checkRun map[string]any) string {
	checkSuite, ok := checkRun["check_suite"].(map[string]any)
	if ok {
		branch := extractStringValue(checkSuite, "head_branch")
		if branch != "" {
			return branch
		}
	}

	return extractFirstPullRequestHeadRef(checkRun["pull_requests"])
}

func extractFirstPullRequestHeadRef(value any) string {
	pullRequests, ok := value.([]any)
	if !ok || len(pullRequests) == 0 {
		return ""
	}

	pullRequest, ok := pullRequests[0].(map[string]any)
	if !ok {
		return ""
	}

	head, ok := pullRequest["head"].(map[string]any)
	if !ok {
		return ""
	}

	return extractStringValue(head, "ref")
}

func extractCheckRunPullRequestNumbers(checkRun map[string]any) []string {
	numbers := extractPullRequestNumbers(checkRun["pull_requests"])
	if len(numbers) > 0 {
		return numbers
	}

	checkSuite, ok := checkRun["check_suite"].(map[string]any)
	if !ok {
		return nil
	}

	return extractPullRequestNumbers(checkSuite["pull_requests"])
}

func extractPullRequestNumbers(value any) []string {
	pullRequests, ok := value.([]any)
	if !ok {
		return nil
	}

	numbers := []string{}
	for _, pullRequestRaw := range pullRequests {
		pullRequest, ok := pullRequestRaw.(map[string]any)
		if !ok {
			continue
		}

		switch number := pullRequest["number"].(type) {
		case float64:
			numbers = append(numbers, strconv.Itoa(int(number)))
		case string:
			numbers = append(numbers, number)
		}
	}

	return numbers
}

func extractStringValue(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}

func checkRunConclusionOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "Success", Value: "success"},
		{Label: "Failure", Value: "failure"},
		{Label: "Neutral", Value: "neutral"},
		{Label: "Cancelled", Value: "cancelled"},
		{Label: "Skipped", Value: "skipped"},
		{Label: "Timed Out", Value: "timed_out"},
		{Label: "Action Required", Value: "action_required"},
		{Label: "Stale", Value: "stale"},
	}
}
