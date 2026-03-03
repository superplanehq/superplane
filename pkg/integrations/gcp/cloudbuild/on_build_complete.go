package cloudbuild

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EmittedEventType = "gcp.cloudbuild.build"
	SubscriptionType = "cloudbuild"
)

type OnBuildComplete struct{}

type OnBuildCompleteConfiguration struct {
	Statuses  []string `json:"statuses" mapstructure:"statuses"`
	TriggerID string   `json:"triggerId" mapstructure:"triggerId"`
}

type OnBuildCompleteMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

func (t *OnBuildComplete) Name() string {
	return "gcp.cloudbuild.onBuildComplete"
}

func (t *OnBuildComplete) Label() string {
	return "Cloud Build • On Build Complete"
}

func (t *OnBuildComplete) Description() string {
	return "Trigger a workflow when a GCP Cloud Build build reaches a terminal status"
}

func (t *OnBuildComplete) Documentation() string {
	return `The On Build Complete trigger starts a workflow execution when a GCP Cloud Build build finishes.

**Trigger behavior:** SuperPlane subscribes to the ` + "`cloud-builds`" + ` Pub/Sub topic that Cloud Build automatically publishes to. Build notifications are pushed to SuperPlane and matched to this trigger.

## Use Cases

- **Post-build automation**: Deploy artifacts, send notifications, or update tickets after a build succeeds
- **Failure handling**: Alert teams or create incidents when builds fail
- **Build pipelines**: Chain multiple build steps across different projects

## Setup

**Required GCP setup:** Ensure the **Cloud Build API** is enabled in your project. The service account used by the integration must have ` + "`roles/pubsub.admin`" + ` to create a push subscription on the ` + "`cloud-builds`" + ` topic.

## Event Data

Each event contains the full Cloud Build resource, including ` + "`id`" + `, ` + "`status`" + ` (SUCCESS, FAILURE, CANCELLED, TIMEOUT), ` + "`buildTriggerId`" + `, ` + "`logUrl`" + `, ` + "`createTime`" + `, ` + "`finishTime`" + `, and more.`
}

func (t *OnBuildComplete) Icon() string {
	return "gcp"
}

func (t *OnBuildComplete) Color() string {
	return "gray"
}

func (t *OnBuildComplete) ExampleData() map[string]any {
	return map[string]any{
		"id":             "12345678-abcd-1234-5678-abcdef012345",
		"projectId":      "my-project",
		"status":         "SUCCESS",
		"buildTriggerId": "abcdefgh-1234-5678-abcd-123456789012",
		"logUrl":         "https://console.cloud.google.com/cloud-build/builds/12345678-abcd-1234-5678-abcdef012345",
		"createTime":     "2025-01-01T00:00:00Z",
		"finishTime":     "2025-01-01T00:05:00Z",
	}
}

func (t *OnBuildComplete) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by build status. Leave empty to receive events for all statuses.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Success", Value: "SUCCESS"},
						{Label: "Failure", Value: "FAILURE"},
						{Label: "Cancelled", Value: "CANCELLED"},
						{Label: "Timeout", Value: "TIMEOUT"},
					},
				},
			},
		},
		{
			Name:        "triggerId",
			Label:       "Cloud Build Trigger",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only receive events from a specific Cloud Build trigger. Leave empty to receive events from all triggers.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTrigger,
				},
			},
		},
	}
}

func (t *OnBuildComplete) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	var metadata OnBuildCompleteMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != "" {
		return nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{"type": SubscriptionType})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnBuildCompleteMetadata{
		SubscriptionID: subscriptionID.String(),
	})
}

func (t *OnBuildComplete) Actions() []core.Action {
	return nil
}

func (t *OnBuildComplete) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnBuildComplete) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var config OnBuildCompleteConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var build struct {
		Status         string `mapstructure:"status"`
		BuildTriggerID string `mapstructure:"buildTriggerId"`
	}
	if err := mapstructure.Decode(ctx.Message, &build); err != nil {
		return fmt.Errorf("failed to decode build message: %w", err)
	}

	if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, build.Status) {
		ctx.Logger.Infof("gcp cloud build: status %q does not match configured statuses, skipping", build.Status)
		return nil
	}

	if config.TriggerID != "" && config.TriggerID != build.BuildTriggerID {
		ctx.Logger.Infof("gcp cloud build: trigger ID %q does not match configured trigger ID, skipping", build.BuildTriggerID)
		return nil
	}

	return ctx.Events.Emit(EmittedEventType, ctx.Message)
}

func (t *OnBuildComplete) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnBuildComplete) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}
