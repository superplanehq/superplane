package oci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ociEventTypeFunctionInvoke  = "com.oraclecloud.functions.invokefunction"
	OnFunctionInvokePayloadType = "oci.onFunctionInvoke"
)

type OnFunctionInvoke struct{}

type OnFunctionInvokeConfiguration struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
}

type OnFunctionInvokeMetadata struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	EventsRuleID  string `json:"eventsRuleId" mapstructure:"eventsRuleId"`
}

func (t *OnFunctionInvoke) Name() string {
	return "oci.onFunctionInvoke"
}

func (t *OnFunctionInvoke) Label() string {
	return "On Function Invoked"
}

func (t *OnFunctionInvoke) Description() string {
	return "Fires when an OCI Function is invoked (via OCI Events / Audit trail)"
}

func (t *OnFunctionInvoke) Documentation() string {
	return `The On Function Invoked trigger starts a workflow execution whenever an OCI Function invocation is recorded by the OCI Audit trail.

## How It Works

When this trigger is configured, SuperPlane creates an **OCI Events rule** in the specified compartment that routes ` + "`com.oraclecloud.functions.invokefunction`" + ` audit events to the shared ONS topic. SuperPlane subscribes to that topic via HTTPS and fires the trigger for each matching event.

> **Note**: OCI Audit events may arrive with a short delay relative to the actual invocation.

## Configuration

- **Compartment**: The compartment to monitor for function invocations.

## Event Data

Each event payload includes:
- ` + "`eventType`" + ` — ` + "`com.oraclecloud.functions.invokefunction`" + `
- ` + "`data.resourceId`" + ` — the function OCID
- ` + "`data.resourceName`" + ` — the function display name
- ` + "`data.compartmentId`" + ` — the compartment OCID
- ` + "`eventTime`" + ` — ISO-8601 timestamp of the invocation audit event
`
}

func (t *OnFunctionInvoke) Icon() string {
	return "oci"
}

func (t *OnFunctionInvoke) Color() string {
	return "red"
}

func (t *OnFunctionInvoke) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment to monitor for function invocations",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
	}
}

func (t *OnFunctionInvoke) ExampleData() map[string]any {
	return exampleDataOnFunctionInvoke()
}

func (t *OnFunctionInvoke) Setup(ctx core.TriggerContext) error {
	config, integrationMetadata, metadata, err := decodeFunctionTriggerSetupInputs(ctx)
	if err != nil {
		return err
	}

	// Already set up for this compartment — just re-request the webhook subscription.
	if metadata.CompartmentID == config.CompartmentID && metadata.EventsRuleID != "" {
		return requestWebhook(ctx, config.CompartmentID, integrationMetadata.TopicID)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if err := cleanupOldFunctionRule(ctx, client, metadata, config.CompartmentID); err != nil {
		return err
	}

	ruleID, err := ensureFunctionEventsRule(ctx, client, config.CompartmentID, integrationMetadata.TopicID)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(OnFunctionInvokeMetadata{
		CompartmentID: config.CompartmentID,
		EventsRuleID:  ruleID,
	}); err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	return requestWebhook(ctx, config.CompartmentID, integrationMetadata.TopicID)
}

func decodeFunctionTriggerSetupInputs(ctx core.TriggerContext) (OnFunctionInvokeConfiguration, IntegrationMetadata, OnFunctionInvokeMetadata, error) {
	var config OnFunctionInvokeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return config, IntegrationMetadata{}, OnFunctionInvokeMetadata{}, fmt.Errorf("failed to decode trigger configuration: %w", err)
	}
	if config.CompartmentID == "" {
		return config, IntegrationMetadata{}, OnFunctionInvokeMetadata{}, fmt.Errorf("compartmentId is required")
	}

	var integrationMetadata IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return config, IntegrationMetadata{}, OnFunctionInvokeMetadata{}, fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if integrationMetadata.TopicID == "" {
		return config, IntegrationMetadata{}, OnFunctionInvokeMetadata{}, fmt.Errorf("integration topic not ready yet; ensure the OCI integration has been fully set up")
	}

	var metadata OnFunctionInvokeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return config, IntegrationMetadata{}, OnFunctionInvokeMetadata{}, fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	return config, integrationMetadata, metadata, nil
}

func cleanupOldFunctionRule(ctx core.TriggerContext, client *Client, metadata OnFunctionInvokeMetadata, newCompartmentID string) error {
	if metadata.EventsRuleID == "" || metadata.CompartmentID == newCompartmentID {
		return nil
	}
	if err := client.DeleteEventsRule(metadata.EventsRuleID); err != nil {
		ctx.Logger.Warnf("failed to delete old Events rule %q: %v", metadata.EventsRuleID, err)
	}
	return nil
}

func ensureFunctionEventsRule(ctx core.TriggerContext, client *Client, compartmentID, topicID string) (string, error) {
	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return "", fmt.Errorf("failed to set up webhook URL: %w", err)
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse webhook URL: %w", err)
	}
	segments := strings.Split(strings.TrimRight(parsedURL.Path, "/"), "/")
	webhookID := segments[len(segments)-1]
	ruleName := fmt.Sprintf("superplane-function-invoked-%s", webhookID)

	condition := `{"eventType": ["com.oraclecloud.functions.invokefunction"]}`
	rule, err := client.CreateEventsRule(compartmentID, ruleName, condition, topicID)
	if err != nil {
		return "", fmt.Errorf("failed to create Events rule: %w", err)
	}

	return rule.ID, nil
}

func (t *OnFunctionInvoke) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnFunctionInvoke) HandleHook(_ core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnFunctionInvoke) Cleanup(ctx core.TriggerContext) error {
	var metadata OnFunctionInvokeMetadata
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

// HandleWebhook processes inbound requests forwarded by OCI Notifications.
func (t *OnFunctionInvoke) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	cfg := OnFunctionInvokeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Handle ONS subscription confirmation handshake.
	if confirmURL := ctx.Headers.Get("X-OCI-NS-ConfirmationURL"); confirmURL != "" {
		return t.handleONSConfirmation(ctx, confirmURL)
	}

	var envelope map[string]any
	if err := json.Unmarshal(ctx.Body, &envelope); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	eventType, _ := envelope["eventType"].(string)
	if eventType != ociEventTypeFunctionInvoke {
		return http.StatusOK, nil, nil
	}

	if !matchesCompartment(envelope, cfg.CompartmentID) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(OnFunctionInvokePayloadType, envelope); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnFunctionInvoke) handleONSConfirmation(ctx core.WebhookRequestContext, confirmURL string) (int, *core.WebhookResponseBody, error) {
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
