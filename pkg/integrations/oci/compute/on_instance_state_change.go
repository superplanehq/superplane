package compute

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

var validEvents = map[string]bool{
	"com.oraclecloud.computeapi.launchinstance.end":    true,
	"com.oraclecloud.computeapi.terminateinstance.end": true,
	"com.oraclecloud.computeapi.updateinstance.end":    true,
	"com.oraclecloud.computeapi.instanceaction.end":    true,
}

type OnInstanceStateChange struct{}

func (t *OnInstanceStateChange) Name() string {
	return "oci.onInstanceStateChange"
}

func (t *OnInstanceStateChange) Label() string {
	return "On Instance State Change"
}

func (t *OnInstanceStateChange) Description() string {
	return "Triggered when a Compute instance changes its state in OCI"
}

func (t *OnInstanceStateChange) Icon() string {
	return "oci"
}

func (t *OnInstanceStateChange) Color() string {
	return "#f30000"
}

func (t *OnInstanceStateChange) Documentation() string {
	return "This trigger executes whenever an instance lifecycle event (launch, terminate, update, or action) is received from OCI."
}

func (t *OnInstanceStateChange) ExampleData() map[string]any {
	return map[string]any{
		"type": "state_changed",
		"data": map[string]any{
			"id":    "ocid1.instance.oc1...",
			"state": "STOPPED",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (t *OnInstanceStateChange) Configuration() []configuration.Field {
	return nil
}

func (t *OnInstanceStateChange) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		EventType string `mapstructure:"eventType"`
		Data      struct {
			ResourceID        string         `mapstructure:"resourceId"`
			AdditionalDetails map[string]any `mapstructure:"additionalDetails"`
		} `mapstructure:"data"`
	}

	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode OCI event: %w", err)
	}

	if !validEvents[event.EventType] {
		return nil
	}

	state := ""
	if val, ok := event.Data.AdditionalDetails["state"]; ok {
		state, _ = val.(string)
	} else if val, ok := event.Data.AdditionalDetails["lifecycleState"]; ok {
		state, _ = val.(string)
	}

	output := map[string]any{
		"id":    event.Data.ResourceID,
		"state": state,
	}

	return ctx.Events.Emit("state_changed", output)
}

func (t *OnInstanceStateChange) Setup(ctx core.TriggerContext) error {
	_, err := ctx.Integration.Subscribe("oci.onInstanceStateChange")
	return err
}

func (t *OnInstanceStateChange) Actions() []core.Action {
	return nil
}

func (t *OnInstanceStateChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnInstanceStateChange) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnInstanceStateChange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
