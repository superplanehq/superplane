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

type OnPipeline struct{}

type OnPipelineConfiguration struct {
	Project  string   `json:"project" mapstructure:"project"`
	Statuses []string `json:"statuses" mapstructure:"statuses"`
}

func (p *OnPipeline) Name() string {
	return "gitlab.onPipeline"
}

func (p *OnPipeline) Label() string {
	return "On Pipeline"
}

func (p *OnPipeline) Description() string {
	return "Listen to pipeline events from GitLab"
}

func (p *OnPipeline) Documentation() string {
	return `The On Pipeline trigger starts a workflow execution when pipeline events occur in a GitLab project.

## Configuration

- **Project** (required): GitLab project to monitor
- **Statuses** (required): Select which pipeline statuses to listen for. Default: success, failed, canceled.

## Outputs

- **Default channel**: Emits pipeline webhook payload data including status, ref, SHA, and project information

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPipeline) Icon() string {
	return "gitlab"
}

func (p *OnPipeline) Color() string {
	return "orange"
}

func (p *OnPipeline) Configuration() []configuration.Field {
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
			Name:     "statuses",
			Label:    "Statuses",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{PipelineStatusSuccess},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Success", Value: PipelineStatusSuccess},
						{Label: "Failed", Value: PipelineStatusFailed},
						{Label: "Canceled", Value: PipelineStatusCanceled},
						{Label: "Skipped", Value: PipelineStatusSkipped},
						{Label: "Manual", Value: PipelineStatusManual},
					},
				},
			},
		},
	}
}

func (p *OnPipeline) Setup(ctx core.TriggerContext) error {
	var config OnPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "pipeline",
		ProjectID: config.Project,
	})
}

func (p *OnPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPipeline) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Pipeline Hook" {
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

	status, ok := p.extractStatus(data)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("status missing from pipeline payload")
	}

	if len(config.Statuses) > 0 && !p.whitelistedStatus(ctx.Logger, status, config.Statuses) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("gitlab.pipeline", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPipeline) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (p *OnPipeline) extractStatus(data map[string]any) (string, bool) {
	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return "", false
	}

	status, ok := attrs["status"].(string)
	if !ok {
		return "", false
	}

	return status, true
}

func (p *OnPipeline) whitelistedStatus(logger *log.Entry, status string, allowedStatuses []string) bool {
	if !slices.Contains(allowedStatuses, status) {
		if logger != nil {
			logger.Infof("Pipeline status %s is not in the allowed list: %v", status, allowedStatuses)
		}
		return false
	}

	return true
}
