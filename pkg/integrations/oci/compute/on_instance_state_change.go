package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceStateChange struct{}

type OnInstanceStateChangeMetadata struct {
	RuleID string `json:"ruleId" mapstructure:"ruleId"`
}

func (t *OnInstanceStateChange) Name() string {
	return "oci.compute.onInstanceStateChange"
}

func (t *OnInstanceStateChange) Label() string {
	return "Compute • On Instance State Change"
}

func (t *OnInstanceStateChange) Description() string {
	return "Triggered when an OCI Compute instance state changes"
}

func (t *OnInstanceStateChange) Icon() string {
	return "oci"
}

func (t *OnInstanceStateChange) Configuration() []configuration.Field {
	return nil
}

func (t *OnInstanceStateChange) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnInstanceStateChange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var payload map[string]any
	if err := json.NewDecoder(ctx.Request.Body).Decode(&payload); err != nil {
		return 400, nil, err
	}

	eventType, _ := payload["eventType"].(string)
	if eventType == "" {
		return 400, nil, fmt.Errorf("missing eventType in OCI payload")
	}

	err := ctx.Events.Emit("oci.compute.instanceEvent", payload)
	if err != nil {
		return 500, nil, err
	}

	return 200, &core.WebhookResponseBody{Data: map[string]any{"status": "accepted"}}, nil
}
