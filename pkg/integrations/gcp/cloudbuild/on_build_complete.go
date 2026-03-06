package cloudbuild

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EmittedEventType = "gcp.cloudbuild.build"
	SubscriptionType = "cloudbuild"

	buildSourceTriggered = "triggered"
	buildSourceDirect    = "direct"
)

type OnBuildComplete struct{}

type OnBuildCompleteConfiguration struct {
	Statuses    []string `json:"statuses" mapstructure:"statuses"`
	TriggerID   string   `json:"triggerId" mapstructure:"triggerId"`
	BuildSource string   `json:"buildSource" mapstructure:"buildSource"`
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

**Required GCP setup:** Ensure the **Cloud Build API** and **Pub/Sub API** are enabled in your project. The service account used by the integration must have ` + "`roles/pubsub.admin`" + ` so SuperPlane can automatically create the ` + "`cloud-builds`" + ` topic and its push subscription.

## Configuration

- **Statuses**: Filter by terminal Cloud Build status.
- **Build Source**: Optionally limit events to trigger-based builds or direct/API builds. Leave empty to listen to both.
- **Cloud Build Trigger**: Filter to a specific Cloud Build trigger. This only applies to trigger-based builds and cannot be combined with **Build Source = Direct/API Builds**.

## Event Data

Each event contains the full Cloud Build resource, including ` + "`id`" + `, ` + "`status`" + ` (SUCCESS, FAILURE, INTERNAL_ERROR, TIMEOUT, CANCELLED, EXPIRED), ` + "`buildTriggerId`" + `, ` + "`logUrl`" + `, ` + "`createTime`" + `, ` + "`finishTime`" + `, and more.`
}

func (t *OnBuildComplete) Icon() string {
	return "gcp"
}

func (t *OnBuildComplete) Color() string {
	return "gray"
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
			Name:        "buildSource",
			Label:       "Build Source",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optionally listen only to trigger-based builds or only to direct/API builds. Leave empty to receive both.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Trigger-Based Builds", Value: buildSourceTriggered},
						{Label: "Direct/API Builds", Value: buildSourceDirect},
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
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "buildSource", Values: []string{"", buildSourceTriggered}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTrigger,
				},
			},
		},
	}
}

func (t *OnBuildComplete) Setup(ctx core.TriggerContext) error {
	if _, err := decodeOnBuildCompleteConfiguration(ctx.Configuration); err != nil {
		return err
	}

	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	if err := scheduleCloudBuildSetupIfNeeded(ctx.Integration); err != nil {
		return err
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
	config, err := decodeOnBuildCompleteConfiguration(ctx.Configuration)
	if err != nil {
		return err
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

	if len(config.Statuses) == 0 && !isTerminalBuildStatus(build.Status) {
		ctx.Logger.Infof("gcp cloud build: status %q is not terminal, skipping", build.Status)
		return nil
	}

	if !matchesBuildSource(config.BuildSource, build.BuildTriggerID) {
		ctx.Logger.Infof("gcp cloud build: build source %q does not match configured build source filter %q, skipping", build.BuildTriggerID, config.BuildSource)
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

func decodeOnBuildCompleteConfiguration(raw any) (OnBuildCompleteConfiguration, error) {
	config := OnBuildCompleteConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return OnBuildCompleteConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.TriggerID = strings.TrimSpace(config.TriggerID)
	config.BuildSource = strings.ToLower(strings.TrimSpace(config.BuildSource))

	switch config.BuildSource {
	case "", buildSourceTriggered, buildSourceDirect:
	default:
		return OnBuildCompleteConfiguration{}, fmt.Errorf("buildSource must be one of %q or %q", buildSourceTriggered, buildSourceDirect)
	}

	if config.BuildSource == buildSourceDirect && config.TriggerID != "" {
		return OnBuildCompleteConfiguration{}, fmt.Errorf("triggerId cannot be combined with buildSource=%s", buildSourceDirect)
	}

	return config, nil
}

func matchesBuildSource(buildSource string, buildTriggerID string) bool {
	hasTriggerID := buildTriggerID != ""

	switch buildSource {
	case buildSourceTriggered:
		return hasTriggerID
	case buildSourceDirect:
		return !hasTriggerID
	default:
		return true
	}
}
