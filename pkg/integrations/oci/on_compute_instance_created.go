package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnComputeInstanceCreated struct{}

const (
	ociEventTypeComputeLaunchEnd        = "com.oraclecloud.computeapi.launchinstance.end"
	OnComputeInstanceCreatedPayloadType = "oci.onComputeInstanceCreated"
)

type OnComputeInstanceCreatedConfiguration struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
}

type OnComputeInstanceCreatedMetadata struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
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
	return `The On Compute Instance Created trigger starts a workflow execution whenever an OCI Compute instance launch completes.

## How It Works

When the OCI integration is set up, SuperPlane automatically creates a shared **OCI Notifications (ONS) topic** and subscribes to it. When this trigger is added to a workflow, SuperPlane automatically creates an **OCI Events rule** in the configured compartment that forwards ` + "`com.oraclecloud.computeapi.launchinstance.end`" + ` events to that topic — no manual OCI configuration is required.

## Configuration

- **Compartment**: The compartment to monitor for new Compute instances.

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
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment to filter instance-created events from. Also used when creating the ONS subscription.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
	}
}

func (t *OnComputeInstanceCreated) ExampleData() map[string]any {
	return exampleDataOnComputeInstanceCreated()
}

func (t *OnComputeInstanceCreated) Setup(ctx core.TriggerContext) error {
	config, integrationMetadata, err := decodeSetupInputs(ctx)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(OnComputeInstanceCreatedMetadata{
		CompartmentID: config.CompartmentID,
	}); err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	return requestWebhook(ctx, config.CompartmentID, integrationMetadata.TopicID)
}

// decodeSetupInputs decodes and validates all inputs needed by Setup.
func decodeSetupInputs(ctx core.TriggerContext) (OnComputeInstanceCreatedConfiguration, IntegrationMetadata, error) {
	var config OnComputeInstanceCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return config, IntegrationMetadata{}, fmt.Errorf("failed to decode trigger configuration: %w", err)
	}
	if config.CompartmentID == "" {
		return config, IntegrationMetadata{}, fmt.Errorf("compartmentId is required")
	}

	var integrationMetadata IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return config, IntegrationMetadata{}, fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if integrationMetadata.TopicID == "" {
		return config, IntegrationMetadata{}, fmt.Errorf("integration topic not ready yet; ensure the OCI integration has been fully set up")
	}

	return config, integrationMetadata, nil
}

// requestWebhook asks the integration to provision a per-trigger HTTPS
// subscription to the shared ONS topic.
func requestWebhook(ctx core.TriggerContext, compartmentID, topicID string) error {
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		CompartmentID: compartmentID,
		TopicID:       topicID,
	})
}

func (t *OnComputeInstanceCreated) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnComputeInstanceCreated) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnComputeInstanceCreated) Cleanup(ctx core.TriggerContext) error {
	// Events rules are shared across triggers for the same integration+compartment.
	// They are cleaned up when the integration itself is deleted (see OCI.Cleanup).
	return nil
}

// HandleWebhook processes inbound requests forwarded by OCI Notifications.
func (t *OnComputeInstanceCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	cfg := OnComputeInstanceCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Handle ONS subscription confirmation handshake before parsing the body.
	// For CUSTOM_HTTPS subscriptions, OCI sends the confirmation URL in the
	// X-OCI-NS-ConfirmationURL HTTP header (not in the JSON body), and the
	// confirmation payload may not be a valid JSON object.
	if confirmURL := ctx.Headers.Get("X-OCI-NS-ConfirmationURL"); confirmURL != "" {
		return t.handleONSConfirmation(ctx, confirmURL)
	}

	envelope, err := parseEventEnvelope(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	if !isComputeLaunchEvent(envelope) {
		return http.StatusOK, nil, nil
	}

	if !matchesCompartment(envelope, cfg.CompartmentID) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(OnComputeInstanceCreatedPayloadType, envelope); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnComputeInstanceCreated) handleONSConfirmation(ctx core.WebhookRequestContext, confirmURL string) (int, *core.WebhookResponseBody, error) {
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

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return http.StatusInternalServerError, nil, fmt.Errorf("ONS confirmation returned %d", resp.StatusCode)
	}

	return http.StatusOK, nil, nil
}

func parseEventEnvelope(body []byte) (map[string]any, error) {
	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}
	return envelope, nil
}

func isComputeLaunchEvent(envelope map[string]any) bool {
	eventType, _ := envelope["eventType"].(string)
	return eventType == ociEventTypeComputeLaunchEnd
}

func matchesCompartment(envelope map[string]any, compartmentID string) bool {
	if compartmentID == "" {
		return true
	}
	data, _ := envelope["data"].(map[string]any)
	envelopeCompartmentID, _ := data["compartmentId"].(string)
	return envelopeCompartmentID == compartmentID
}

// validateONSConfirmationURL guards against SSRF by ensuring the URL:
//   - uses the https scheme
//   - has a hostname that ends with .oraclecloud.com
//   - is not a raw IP address
func validateONSConfirmationURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid confirmation URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("confirmation URL must use https, got %q", u.Scheme)
	}
	hostname := u.Hostname()
	if net.ParseIP(hostname) != nil {
		return fmt.Errorf("confirmation URL must not be a raw IP address")
	}
	if !strings.HasSuffix(hostname, ".oraclecloud.com") {
		return fmt.Errorf("confirmation URL hostname %q is not an allowed OCI domain", hostname)
	}
	return nil
}
