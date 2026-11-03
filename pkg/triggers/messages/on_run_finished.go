package messages

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const OnRunFinishedTriggerName = "onRunFinished"
const RunFinishedPayloadType = "run.finished"

type OnRunFinished struct{}

type OnRunFinishedConfiguration struct {
	SourceEntryNodeID string `json:"sourceEntryNodeId" mapstructure:"sourceEntryNodeId"`
}

func init() { registry.RegisterTrigger(OnRunFinishedTriggerName, &OnRunFinished{}) }

func (t *OnRunFinished) Name() string { return OnRunFinishedTriggerName }

func (t *OnRunFinished) Label() string { return "On Run Finished" }

func (t *OnRunFinished) Description() string {
	return "Start when a run from the selected entry node finishes in this canvas"
}

func (t *OnRunFinished) Documentation() string {
	return "Start a separate run when a run in this canvas finishes. Select its source entry-node ID. Passed, failed, and cancelled runs emit events, including runs that fail before execution starts. The event contains runId, result, and entryNodeId. This trigger does not react to its own runs."
}

func (t *OnRunFinished) Color() string { return "gray" }

func (t *OnRunFinished) Icon() string { return "flag" }

func (t *OnRunFinished) Configuration() []configuration.Field {
	return []configuration.Field{{Name: "sourceEntryNodeId", Label: "Source entry node ID", Type: configuration.FieldTypeString, Required: true,
		Description: "Only receive runs that start at this node in the same canvas.",
		TypeOptions: &configuration.TypeOptions{String: &configuration.StringTypeOptions{AllowExpressions: new(bool)}}}}
}

func (t *OnRunFinished) ExampleData() map[string]any {
	return map[string]any{"type": RunFinishedPayloadType, "timestamp": "2026-11-03T12:00:00Z", "data": map[string]any{"runId": "123", "result": "passed", "entryNodeId": "assessment"}}
}

func (t *OnRunFinished) Setup(ctx core.TriggerContext) error {
	var config OnRunFinishedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if strings.TrimSpace(config.SourceEntryNodeID) == "" {
		return fmt.Errorf("source entry node ID is required")
	}
	return nil
}

func (t *OnRunFinished) Hooks() []core.Hook { return nil }

func (t *OnRunFinished) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, fmt.Errorf("hook %s is not supported", ctx.Name)
}

func (t *OnRunFinished) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (t *OnRunFinished) Cleanup(ctx core.TriggerContext) error { return nil }
