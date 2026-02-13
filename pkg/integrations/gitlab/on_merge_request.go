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

## Configuration

- **Project** (required): GitLab project to monitor
- **Actions** (required): Select which merge request actions to listen for (open, close, merge, etc.). Default: open.

## Outputs

- **Default channel**: Emits merge request payload data with action, project, and object attributes`
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

func (m *OnMergeRequest) Actions() []core.Action {
	return []core.Action{}
}

func (m *OnMergeRequest) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Merge Request Hook" {
		return http.StatusOK, nil
	}

	code, err := verifyWebhookToken(ctx)
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if len(config.Actions) > 0 && !m.whitelistedAction(ctx.Logger, data, config.Actions) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("gitlab.mergeRequest", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
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
