package oci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceStateChange struct{}

const OnInstanceStateChangePayloadType = "oci.onInstanceStateChange"

var ociInstanceStateChangeEventTypes = map[string]struct{}{
	"com.oraclecloud.computeapi.startinstance.end":     {},
	"com.oraclecloud.computeapi.stopinstance.end":      {},
	"com.oraclecloud.computeapi.terminateinstance.end": {},
	"com.oraclecloud.computeapi.resetinstance.end":     {},
	"com.oraclecloud.computeapi.softstopinstance.end":  {},
	"com.oraclecloud.computeapi.softresetinstance.end": {},
}

type OnInstanceStateChangeConfiguration struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
}

type OnInstanceStateChangeMetadata struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	EventsRuleID  string `json:"eventsRuleId" mapstructure:"eventsRuleId"`
}

func (t *OnInstanceStateChange) Name() string {
	return "oci.onInstanceStateChange"
}

func (t *OnInstanceStateChange) Label() string {
	return "On Instance State Change"
}

func (t *OnInstanceStateChange) Description() string {
	return "Fires when an OCI Compute instance starts, stops, resets, or terminates"
}

func (t *OnInstanceStateChange) Documentation() string {
	return `The On Instance State Change trigger starts a workflow execution whenever an Oracle Cloud Infrastructure Compute instance completes a power or termination operation.

## How It Works

When this trigger is added to a workflow, SuperPlane creates an **OCI Events rule** in the configured compartment that forwards Compute instance lifecycle events to the integration's shared OCI Notifications topic.

## Configuration

- **Compartment**: The compartment to monitor for Compute instance state changes.

## Event Data

Each event payload includes:
- ` + "`eventType`" + ` — the OCI Compute API event type, such as ` + "`com.oraclecloud.computeapi.stopinstance.end`" + `
- ` + "`data.resourceId`" + ` — the instance OCID
- ` + "`data.resourceName`" + ` — the instance display name
- ` + "`data.compartmentId`" + ` — the compartment OCID
- ` + "`data.availabilityDomain`" + ` — the availability domain
- ` + "`eventTime`" + ` — ISO-8601 timestamp of the event
`
}

func (t *OnInstanceStateChange) Icon() string {
	return "oci"
}

func (t *OnInstanceStateChange) Color() string {
	return "red"
}

func (t *OnInstanceStateChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment to filter instance state-change events from",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
	}
}

func (t *OnInstanceStateChange) ExampleData() map[string]any {
	return exampleDataOnInstanceStateChange()
}

func (t *OnInstanceStateChange) Setup(ctx core.TriggerContext) error {
	var config OnInstanceStateChangeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	if config.CompartmentID == "" {
		return fmt.Errorf("compartmentId is required")
	}

	var integrationMetadata IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if integrationMetadata.TopicID == "" {
		return fmt.Errorf("integration topic not ready yet; ensure the OCI integration has been fully set up")
	}

	var metadata OnInstanceStateChangeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	if metadata.CompartmentID == config.CompartmentID && metadata.EventsRuleID != "" {
		return ctx.Integration.RequestWebhook(WebhookConfiguration{
			CompartmentID: config.CompartmentID,
			TopicID:       integrationMetadata.TopicID,
		})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if metadata.EventsRuleID != "" && metadata.CompartmentID != config.CompartmentID {
		if err := client.DeleteEventsRule(metadata.EventsRuleID); err != nil {
			ctx.Logger.Warnf("failed to delete old Events rule %q: %v", metadata.EventsRuleID, err)
		}
	}

	condition := `{"eventType": ["com.oraclecloud.computeapi.startinstance.end","com.oraclecloud.computeapi.stopinstance.end","com.oraclecloud.computeapi.terminateinstance.end","com.oraclecloud.computeapi.resetinstance.end","com.oraclecloud.computeapi.softstopinstance.end","com.oraclecloud.computeapi.softresetinstance.end"]}`

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to set up webhook URL: %w", err)
	}
	ruleName := fmt.Sprintf("superplane-instance-state-change-%s", path.Base(webhookURL))
	rule, err := client.CreateEventsRule(
		config.CompartmentID,
		ruleName,
		condition,
		integrationMetadata.TopicID,
	)
	if err != nil {
		return fmt.Errorf("failed to create Events rule: %w", err)
	}

	if err := ctx.Metadata.Set(OnInstanceStateChangeMetadata{
		CompartmentID: config.CompartmentID,
		EventsRuleID:  rule.ID,
	}); err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		CompartmentID: config.CompartmentID,
		TopicID:       integrationMetadata.TopicID,
	})
}

func (t *OnInstanceStateChange) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnInstanceStateChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnInstanceStateChange) Cleanup(ctx core.TriggerContext) error {
	var metadata OnInstanceStateChangeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		ctx.Logger.Warnf("failed to decode trigger metadata during cleanup: %v", err)
		return nil
	}

	if metadata.EventsRuleID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client during cleanup: %w", err)
	}

	if err := client.DeleteEventsRule(metadata.EventsRuleID); err != nil {
		ctx.Logger.Warnf("failed to delete Events rule %q during cleanup: %v", metadata.EventsRuleID, err)
	}

	return nil
}

func (t *OnInstanceStateChange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	cfg := OnInstanceStateChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(ctx.Body, &envelope); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	if confirmURL := ctx.Headers.Get("X-OCI-NS-ConfirmationURL"); confirmURL != "" {
		if err := validateONSConfirmationURL(confirmURL); err != nil {
			return http.StatusBadRequest, nil, fmt.Errorf("refusing ONS confirmation URL: %w", err)
		}
		req, err := http.NewRequest(http.MethodGet, confirmURL, nil)
		if err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to build ONS confirmation request: %w", err)
		}
		resp, err := ctx.HTTP.Do(req)
		if err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to confirm ONS subscription: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return http.StatusInternalServerError, nil, fmt.Errorf("ONS confirmation returned %d", resp.StatusCode)
		}
		return http.StatusOK, nil, nil
	}

	eventType, _ := envelope["eventType"].(string)
	if _, ok := ociInstanceStateChangeEventTypes[eventType]; !ok {
		return http.StatusOK, nil, nil
	}

	if cfg.CompartmentID != "" {
		data, _ := envelope["data"].(map[string]any)
		compartmentID, _ := data["compartmentId"].(string)
		if compartmentID != cfg.CompartmentID {
			return http.StatusOK, nil, nil
		}
	}

	if err := ctx.Events.Emit(OnInstanceStateChangePayloadType, envelope); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil, nil
}
