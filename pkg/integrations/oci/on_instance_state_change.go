package oci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnInstanceStateChange struct{}

const OnInstanceStateChangePayloadType = "oci.onInstanceStateChange"

var ociInstanceStateChangeEventTypes = map[string]struct{}{
	"com.oraclecloud.computeapi.instanceaction.end":    {},
	"com.oraclecloud.computeapi.terminateinstance.end": {},
}

var ociInstanceStateChangeActionTypes = map[string]struct{}{
	"start":     {},
	"stop":      {},
	"reset":     {},
	"softstop":  {},
	"softreset": {},
}

const (
	ociInstanceStateChangeStart     = "start"
	ociInstanceStateChangeStop      = "stop"
	ociInstanceStateChangeReset     = "reset"
	ociInstanceStateChangeSoftStop  = "softstop"
	ociInstanceStateChangeSoftReset = "softreset"
	ociInstanceStateChangeTerminate = "terminate"
)

var ociInstanceStateChangeOptions = []configuration.FieldOption{
	{Label: "Start", Value: ociInstanceStateChangeStart},
	{Label: "Stop", Value: ociInstanceStateChangeStop},
	{Label: "Reset", Value: ociInstanceStateChangeReset},
	{Label: "Soft Stop", Value: ociInstanceStateChangeSoftStop},
	{Label: "Soft Reset", Value: ociInstanceStateChangeSoftReset},
	{Label: "Terminate", Value: ociInstanceStateChangeTerminate},
}

type OnInstanceStateChangeConfiguration struct {
	CompartmentID string   `json:"compartmentId" mapstructure:"compartmentId"`
	StateChanges  []string `json:"stateChanges" mapstructure:"stateChanges"`
}

type OnInstanceStateChangeMetadata struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	EventsRuleID  string `json:"eventsRuleId" mapstructure:"eventsRuleId"`
	Condition     string `json:"condition" mapstructure:"condition"`
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
- **State Changes**: Optional list of state changes to emit. Leave empty to emit all supported state changes.

## Event Data

Each event payload includes:
- ` + "`eventType`" + ` — the OCI Compute API event type, such as ` + "`com.oraclecloud.computeapi.instanceaction.end`" + `
- ` + "`data.resourceId`" + ` — the instance OCID
- ` + "`data.resourceName`" + ` — the instance display name
- ` + "`data.compartmentId`" + ` — the compartment OCID
- ` + "`data.availabilityDomain`" + ` — the availability domain
- ` + "`data.additionalDetails.instanceActionType`" + ` — the completed power action, such as ` + "`start`" + `, ` + "`stop`" + `, or ` + "`softstop`" + `
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
		{
			Name:     "stateChanges",
			Label:    "State Changes",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default: []string{
				ociInstanceStateChangeStart,
				ociInstanceStateChangeStop,
				ociInstanceStateChangeReset,
				ociInstanceStateChangeSoftStop,
				ociInstanceStateChangeSoftReset,
				ociInstanceStateChangeTerminate,
			},
			Description: "Only emit events for these state changes. Leave empty to emit all supported state changes.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: ociInstanceStateChangeOptions,
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
	if err := validateStateChanges(config.StateChanges); err != nil {
		return err
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

	condition := `{"eventType": ["com.oraclecloud.computeapi.instanceaction.end","com.oraclecloud.computeapi.terminateinstance.end"]}`

	if metadata.CompartmentID == config.CompartmentID && metadata.EventsRuleID != "" && metadata.Condition == condition {
		return ctx.Integration.RequestWebhook(WebhookConfiguration{
			CompartmentID: config.CompartmentID,
			TopicID:       integrationMetadata.TopicID,
		})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if metadata.EventsRuleID != "" {
		if err := client.DeleteEventsRule(metadata.EventsRuleID); err != nil {
			ctx.Logger.Warnf("failed to delete old Events rule %q: %v", metadata.EventsRuleID, err)
		}
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to set up webhook URL: %w", err)
	}
	webhookID, err := extractWebhookID(webhookURL)
	if err != nil {
		return err
	}
	ruleName := fmt.Sprintf("superplane-instance-state-change-%s", webhookID)
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
		Condition:     condition,
	}); err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		CompartmentID: config.CompartmentID,
		TopicID:       integrationMetadata.TopicID,
	})
}

func (t *OnInstanceStateChange) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnInstanceStateChange) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown hook: %s", ctx.Name)
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

	if confirmURL := ctx.Headers.Get("X-OCI-NS-ConfirmationURL"); confirmURL != "" {
		return confirmONSSubscription(ctx, confirmURL)
	}

	var envelope map[string]any
	if err := json.Unmarshal(ctx.Body, &envelope); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	eventType, _ := envelope["eventType"].(string)
	if _, ok := ociInstanceStateChangeEventTypes[eventType]; !ok {
		return http.StatusOK, nil, nil
	}

	data, _ := envelope["data"].(map[string]any)
	if eventType == "com.oraclecloud.computeapi.instanceaction.end" {
		additionalDetails, _ := data["additionalDetails"].(map[string]any)
		actionType, _ := additionalDetails["instanceActionType"].(string)
		if _, ok := ociInstanceStateChangeActionTypes[actionType]; !ok {
			return http.StatusOK, nil, nil
		}
		if len(cfg.StateChanges) > 0 && !slices.Contains(cfg.StateChanges, actionType) {
			return http.StatusOK, nil, nil
		}
	}

	if eventType == "com.oraclecloud.computeapi.terminateinstance.end" {
		if len(cfg.StateChanges) > 0 && !slices.Contains(cfg.StateChanges, ociInstanceStateChangeTerminate) {
			return http.StatusOK, nil, nil
		}
	}

	if cfg.CompartmentID != "" {
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

func validateStateChanges(stateChanges []string) error {
	for _, stateChange := range stateChanges {
		if !slices.Contains([]string{
			ociInstanceStateChangeStart,
			ociInstanceStateChangeStop,
			ociInstanceStateChangeReset,
			ociInstanceStateChangeSoftStop,
			ociInstanceStateChangeSoftReset,
			ociInstanceStateChangeTerminate,
		}, stateChange) {
			return fmt.Errorf("unsupported state change: %s", stateChange)
		}
	}
	return nil
}
