package compute

import (
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceCreated struct{}

func (t *OnInstanceCreated) Name() string        { return "onInstanceCreated" }
func (t *OnInstanceCreated) Label() string       { return "On Instance Created" }
func (t *OnInstanceCreated) Description() string { return "Emits when a new OCI Compute instance reaches RUNNING state" }
func (t *OnInstanceCreated) Icon() string        { return "oci" }
func (t *OnInstanceCreated) Color() string       { return "#f30000" }
func (t *OnInstanceCreated) Documentation() string {
	return "Triggered when a new OCI Compute instance reaches the RUNNING state."
}

func (t *OnInstanceCreated) ExampleData() map[string]any {
	return map[string]any{
		"resourceName": "instance-20260421",
		"resourceId":   "ocid1.instance.oc1...",
	}
}

func (t *OnInstanceCreated) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnInstanceCreated) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.Subscribe(t.Name())
}

func (t *OnInstanceCreated) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		EventType string `mapstructure:"eventType"`
		Data      struct {
			ResourceName string `mapstructure:"resourceName"`
			ResourceID   string `mapstructure:"resourceId"`
		} `mapstructure:"data"`
	}

	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return nil
	}

	if event.EventType != "com.oraclecloud.computeapi.launchinstance.end" {
		return nil
	}

	return ctx.Events.Emit("created", event.Data)
}

func (t *OnInstanceCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (t *OnInstanceCreated) Actions() []core.Action {
	return nil
}

func (t *OnInstanceCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnInstanceCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}
