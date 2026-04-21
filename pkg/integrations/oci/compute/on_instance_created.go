package compute

import (
	"encoding/json"
	"fmt"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceCreated struct{}

func (t *OnInstanceCreated) Name() string {
	return "oci.compute.onInstanceCreated"
}

func (t *OnInstanceCreated) Label() string {
	return "Compute • On Instance Created"
}

func (t *OnInstanceCreated) Description() string {
	return "Triggered when a new OCI Compute instance is created"
}

func (t *OnInstanceCreated) Icon() string {
	return "oci"
}

func (t *OnInstanceCreated) Configuration() []configuration.Field {
	return nil
}

func (t *OnInstanceCreated) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnInstanceCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var payload map[string]any
	if err := json.NewDecoder(ctx.Request.Body).Decode(&payload); err != nil {
		return 400, nil, err
	}

	eventType, _ := payload["eventType"].(string)
	if eventType != "com.oraclecloud.computeapi.launchinstance.end" {
		return 200, nil, nil
	}

	err := ctx.Events.Emit("oci.compute.instanceCreated", payload)
	if err != nil {
		return 500, nil, err
	}

	return 200, &core.WebhookResponseBody{Data: map[string]any{"status": "accepted"}}, nil
}
