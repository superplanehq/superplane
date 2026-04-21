package compute

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceStateChange struct{}

func (t *OnInstanceStateChange) Name() string {
	return "onInstanceStateChange"
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
	return "This trigger executes whenever a 'com.oraclecloud.computeapi.instance.statechange' event is received from OCI."
}

func (t *OnInstanceStateChange) ExampleData() map[string]any {
	return map[string]any{
		"type": "state_changed",
		"data": map[string]any{
			"id":    "ocid1.instance.oc1...",
			"state": "STOPPED",
		},
		"timestamp": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}
}

func (t *OnInstanceStateChange) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		EventType string `mapstructure:"eventType"`
		Data      struct {
			ResourceID string `mapstructure:"resourceId"`
			State      string `mapstructure:"state"`
		} `mapstructure:"data"`
	}

	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode OCI event: %w", err)
	}

	if event.EventType != "com.oraclecloud.computeapi.instance.statechange" {
		return nil
	}

	output := map[string]any{
		"id":    event.Data.ResourceID,
		"state": event.Data.State,
	}

	return ctx.Events.Emit("state_changed", output)
}

func (t *OnInstanceStateChange) Setup(ctx core.TriggerContext) error {
	_, err := ctx.Integration.Subscribe("onInstanceStateChange")
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
