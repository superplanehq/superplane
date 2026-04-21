package compute

import (
	"encoding/json"
	"fmt"
	"github.com/superplanehq/superplane/pkg/configuration"
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
	return "Emits when an OCI Compute instance changes its lifecycle state"
}

func (t *OnInstanceStateChange) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnInstanceStateChange) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.Subscribe(t.Name(), nil)
}

func (t *OnInstanceStateChange) OnIntegrationMessage(ctx core.TriggerContext, message []byte) error {
	var event struct {
		EventType string `json:"eventType"`
		Data      struct {
			ResourceID string `json:"resourceId"`
			State      string `json:"state"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		return nil
	}

	if event.EventType != "com.oraclecloud.computeapi.instance.statechange" {
		return nil
	}

	return ctx.Emit("changed", "oci.instance.state", []any{event.Data})
}

func (t *OnInstanceStateChange) Cleanup(ctx core.TriggerContext) error {
	return nil
}
