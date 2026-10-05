package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Bitbucket publishes every pull request state change under its own event key, so an
// action filter maps one-to-one onto the webhook events we subscribe to.
const pullRequestEventPrefix = "pullrequest:"

var pullRequestActions = []configuration.FieldOption{
	{Label: "Opened", Value: "created"},
	{Label: "Updated", Value: "updated"},
	{Label: "Merged", Value: "fulfilled"},
	{Label: "Declined", Value: "rejected"},
	{Label: "Approved", Value: "approved"},
	{Label: "Approval removed", Value: "unapproved"},
	{Label: "Changes requested", Value: "changes_request_created"},
	{Label: "Changes request removed", Value: "changes_request_removed"},
}

type OnPullRequest struct{}

type OnPullRequestConfiguration struct {
	Repository     string                    `json:"repository" mapstructure:"repository"`
	Actions        []string                  `json:"actions" mapstructure:"actions"`
	TargetBranches []configuration.Predicate `json:"targetBranches" mapstructure:"targetBranches"`
}

func (p *OnPullRequest) Name() string {
	return "bitbucket.onPullRequest"
}

func (p *OnPullRequest) Label() string {
	return "On Pull Request"
}

func (p *OnPullRequest) Description() string {
	return "Listen to Bitbucket pull request events"
}

func (p *OnPullRequest) Documentation() string {
	return `The On Pull Request trigger starts a workflow execution when a pull request changes state in a Bitbucket repository.

## Use Cases

- **Preview environments**: Provision an ephemeral environment when a pull request is opened, and tear it down when it is merged or declined
- **Review automation**: Run checks, ask an agent for a review, or notify a channel when a pull request is opened or updated
- **Merge gates**: React to approvals and change requests to drive a deployment decision
- **Release automation**: Start a deploy when a pull request targeting ` + "`main`" + ` is merged

## Configuration

- **Repository** (required): The Bitbucket repository to monitor
- **Actions** (required): Which pull request events to listen for. Default: Opened.
- **Target Branches** (optional): Only fire when the pull request targets a matching branch. Leave empty to accept every target branch.

## Event Data

Each event includes:
- **pullrequest**: The full pull request, including ` + "`id`" + `, ` + "`title`" + `, ` + "`state`" + `, ` + "`source`" + `, ` + "`destination`" + ` and ` + "`links`" + `
- **repository**: Repository information
- **actor**: The user that triggered the event
- **approval** / **changes_request**: Present on approval and change-request events

Common expression paths:
- Pull request ID: ` + "`root().data.pullrequest.id`" + `
- Title: ` + "`root().data.pullrequest.title`" + `
- State: ` + "`root().data.pullrequest.state`" + `
- Source branch: ` + "`root().data.pullrequest.source.branch.name`" + `
- Target branch: ` + "`root().data.pullrequest.destination.branch.name`" + `
- Head commit: ` + "`root().data.pullrequest.source.commit.hash`" + `
- Pull request URL: ` + "`root().data.pullrequest.links.html.href`" + `

Note that Bitbucket reports the merged state as the ` + "`fulfilled`" + ` action and the declined state as ` + "`rejected`" + `.

## Webhook Setup

This trigger automatically sets up a Bitbucket webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPullRequest) Icon() string {
	return "bitbucket"
}

func (p *OnPullRequest) Color() string {
	return "blue"
}

func (p *OnPullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "actions",
			Label:       "Actions",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"created"},
			Description: "Which pull request events should start a run",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: pullRequestActions,
				},
			},
		},
		{
			Name:        "targetBranches",
			Label:       "Target Branches",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only fire for pull requests targeting a matching branch. Leave empty to accept any target branch.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (p *OnPullRequest) Setup(ctx core.TriggerContext) error {
	config := OnPullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	if err := validatePullRequestActions(config.Actions); err != nil {
		return err
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     pullRequestEventTypes(config.Actions),
		RepositorySlug: repo.Slug,
	})
}

func (p *OnPullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPullRequest) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnPullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// Verify the event type. Webhooks are shared between triggers on the same
	// repository, so deliveries for other event families are simply ignored.
	//
	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}

	if !slices.Contains(pullRequestEventTypes(config.Actions), eventKey) {
		return http.StatusOK, nil, nil
	}

	//
	// Verify the webhook signature.
	//
	if code, err := verifyWebhookSignature(ctx); err != nil {
		return code, nil, err
	}

	//
	// Parse the webhook payload.
	//
	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if len(config.TargetBranches) > 0 {
		targetBranch := pullRequestTargetBranch(data)
		if targetBranch == "" || !configuration.MatchesAnyPredicate(config.TargetBranches, targetBranch) {
			return http.StatusOK, nil, nil
		}
	}

	if err := ctx.Events.Emit("bitbucket.pullRequest", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (p *OnPullRequest) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func pullRequestEventTypes(actions []string) []string {
	eventTypes := make([]string, 0, len(actions))
	for _, action := range actions {
		eventTypes = append(eventTypes, pullRequestEventPrefix+action)
	}

	return eventTypes
}

func validatePullRequestActions(actions []string) error {
	for _, action := range actions {
		known := slices.ContainsFunc(pullRequestActions, func(option configuration.FieldOption) bool {
			return option.Value == action
		})

		if !known {
			return fmt.Errorf("unsupported pull request action %q", action)
		}
	}

	return nil
}

// pullRequestTargetBranch reads the destination branch out of a pull request payload.
func pullRequestTargetBranch(data map[string]any) string {
	pullRequest, ok := data["pullrequest"].(map[string]any)
	if !ok {
		return ""
	}

	destination, ok := pullRequest["destination"].(map[string]any)
	if !ok {
		return ""
	}

	branch, ok := destination["branch"].(map[string]any)
	if !ok {
		return ""
	}

	name, _ := branch["name"].(string)
	return strings.TrimSpace(name)
}
