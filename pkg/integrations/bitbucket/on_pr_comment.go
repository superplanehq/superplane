package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

var pullRequestCommentActions = []configuration.FieldOption{
	{Label: "Created", Value: "comment_created"},
	{Label: "Updated", Value: "comment_updated"},
	{Label: "Deleted", Value: "comment_deleted"},
}

type OnPRComment struct{}

type OnPRCommentConfiguration struct {
	Repository    string   `json:"repository" mapstructure:"repository"`
	Actions       []string `json:"actions" mapstructure:"actions"`
	ContentFilter string   `json:"contentFilter" mapstructure:"contentFilter"`
}

func (c *OnPRComment) Name() string {
	return "bitbucket.onPRComment"
}

func (c *OnPRComment) Label() string {
	return "On Pull Request Comment"
}

func (c *OnPRComment) Description() string {
	return "Listen to comments on Bitbucket pull requests"
}

func (c *OnPRComment) Documentation() string {
	return `The On Pull Request Comment trigger starts a workflow execution when someone comments on a Bitbucket pull request.

## Use Cases

- **ChatOps**: Let reviewers run a workflow by commenting ` + "`/deploy`" + ` or ` + "`/preview`" + ` on a pull request
- **Agent hand-off**: Send a comment to a coding agent and post the result back on the pull request
- **Review tracking**: Notify a channel when a discussion starts on an open pull request

## Configuration

- **Repository** (required): The Bitbucket repository to monitor
- **Actions** (required): Which comment events to listen for. Default: Created.
- **Content Filter** (optional): Regex pattern the comment body must match, e.g. ` + "`^/deploy`" + `. Leave empty to accept every comment.

## Event Data

Each event includes:
- **comment**: The comment, including ` + "`id`" + ` and ` + "`content.raw`" + `
- **pullrequest**: The pull request the comment belongs to
- **repository**: Repository information
- **actor**: The user that wrote the comment

Common expression paths:
- Comment body: ` + "`root().data.comment.content.raw`" + `
- Comment author: ` + "`root().data.actor.display_name`" + `
- Pull request ID: ` + "`root().data.pullrequest.id`" + `
- Source branch: ` + "`root().data.pullrequest.source.branch.name`" + `

## Webhook Setup

This trigger automatically sets up a Bitbucket webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (c *OnPRComment) Icon() string {
	return "bitbucket"
}

func (c *OnPRComment) Color() string {
	return "blue"
}

func (c *OnPRComment) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "actions",
			Label:       "Actions",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"comment_created"},
			Description: "Which comment events should start a run",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: pullRequestCommentActions,
				},
			},
		},
		{
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., /deploy",
			Description: "Optional regex pattern to filter comments by content",
		},
	}
}

func (c *OnPRComment) Setup(ctx core.TriggerContext) error {
	config := OnPRCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	for _, action := range config.Actions {
		known := slices.ContainsFunc(pullRequestCommentActions, func(option configuration.FieldOption) bool {
			return option.Value == action
		})

		if !known {
			return fmt.Errorf("unsupported comment action %q", action)
		}
	}

	if config.ContentFilter != "" {
		if _, err := regexp.Compile(config.ContentFilter); err != nil {
			return fmt.Errorf("invalid content filter pattern: %w", err)
		}
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

func (c *OnPRComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *OnPRComment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (c *OnPRComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnPRCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}

	if !slices.Contains(pullRequestEventTypes(config.Actions), eventKey) {
		return http.StatusOK, nil, nil
	}

	if code, err := verifyWebhookSignature(ctx); err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	matched, err := c.matchesContentFilter(config.ContentFilter, data)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if !matched {
		ctx.Logger.Info("Comment does not match the content filter - ignoring")
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("bitbucket.pullRequestComment", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (c *OnPRComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// matchesContentFilter checks the comment body against the regex filter. An empty
// filter always matches, and a comment delivered without a body never does.
func (c *OnPRComment) matchesContentFilter(filter string, data map[string]any) (bool, error) {
	if filter == "" {
		return true, nil
	}

	comment, ok := data["comment"].(map[string]any)
	if !ok {
		return false, nil
	}

	content, ok := comment["content"].(map[string]any)
	if !ok {
		return false, nil
	}

	raw, ok := content["raw"].(string)
	if !ok {
		return false, nil
	}

	matched, err := regexp.MatchString(filter, raw)
	if err != nil {
		return false, fmt.Errorf("invalid content filter pattern: %w", err)
	}

	return matched, nil
}
