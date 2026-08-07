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

type OnMergeComment struct{}

type OnMergeCommentConfiguration struct {
	Project       string `json:"project" mapstructure:"project"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func (m *OnMergeComment) Name() string {
	return "gitlab.onMergeComment"
}

func (m *OnMergeComment) Label() string {
	return "On Merge Comment"
}

func (m *OnMergeComment) Description() string {
	return "Listen to merge request comment events from GitLab"
}

func (m *OnMergeComment) Documentation() string {
	return `The On Merge Comment trigger starts a workflow execution when a comment is added to a merge request in a GitLab project.

## Use Cases

- **Command processing**: Process slash commands in merge request comments (e.g., /deploy, /test)
- **Bot interactions**: Respond to merge request comments with automated actions
- **Notification systems**: Notify teams when important merge request comments are added

## Configuration

- **Project** (required): GitLab project to monitor
- **Content Filter** (optional): Regex pattern to filter comments by content (e.g., ` + "`/deploy`" + ` to only trigger on comments containing "/deploy")

## Event Data

Each comment event includes:
- **object_attributes**: Comment information including note body, author, and URL
- **merge_request**: Merge request the comment was added to
- **user**: User who added the comment
- **project**: Project information

Common expression paths:
- Merge request IID: ` + "`root().data.merge_request.iid`" + `
- Merge request title: ` + "`root().data.merge_request.title`" + `
- Comment body: ` + "`root().data.object_attributes.note`" + `
- Comment URL: ` + "`root().data.object_attributes.url`" + `

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (m *OnMergeComment) Icon() string {
	return "gitlab"
}

func (m *OnMergeComment) Color() string {
	return "orange"
}

func (m *OnMergeComment) Configuration() []configuration.Field {
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
			Placeholder: "e.g., /deploy",
			Description: "Optional regex pattern to filter comments by content",
		},
	}
}

func (m *OnMergeComment) Setup(ctx core.TriggerContext) error {
	var config OnMergeCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ContentFilter != "" {
		if _, err := regexp.Compile(config.ContentFilter); err != nil {
			return fmt.Errorf("invalid content filter pattern: %w", err)
		}
	}

	// Triggers register a project webhook at setup time, so the project must be
	// a concrete value rather than an expression resolved at runtime.
	if err := ensureConcreteProject(config.Project); err != nil {
		return err
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "note",
		ProjectID: config.Project,
	})
}

func (m *OnMergeComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *OnMergeComment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnMergeComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnMergeCommentConfiguration
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

	if !m.isMergeRequestComment(ctx.Logger, data) {
		return http.StatusOK, nil, nil
	}

	matched, err := matchesNoteContentFilter(config.ContentFilter, data)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	if !matched {
		ctx.Logger.Info("Comment does not match the content filter - ignoring")
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.mergeComment", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (m *OnMergeComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (m *OnMergeComment) isMergeRequestComment(logger *log.Entry, data map[string]any) bool {
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

	if noteableType != "MergeRequest" {
		logger.Infof("Comment is on a %s, not a merge request - ignoring", noteableType)
		return false
	}

	if noteType, _ := attrs["type"].(string); noteType == "DiffNote" {
		logger.Info("Comment is a diff note, not a regular merge request comment - ignoring")
		return false
	}

	return true
}
