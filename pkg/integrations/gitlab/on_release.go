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

type OnRelease struct{}

type OnReleaseConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (r *OnRelease) Name() string {
	return "gitlab.onRelease"
}

func (r *OnRelease) Label() string {
	return "On Release"
}

func (r *OnRelease) Description() string {
	return "Listen to release events from GitLab"
}

func (r *OnRelease) Documentation() string {
	return `The On Release trigger starts a workflow execution when release events occur in a GitLab project.

## Configuration

- **Project** (required): GitLab project to monitor
- **Actions** (required): Select which release actions to listen for. Default: create.

## Outputs

- **Default channel**: Emits release payload data with action and release metadata`
}

func (r *OnRelease) Icon() string {
	return "gitlab"
}

func (r *OnRelease) Color() string {
	return "orange"
}

func (r *OnRelease) Configuration() []configuration.Field {
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
						{Label: "Updated", Value: "update"},
						{Label: "Deleted", Value: "delete"},
					},
				},
			},
		},
	}
}

func (r *OnRelease) Setup(ctx core.TriggerContext) error {
	var config OnReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "releases",
		ProjectID: config.Project,
	})
}

func (r *OnRelease) Actions() []core.Action {
	return []core.Action{}
}

func (r *OnRelease) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (r *OnRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Release Hook" {
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

	if len(config.Actions) > 0 && !r.whitelistedAction(ctx.Logger, data, config.Actions) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("gitlab.release", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (r *OnRelease) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (r *OnRelease) extractAction(data map[string]any) (string, bool) {
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

func (r *OnRelease) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
	action, ok := r.extractAction(data)
	if !ok {
		return false
	}

	if !slices.Contains(allowedActions, action) {
		logger.Infof("Action %s is not in the allowed list: %v", action, allowedActions)
		return false
	}

	return true
}
