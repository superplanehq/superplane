package pulls

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type OnPullRequest struct{}

type OnPullRequestConfiguration struct {
	Repository              string   `json:"repository" mapstructure:"repository"`
	Actions                 []string `json:"actions" mapstructure:"actions"`
	IgnoreDrafts            bool     `json:"ignoreDrafts" mapstructure:"ignoreDrafts"`
	OnlyFactoryPullRequests bool     `json:"onlyFactoryPullRequests" mapstructure:"onlyFactoryPullRequests"`
}

func (p *OnPullRequest) Name() string {
	return "github.onPullRequest"
}

func (p *OnPullRequest) Label() string {
	return "On Pull Request"
}

func (p *OnPullRequest) Description() string {
	return "Listen to pull request events"
}

func (p *OnPullRequest) Documentation() string {
	return `The On Pull Request trigger starts a workflow execution when pull request events occur in a GitHub repository.

## Use Cases

- **PR automation**: Automate actions when PRs are opened, merged, or closed
- **Code review workflows**: Trigger review processes or notifications
- **CI/CD integration**: Run tests or builds on PR events
- **Status updates**: Update systems when PR status changes

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Actions**: Select which PR actions to listen for (opened, edited, closed, synchronize, etc.)
- **Ignore draft pull requests**: Do not start a run when the pull request is a draft.
- **Only pull requests in this factory**: Start a run only when the pull request belongs to this factory. The event then includes the factory pull request and the task.

## Event Data

Each PR event includes:
- **action**: The action that triggered the event (opened, edited, closed, synchronize, etc.)
- **changes**: When action is edited, includes what changed (title, body, base branch)
- **pull_request**: Complete PR information including title, body, state, labels
- **repository**: Repository information
- **sender**: User who triggered the event
- **pullRequest**: The factory pull request. Present only when **Only pull requests in this factory** is on and the pull request belongs to this factory.
- **workOrder**: The task for that pull request. Present in the same case.

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPullRequest) Icon() string {
	return "github"
}

func (p *OnPullRequest) Color() string {
	return "gray"
}

func (p *OnPullRequest) Configuration() []configuration.Field {
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
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"opened"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Assigned", Value: "assigned"},
						{Label: "Unassigned", Value: "unassigned"},
						{Label: "Labeled", Value: "labeled"},
						{Label: "Unlabeled", Value: "unlabeled"},
						{Label: "Opened", Value: "opened"},
						{Label: "Edited", Value: "edited"},
						{Label: "Closed", Value: "closed"},
						{Label: "Reopened", Value: "reopened"},
						{Label: "Synchronize", Value: "synchronize"},
						{Label: "Converted to draft", Value: "converted_to_draft"},
						{Label: "Locked", Value: "locked"},
						{Label: "Unlocked", Value: "unlocked"},
						{Label: "Enqueued", Value: "enqueued"},
						{Label: "Dequeued", Value: "dequeued"},
						{Label: "Milestoned", Value: "milestoned"},
						{Label: "Demilestoned", Value: "demilestoned"},
						{Label: "Ready for review", Value: "ready_for_review"},
						{Label: "Review requested", Value: "review_requested"},
						{Label: "Review request removed", Value: "review_request_removed"},
						{Label: "Auto merge enabled", Value: "auto_merge_enabled"},
						{Label: "Auto merge disabled", Value: "auto_merge_disabled"},
					},
				},
			},
		},
		{
			Name:        "ignoreDrafts",
			Label:       "Ignore draft pull requests",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Do not start a run when the pull request is a draft.",
		},
		{
			Name:        "onlyFactoryPullRequests",
			Label:       "Only pull requests in this factory",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Start a run only when the pull request belongs to this factory.",
		},
	}
}

func (p *OnPullRequest) Setup(ctx core.TriggerContext) error {
	err := common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventType:  "pull_request",
		Repository: config.Repository,
	})
}

func (p *OnPullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPullRequest) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, p.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config := OnPullRequestConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		ctx.Logger.Errorf("Missing X-GitHub-Event header")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "pull_request" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a pull_request event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := common.VerifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if !common.WhitelistedAction(data, config.Actions) {
		action, ok := common.ExtractAction(data)
		if !ok {
			ctx.Logger.Info("Ignoring event - without a valid action")
			return http.StatusOK, nil, nil
		}

		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	if config.IgnoreDrafts && pullRequestIsDraft(data) {
		ctx.Logger.Info("Ignoring event - pull request is a draft")
		return http.StatusOK, nil, nil
	}

	if config.OnlyFactoryPullRequests {
		matched, code, matchErr := matchFactoryPullRequest(ctx, data)
		if matchErr != nil || !matched {
			return code, nil, matchErr
		}
	}

	err = ctx.Events.Emit("github.pullRequest", data)
	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func matchFactoryPullRequest(ctx core.WebhookRequestContext, data map[string]any) (bool, int, error) {
	if ctx.Factory == nil {
		return false, http.StatusInternalServerError, errors.New("app is not owned by a factory")
	}

	lookup, ok := factoryPullRequestLookup(data)
	if !ok {
		ctx.Logger.Info("Ignoring event - pull request identity is incomplete")
		return false, http.StatusOK, nil
	}

	match, err := ctx.Factory.FindPullRequest(lookup)
	if err != nil {
		if errors.Is(err, core.ErrPullRequestNotFound) {
			ctx.Logger.Info("Ignoring event - pull request is not in this factory")
			return false, http.StatusOK, nil
		}
		return false, http.StatusInternalServerError, err
	}

	data["pullRequest"] = match.PullRequest
	data["workOrder"] = match.WorkOrder
	return true, http.StatusOK, nil
}

func factoryPullRequestLookup(event map[string]any) (core.FindPullRequestParams, bool) {
	repository, _ := event["repository"].(map[string]any)
	fullName, _ := repository["full_name"].(string)
	pullRequest, _ := event["pull_request"].(map[string]any)
	number, ok := int64FromJSON(pullRequest["number"])
	pageURL, _ := pullRequest["html_url"].(string)
	if fullName == "" || !ok || number <= 0 {
		return core.FindPullRequestParams{}, false
	}

	return core.FindPullRequestParams{
		Provider:   "github",
		Repository: fullName,
		Number:     number,
		URL:        pageURL,
	}, true
}

func (p *OnPullRequest) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func pullRequestIsDraft(data map[string]any) bool {
	pullRequest, ok := data["pull_request"].(map[string]any)
	if !ok {
		return false
	}

	draft, ok := pullRequest["draft"].(bool)
	return ok && draft
}
