package compute

import (
	"encoding/json"
	"fmt"
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
	return "Emits when a new OCI Compute instance reaches RUNNING state"
}

func (t *OnInstanceCreated) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnInstanceCreated) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.Subscribe(t.Name(), nil)
}

func (t *OnInstanceCreated) OnIntegrationMessage(ctx core.TriggerContext, message []byte) error {
	var event struct {
		EventType string `json:"eventType"`
		Data      struct {
			ResourceName string `json:"resourceName"`
			ResourceID   string `json:"resourceId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		return nil // Not our event or malformed
	}

	if event.EventType != "com.oraclecloud.computeapi.launchinstance.end" {
		return nil
	}

	return ctx.Emit("created", "oci.instance", []any{event.Data})
}

func (t *OnInstanceCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}
