package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueComment struct{}

type OnIssueCommentConfiguration struct {
	Project       string `json:"project" mapstructure:"project"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func (m *OnIssueComment) Name() string {
	return "gitlab.onIssueComment"
}

func (m *OnIssueComment) Label() string {
	return "On Issue Comment"
}

func (m *OnIssueComment) Description() string {
	return "Listen to issue comment events from GitLab"
}

func (m *OnIssueComment) Documentation() string {
	return `The On Issue Comment trigger starts a workflow execution when a comment is added to an issue in a GitLab project.

## Use Cases

- **Command processing**: Process slash commands in issue comments (e.g., ` + "`/sp-investigate`" + ` to trigger an agent that investigates whether the issue is worth fixing and rates its urgency)
- **Bot interactions**: Respond to issue comments with automated actions
- **Notification systems**: Notify teams when important issue comments are added

## Configuration

- **Project** (required): GitLab project to monitor
- **Content Filter** (optional): Regex pattern to filter comments by content (e.g., ` + "`/sp-investigate`" + ` to only trigger on comments containing "/sp-investigate")

## Event Data

Each comment event includes:
- **object_attributes**: Comment information including note body, author, and URL
- **issue**: Issue the comment was added to
- **user**: User who added the comment
- **project**: Project information

Common expression paths:
- Issue IID: ` + "`root().data.issue.iid`" + `
- Issue title: ` + "`root().data.issue.title`" + `
- Comment body: ` + "`root().data.object_attributes.note`" + `
- Comment URL: ` + "`root().data.object_attributes.url`" + `

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (m *OnIssueComment) Icon() string {
	return "gitlab"
}

func (m *OnIssueComment) Color() string {
	return "orange"
}

func (m *OnIssueComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., /sp-investigate",
			Description: "Optional regex pattern to filter comments by content",
		},
	}
}

func (m *OnIssueComment) Setup(ctx core.TriggerContext) error {
	var config OnIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ContentFilter != "" {
		if _, err := regexp.Compile(config.ContentFilter); err != nil {
			return fmt.Errorf("invalid content filter pattern: %w", err)
		}
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "note",
		ProjectID: config.Project,
	})
}

func (m *OnIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *OnIssueComment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Note Hook" {
		return http.StatusOK, nil, nil
	}

	code, err := verifyWebhookToken(ctx)
	if err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if !m.isIssueComment(ctx.Logger, data) {
		return http.StatusOK, nil, nil
	}

	matched, err := m.matchesContentFilter(config.ContentFilter, data)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	if !matched {
		ctx.Logger.Info("Comment does not match the content filter - ignoring")
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.issueComment", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (m *OnIssueComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (m *OnIssueComment) isIssueComment(logger *log.Entry, data map[string]any) bool {
	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false
	}

	if system, ok := attrs["system"].(bool); ok && system {
		logger.Info("Comment is a system-generated note - ignoring")
		return false
	}

	noteableType, ok := attrs["noteable_type"].(string)
	if !ok {
		return false
	}

	if noteableType != "Issue" {
		logger.Infof("Comment is on a %s, not an issue - ignoring", noteableType)
		return false
	}

	return true
}

func (m *OnIssueComment) matchesContentFilter(filter string, data map[string]any) (bool, error) {
	if filter == "" {
		return true, nil
	}

	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false, nil
	}

	note, ok := attrs["note"].(string)
	if !ok {
		return false, nil
	}

	matched, err := regexp.MatchString(filter, note)
	if err != nil {
		return false, fmt.Errorf("invalid content filter pattern: %w", err)
	}

	return matched, nil
}
