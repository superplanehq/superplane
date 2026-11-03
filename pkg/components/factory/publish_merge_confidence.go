package factory

import (
	"fmt"
	"maps"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const PublishMergeConfidenceComponentName = "publishMergeConfidence"
const PublishMergeConfidenceHook = "publishMergeConfidence"
const MergeConfidencePublishedPayloadType = "mergeConfidence.published"

type PublishMergeConfidence struct{}
type PublishMergeConfidenceConfiguration struct {
	SourceRunID string `json:"sourceRunId" mapstructure:"sourceRunId"`
}

func init() {
	registry.RegisterAction(PublishMergeConfidenceComponentName, &PublishMergeConfidence{})
}

func (c *PublishMergeConfidence) Name() string { return PublishMergeConfidenceComponentName }

func (c *PublishMergeConfidence) Label() string { return "Publish Merge Confidence" }

func (c *PublishMergeConfidence) Description() string {
	return "Publish a completed assessment in the pull request description"
}

func (c *PublishMergeConfidence) Documentation() string {
	return "Publish the saved checks for a completed assessment run in this factory. Set sourceRunId from an On Run Finished event. Publication preserves the existing description. Transient host failures retry independently. Execution details show attempts and skipped outcomes. Newer assessments and changed commits stop publication."
}

func (c *PublishMergeConfidence) Icon() string { return "file-check" }

func (c *PublishMergeConfidence) Color() string { return "blue" }

func (c *PublishMergeConfidence) Configuration() []configuration.Field {
	return []configuration.Field{{Name: "sourceRunId", Label: "Source run ID", Type: configuration.FieldTypeString, Required: true,
		Description: "The completed assessment run in this factory."}}
}

func (c *PublishMergeConfidence) ExampleOutput() map[string]any {
	return map[string]any{"type": MergeConfidencePublishedPayloadType, "timestamp": "2026-11-03T12:00:00Z", "data": map[string]any{"sourceRunId": "123", "status": "published", "attempts": 1}}
}

func (c *PublishMergeConfidence) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishMergeConfidence) Setup(core.SetupContext) error { return nil }

func (c *PublishMergeConfidence) Execute(ctx core.ExecutionContext) error {
	var config PublishMergeConfidenceConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if _, err := uuid.Parse(config.SourceRunID); err != nil {
		return fmt.Errorf("source run ID must be a valid run ID")
	}
	metadata, _ := ctx.Metadata.Get().(map[string]any)
	metadata = maps.Clone(metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["sourceRunId"], metadata["status"], metadata["attempts"] = config.SourceRunID, "queued", 0
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}
	// The request worker calls the publisher after this execution transaction
	// commits, so host calls do not hold an execution lock or connection.
	return ctx.Requests.ScheduleActionCall(PublishMergeConfidenceHook, map[string]any{"baseURL": ctx.BaseURL}, time.Second)
}

func (c *PublishMergeConfidence) Hooks() []core.Hook {
	return []core.Hook{{Name: PublishMergeConfidenceHook, Type: core.HookTypeInternal}}
}

func (c *PublishMergeConfidence) HandleHook(core.ActionHookContext) error {
	return fmt.Errorf("publication requires the scheduled request dispatcher")
}

func (c *PublishMergeConfidence) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *PublishMergeConfidence) Cancel(core.ExecutionContext) error { return nil }

func (c *PublishMergeConfidence) Cleanup(core.SetupContext) error { return nil }
