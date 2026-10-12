package bitbucket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPullRequest struct{}

type OnPullRequestConfiguration struct {
	Repository              string   `json:"repository" mapstructure:"repository"`
	Actions                 []string `json:"actions" mapstructure:"actions"`
	OnlyFactoryPullRequests bool     `json:"onlyFactoryPullRequests" mapstructure:"onlyFactoryPullRequests"`
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
	return `The On Pull Request trigger starts a workflow execution when pull request events occur in a Bitbucket repository.

## Use Cases

- **PR automation**: Automate actions when PRs are opened, updated, approved, merged, or declined
- **Code review workflows**: Trigger review processes or notifications

## Configuration

- **Repository**: Select the Bitbucket repository to monitor
- **Actions**: Select which pull request actions to listen for (created, updated, approved, unapproved, merged, declined)

## Event Data

Each PR event includes:
- **action**: The normalized action (created, updated, approved, unapproved, merged, declined)
- **pullrequest**: Complete PR information including title, state, source and destination branches
- **repository**: Repository information
- **actor**: User who triggered the event

## Webhook Setup

This trigger automatically sets up a Bitbucket webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed. One repository webhook is shared between triggers.`
}

func (p *OnPullRequest) Icon() string {
	return "bitbucket"
}

func (p *OnPullRequest) Color() string {
	return "blue"
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
			Default:  []string{"created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
						{Label: "Approved", Value: "approved"},
						{Label: "Unapproved", Value: "unapproved"},
						{Label: "Merged", Value: "merged"},
						{Label: "Declined", Value: "declined"},
					},
				},
			},
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
	config := OnPullRequestConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	keys, err := pullRequestEventKeysForActions(config.Actions)
	if err != nil {
		return err
	}
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     keys,
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
	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}

	if !strings.HasPrefix(eventKey, "pullrequest:") {
		return http.StatusOK, nil, nil
	}

	if code, err := verifyBitbucketSignature(ctx); err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	config := OnPullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if !payloadRepositoryMatchesTrigger(ctx, config.Repository) {
		return http.StatusOK, nil, nil
	}

	event, ok := normalizeBitbucketPullRequestEvent(eventKey, data)
	if !ok {
		return http.StatusOK, nil, nil
	}

	if len(config.Actions) > 0 && !slices.ContainsFunc(config.Actions, func(action string) bool {
		return strings.EqualFold(strings.TrimSpace(action), event["action"].(string))
	}) {
		return http.StatusOK, nil, nil
	}

	if config.OnlyFactoryPullRequests {
		matched, code, matchErr := matchFactoryPullRequest(ctx, event)
		if matchErr != nil || !matched {
			return code, nil, matchErr
		}
	}

	if err := ctx.Events.Emit("bitbucket.pullRequest", event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func matchFactoryPullRequest(ctx core.WebhookRequestContext, event map[string]any) (bool, int, error) {
	if ctx.Factory == nil {
		return false, http.StatusInternalServerError, fmt.Errorf("app is not owned by a factory")
	}

	lookup, ok := factoryPullRequestLookup(event)
	if !ok {
		return false, http.StatusOK, nil
	}

	match, err := ctx.Factory.FindPullRequest(lookup)
	if err != nil {
		if errors.Is(err, core.ErrPullRequestNotFound) {
			return false, http.StatusOK, nil
		}
		return false, http.StatusInternalServerError, err
	}

	event["pullRequest"] = match.PullRequest
	event["workOrder"] = match.WorkOrder
	return true, http.StatusOK, nil
}

func factoryPullRequestLookup(event map[string]any) (core.FindPullRequestParams, bool) {
	repository, _ := event["repository"].(map[string]any)
	fullName, _ := repository["full_name"].(string)
	pullRequest, _ := event["pullrequest"].(map[string]any)
	number, ok := int64FromJSON(pullRequest["id"])
	links, _ := pullRequest["links"].(map[string]any)
	html, _ := links["html"].(map[string]any)
	pageURL, _ := html["href"].(string)
	if fullName == "" || !ok || number <= 0 {
		return core.FindPullRequestParams{}, false
	}

	return core.FindPullRequestParams{
		Provider:   "bitbucket",
		Repository: fullName,
		Number:     number,
		URL:        pageURL,
	}, true
}

func int64FromJSON(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func (p *OnPullRequest) Cleanup(ctx core.TriggerContext) error {
	return nil
}
