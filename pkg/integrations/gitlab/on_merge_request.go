package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnMergeRequest struct{}

type OnMergeRequestConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (m *OnMergeRequest) Name() string {
	return "gitlab.onMergeRequest"
}

func (m *OnMergeRequest) Label() string {
	return "On Merge Request"
}

func (m *OnMergeRequest) Description() string {
	return "Listen to merge request events from GitLab"
}

func (m *OnMergeRequest) Documentation() string {
	return `The On Merge Request trigger starts a workflow execution when merge request events occur in a GitLab project.

## Use Cases

- **MR automation**: Automate actions when merge requests are opened, merged, or closed
- **Code review workflows**: Trigger review processes or notifications
- **CI/CD integration**: Run tests, builds, or preview environments on merge request events
- **Status updates**: Update systems when merge request status changes

## Configuration

- **Project** (required): GitLab project to monitor
- **Actions** (required): Select which merge request actions to listen for (open, close, reopen, update, approved, merge, etc.). Default: open.

## Event Data

Each merge request event includes:
- **object_attributes**: Complete merge request information including title, description, state, action, source/target branches, and URL
- **changes**: When the merge request is updated, includes what changed (title, description, labels, etc.)
- **assignees**: Users assigned to the merge request
- **reviewers**: Users requested to review the merge request
- **labels**: Labels applied to the merge request
- **project**: Project information
- **repository**: Repository information
- **user**: User who triggered the event

Common expression paths:
- Merge request IID: ` + "`root().data.object_attributes.iid`" + `
- Merge request title: ` + "`root().data.object_attributes.title`" + `
- Action: ` + "`root().data.object_attributes.action`" + `
- State: ` + "`root().data.object_attributes.state`" + `
- Source branch: ` + "`root().data.object_attributes.source_branch`" + `
- Target branch: ` + "`root().data.object_attributes.target_branch`" + `
- Merge request URL: ` + "`root().data.object_attributes.url`" + `

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (m *OnMergeRequest) Icon() string {
	return "gitlab"
}

func (m *OnMergeRequest) Color() string {
	return "orange"
}

func (m *OnMergeRequest) Configuration() []configuration.Field {
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
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"open"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Opened", Value: "open"},
						{Label: "Closed", Value: "close"},
						{Label: "Reopened", Value: "reopen"},
						{Label: "Updated", Value: "update"},
						{Label: "Approval Added", Value: "approval"},
						{Label: "Approved", Value: "approved"},
						{Label: "Approval Removed", Value: "unapproval"},
						{Label: "Unapproved", Value: "unapproved"},
						{Label: "Merged", Value: "merge"},
					},
				},
			},
		},
	}
}

func (m *OnMergeRequest) Setup(ctx core.TriggerContext) error {
	var config OnMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "merge_requests",
		ProjectID: config.Project,
	})
}

func (m *OnMergeRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *OnMergeRequest) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Merge Request Hook" {
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

	if len(config.Actions) > 0 && !m.whitelistedAction(ctx.Logger, data, config.Actions) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.mergeRequest", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (m *OnMergeRequest) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (m *OnMergeRequest) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false
	}

	action, ok := attrs["action"].(string)
	if !ok {
		return false
	}

	if !slices.Contains(allowedActions, action) {
		logger.Infof("Action %s is not in the allowed list: %v", action, allowedActions)
		return false
	}

	return true
}
