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

type OnMilestone struct{}

type OnMilestoneConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (m *OnMilestone) Name() string {
	return "gitlab.onMilestone"
}

func (m *OnMilestone) Label() string {
	return "On Milestone"
}

func (m *OnMilestone) Description() string {
	return "Listen to milestone events from GitLab"
}

func (m *OnMilestone) Documentation() string {
	return `The On Milestone trigger starts a workflow execution when milestone events occur in a GitLab project.

## Configuration

- **Project** (required): GitLab project to monitor
- **Actions** (required): Select which milestone actions to listen for. Default: create.

## Outputs

- **Default channel**: Emits milestone payload data with action, project, and object attributes`
}

func (m *OnMilestone) Icon() string {
	return "gitlab"
}

func (m *OnMilestone) Color() string {
	return "orange"
}

func (m *OnMilestone) Configuration() []configuration.Field {
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
			Default:  []string{"create"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "create"},
						{Label: "Closed", Value: "close"},
						{Label: "Reopened", Value: "reopen"},
						{Label: "Deleted", Value: "delete"},
					},
				},
			},
		},
	}
}

func (m *OnMilestone) Setup(ctx core.TriggerContext) error {
	var config OnMilestoneConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "milestone",
		ProjectID: config.Project,
	})
}

func (m *OnMilestone) Actions() []core.Action {
	return []core.Action{}
}

func (m *OnMilestone) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnMilestone) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnMilestoneConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Milestone Hook" {
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

	if err := ctx.Events.Emit("gitlab.milestone", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (m *OnMilestone) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (m *OnMilestone) extractAction(data map[string]any) (string, bool) {
	action, ok := data["action"].(string)
	if ok {
		return action, true
	}

	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return "", false
	}

	action, ok = attrs["action"].(string)
	if !ok {
		return "", false
	}

	return action, true
}

func (m *OnMilestone) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
	action, ok := m.extractAction(data)
	if !ok {
		return false
	}

	if !slices.Contains(allowedActions, action) {
		logger.Infof("Action %s is not in the allowed list: %v", action, allowedActions)
		return false
	}

	return true
}
