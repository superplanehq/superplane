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

type OnMRDiffNote struct{}

type OnMRDiffNoteConfiguration struct {
	Project       string `json:"project" mapstructure:"project"`
	ContentFilter string `json:"contentFilter" mapstructure:"contentFilter"`
}

func (m *OnMRDiffNote) Name() string {
	return "gitlab.onMRDiffNote"
}

func (m *OnMRDiffNote) Label() string {
	return "On MR Diff Note"
}

func (m *OnMRDiffNote) Description() string {
	return "Listen to merge request diff (inline code review) comment events from GitLab"
}

func (m *OnMRDiffNote) Documentation() string {
	return `The On MR Diff Note trigger starts a workflow execution when an inline comment is added to a merge request's diff in a GitLab project.

Diff notes (also known as inline comments) are comments left on a specific line of a merge request's changes, as opposed to a regular top-level merge request comment. Use ` + "`gitlab.onMergeComment`" + ` if you want to react to regular merge request discussion comments instead.

## Use Cases

- **Review triage**: Have an agent triage a merge request review by reading the diff note together with the file and line it was left on
- **Command processing**: Process slash commands left as inline review comments (e.g., /fix, /explain)
- **Notification systems**: Notify teams when important inline review comments are added

## Configuration

- **Project** (required): GitLab project to monitor
- **Content Filter** (optional): Regex pattern to filter comments by content (e.g., ` + "`/fix`" + ` to only trigger on comments containing "/fix")

## Event Data

Each diff note event includes:
- **object_attributes**: Comment information including note body, author, URL, and the diff **position** (file paths, line numbers, and commit SHAs the comment is anchored to)
- **merge_request**: Merge request the comment was added to
- **user**: User who added the comment
- **project**: Project information

Common expression paths:
- Merge request IID: ` + "`root().data.merge_request.iid`" + `
- Merge request title: ` + "`root().data.merge_request.title`" + `
- Comment body: ` + "`root().data.object_attributes.note`" + `
- Diff file path: ` + "`root().data.object_attributes.position.new_path`" + `
- Diff line number: ` + "`root().data.object_attributes.position.new_line`" + `

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (m *OnMRDiffNote) Icon() string {
	return "gitlab"
}

func (m *OnMRDiffNote) Color() string {
	return "orange"
}

func (m *OnMRDiffNote) Configuration() []configuration.Field {
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
			Placeholder: "e.g., /fix",
			Description: "Optional regex pattern to filter comments by content",
		},
	}
}

func (m *OnMRDiffNote) Setup(ctx core.TriggerContext) error {
	var config OnMRDiffNoteConfiguration
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

func (m *OnMRDiffNote) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *OnMRDiffNote) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnMRDiffNote) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnMRDiffNoteConfiguration
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

	if !m.isDiffNoteComment(ctx.Logger, data) {
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

	if err := ctx.Events.Emit("gitlab.mrDiffNote", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (m *OnMRDiffNote) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (m *OnMRDiffNote) isDiffNoteComment(logger *log.Entry, data map[string]any) bool {
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

	noteType, _ := attrs["type"].(string)
	if noteType != "DiffNote" {
		logger.Info("Comment is not a diff note - ignoring")
		return false
	}

	return true
}

func (m *OnMRDiffNote) matchesContentFilter(filter string, data map[string]any) (bool, error) {
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
