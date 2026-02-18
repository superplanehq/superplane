package harness

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPipelineCompleted struct{}

type OnPipelineCompletedConfiguration struct {
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type OnPipelineCompletedMetadata struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

func (t *OnPipelineCompleted) Name() string {
	return "harness.onPipelineCompleted"
}

func (t *OnPipelineCompleted) Label() string {
	return "On Pipeline Completed"
}

func (t *OnPipelineCompleted) Description() string {
	return "Runs when a Harness pipeline completes"
}

func (t *OnPipelineCompleted) Documentation() string {
	return `The On Pipeline Completed trigger starts a workflow when a Harness pipeline finishes execution.

## Use Cases

- **Failure alerts**: Notify Slack or create a ticket when a pipeline fails
- **Pipeline orchestration**: Chain workflows based on pipeline completion
- **Status monitoring**: Track CI/CD pipeline results across projects
- **Post-deployment actions**: Run follow-up tasks after successful deployments

## Setup

This trigger requires a webhook to be configured in Harness:

1. In SuperPlane, add this trigger to your workflow and copy the webhook URL displayed
2. In Harness, go to your pipeline's **Notify** settings
3. Add a new notification rule with **Webhook** as the channel
4. Paste the SuperPlane webhook URL
5. Select the pipeline events you want to listen for (e.g. Pipeline Success, Pipeline Failed)

## Configuration

- **Event Types**: Optionally filter which pipeline events to process (e.g. PipelineSuccess, PipelineFailed). Leave empty to receive all pipeline completion events.

## Event Data

Each pipeline completion event includes:
- **pipelineName**: Name of the pipeline
- **pipelineIdentifier**: Pipeline identifier
- **projectIdentifier**: Project the pipeline belongs to
- **orgIdentifier**: Organization identifier
- **eventType**: Type of event (PipelineSuccess, PipelineFailed, PipelineEnd)
- **executionUrl**: Direct link to the execution in Harness
- **triggeredBy**: Who or what triggered the pipeline
- **startTime/endTime**: Execution timestamps`
}

func (t *OnPipelineCompleted) Icon() string {
	return "workflow"
}

func (t *OnPipelineCompleted) Color() string {
	return "blue"
}

func (t *OnPipelineCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "eventTypes",
			Label:       "Event Types",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Filter by event types (e.g. PipelineSuccess, PipelineFailed, PipelineEnd). Leave empty to receive all pipeline completion events.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Event Type",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (t *OnPipelineCompleted) Setup(ctx core.TriggerContext) error {
	var metadata OnPipelineCompletedMetadata
	if ctx.Metadata != nil {
		_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	}

	// If webhook URL is already set, skip setup.
	if metadata.WebhookURL != "" {
		return nil
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook URL: %w", err)
	}

	metadata.WebhookURL = webhookURL

	if ctx.Metadata != nil {
		return ctx.Metadata.Set(metadata)
	}

	return nil
}

func (t *OnPipelineCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var payload map[string]any
	err := json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventData, ok := payload["eventData"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("eventData missing from webhook payload")
	}

	eventType, _ := eventData["eventType"].(string)
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("eventType missing from webhook payload")
	}

	err = ctx.Events.Emit("harness.pipeline.completed", payload)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnPipelineCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnPipelineCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnPipelineCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}
