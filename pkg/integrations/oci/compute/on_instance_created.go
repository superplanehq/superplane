package compute

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceCreated struct{}

func (t *OnInstanceCreated) Name() string {
	return "onInstanceCreated"
}

func (t *OnInstanceCreated) Label() string {
	return "On Instance Created"
}

func (t *OnInstanceCreated) Description() string {
	return "Triggered when a new Compute instance is created in OCI"
}

func (t *OnInstanceCreated) Icon() string {
	return "oci"
}

func (t *OnInstanceCreated) Color() string {
	return "#f30000"
}

func (t *OnInstanceCreated) Documentation() string {
	return "This trigger executes whenever a 'com.oraclecloud.computeapi.launchinstance.end' event is received from OCI."
}

func (t *OnInstanceCreated) ExampleData() map[string]any {
	return map[string]any{
		"type": "created",
		"data": map[string]any{
			"id":          "ocid1.instance.oc1...",
			"displayName": "my-instance",
			"state":       "RUNNING",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (t *OnInstanceCreated) Configuration() []configuration.Field {
	return nil
}

func (t *OnInstanceCreated) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		EventType string `mapstructure:"eventType"`
		Data      struct {
			ResourceID   string `mapstructure:"resourceId"`
			ResourceName string `mapstructure:"resourceName"`
		} `mapstructure:"data"`
	}

	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode OCI event: %w", err)
	}

	if event.EventType != "com.oraclecloud.computeapi.launchinstance.end" {
		return nil
	}

	output := map[string]any{
		"id":          event.Data.ResourceID,
		"displayName": event.Data.ResourceName,
		"state":       "RUNNING",
	}

	return ctx.Events.Emit("created", output)
}

func (t *OnInstanceCreated) Setup(ctx core.TriggerContext) error {
	_, err := ctx.Integration.Subscribe("onInstanceCreated")
	return err
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

func (t *OnInstanceCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 404, nil, nil
}
