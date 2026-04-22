package oci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// OnComputeInstanceCreated is a trigger that fires when an OCI Compute instance
// reaches RUNNING state. It receives events via an OCI Events Service webhook
// (via OCI Notifications HTTPS subscription) and emits when the event type is
// com.oraclecloud.computeapi.launchinstance.end.
type OnComputeInstanceCreated struct{}

const (
	ociEventTypeComputeLaunchEnd = "com.oraclecloud.computeapi.launchinstance.end"
)

type OnComputeInstanceCreatedConfiguration struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	SecretToken   string `json:"secretToken" mapstructure:"secretToken"`
}

func (t *OnComputeInstanceCreated) Name() string {
	return "oci.onComputeInstanceCreated"
}

func (t *OnComputeInstanceCreated) Label() string {
	return "On Compute Instance Created"
}

func (t *OnComputeInstanceCreated) Description() string {
	return "Fires when a new OCI Compute instance reaches RUNNING state (via OCI Events Service)"
}

func (t *OnComputeInstanceCreated) Documentation() string {
	return `The On Compute Instance Created trigger starts a workflow execution whenever an OCI Compute instance launch completes and the instance reaches **RUNNING** state.

## How It Works

Events are delivered through the OCI Events Service → OCI Notifications (ONS) → HTTPS subscription chain:

1. SuperPlane generates a unique webhook URL for this trigger.
2. In OCI, create a **Notifications topic** and add an **HTTPS subscription** pointing to the webhook URL.
3. Create an **Events rule** in the compartment you want to monitor:
   - **Conditions**: Event Type = ` + "`com.oraclecloud.computeapi.launchinstance.end`" + `
   - **Actions**: Send to the ONS topic created above.
4. Optionally set a **Secret Token** in both SuperPlane and the subscription to validate incoming requests.

## Configuration

- **Compartment OCID**: Filter events to a specific compartment. Leave blank to accept events from any compartment.
- **Secret Token**: Optional shared secret to validate that the webhook came from OCI Notifications.

## Event Data

Each event payload includes:
- ` + "`eventType`" + ` — ` + "`com.oraclecloud.computeapi.launchinstance.end`" + `
- ` + "`data.resourceId`" + ` — the instance OCID
- ` + "`data.resourceName`" + ` — the instance display name
- ` + "`data.compartmentId`" + ` — the compartment OCID
- ` + "`data.availabilityDomain`" + ` — the availability domain
- ` + "`eventTime`" + ` — ISO-8601 timestamp of the event
`
}

func (t *OnComputeInstanceCreated) Icon() string {
	return "oci"
}

func (t *OnComputeInstanceCreated) Color() string {
	return "red"
}

func (t *OnComputeInstanceCreated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment (Filter)",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only emit events from this compartment. Leave blank to accept events from any compartment.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
		{
			Name:        "secretToken",
			Label:       "Secret Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Optional shared secret to validate incoming webhook requests from OCI Notifications.",
		},
	}
}

func (t *OnComputeInstanceCreated) ExampleData() map[string]any {
	return exampleDataOnComputeInstanceCreated()
}

func (t *OnComputeInstanceCreated) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnComputeInstanceCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnComputeInstanceCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnComputeInstanceCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// HandleWebhook processes inbound requests from OCI Notifications.
// OCI Notifications sends a JSON payload with an eventType field.
// If the payload is an ONS subscription confirmation, it replies with 200 to confirm it.
func (t *OnComputeInstanceCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	cfg := OnComputeInstanceCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate the optional secret token via the Authorization header.
	if cfg.SecretToken != "" {
		authHeader := ctx.Headers.Get("Authorization")
		if !strings.EqualFold(authHeader, "Bearer "+cfg.SecretToken) {
			return http.StatusUnauthorized, nil, fmt.Errorf("invalid secret token")
		}
	}

	// OCI Notifications sends a JSON envelope. Parse it to extract the event type.
	var envelope map[string]any
	if err := json.Unmarshal(ctx.Body, &envelope); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	// Handle ONS subscription confirmation handshake — just acknowledge it.
	if confirmURL, ok := envelope["confirmationUrl"].(string); ok && confirmURL != "" {
		confirmBody, _ := json.Marshal(envelope)
		return http.StatusOK, &core.WebhookResponseBody{Body: confirmBody, ContentType: "application/json"}, nil
	}

	eventType, _ := envelope["eventType"].(string)
	if eventType != ociEventTypeComputeLaunchEnd {
		// Ignore non-launch events silently.
		return http.StatusOK, nil, nil
	}

	// Filter by compartmentId if configured.
	if cfg.CompartmentID != "" {
		data, _ := envelope["data"].(map[string]any)
		compartmentID, _ := data["compartmentId"].(string)
		if compartmentID != cfg.CompartmentID {
			return http.StatusOK, nil, nil
		}
	}

	return http.StatusOK, &core.WebhookResponseBody{Body: ctx.Body, ContentType: "application/json"}, nil
}
