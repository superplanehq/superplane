package compute

import (
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceStateChange struct{}

func (t *OnInstanceStateChange) Name() string        { return "onInstanceStateChange" }
func (t *OnInstanceStateChange) Label() string       { return "On Instance State Change" }
func (t *OnInstanceStateChange) Description() string { return "Emits when an OCI Compute instance state changes" }
func (t *OnInstanceStateChange) Icon() string        { return "oci" }
func (t *OnInstanceStateChange) Color() string       { return "#f30000" }
func (t *OnInstanceStateChange) Documentation() string {
	return "Triggered when the state of an OCI Compute instance changes (e.g., STARTING, STOPPED)."
}

func (t *OnInstanceStateChange) ExampleData() map[string]any {
	return map[string]any{
		"resourceName": "instance-20260421",
		"resourceId":   "ocid1.instance.oc1...",
		"state":        "STOPPED",
	}
}

func (t *OnInstanceStateChange) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnInstanceStateChange) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.Subscribe(t.Name())
}

func (t *OnInstanceStateChange) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		EventType string `mapstructure:"eventType"`
		Data      struct {
			ResourceName string `mapstructure:"resourceName"`
			ResourceID   string `mapstructure:"resourceId"`
			State        string `mapstructure:"state"`
		} `mapstructure:"data"`
	}

	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return nil
	}

	if event.EventType != "com.oraclecloud.computeapi.instance.statechange" {
		return nil
	}

	return ctx.Events.Emit("state_changed", event.Data)
}

func (t *OnInstanceStateChange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
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
