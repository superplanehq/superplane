package oci

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
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
	EventsRuleID  string `json:"eventsRuleId" mapstructure:"eventsRuleId"`
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
	var config OnComputeInstanceCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	if config.CompartmentID == "" {
		return fmt.Errorf("compartmentId is required")
	}

	// Read the shared topic OCID from the integration metadata set during Sync.
	var integrationMetadata IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if integrationMetadata.TopicID == "" {
		return fmt.Errorf("integration topic not ready yet; ensure the OCI integration has been fully set up")
	}

	var metadata OnComputeInstanceCreatedMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	// If the Events rule already exists for this compartment, skip re-creation but
	// still call RequestWebhook so the subscription is retried if it previously
	// failed or was never provisioned.
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

	// Delete old Events rule if compartment changed.
	if metadata.EventsRuleID != "" && metadata.CompartmentID != config.CompartmentID {
		if err := client.DeleteEventsRule(metadata.EventsRuleID); err != nil {
			ctx.Logger.Warnf("failed to delete old Events rule %q: %v", metadata.EventsRuleID, err)
		}
	}

	condition := `{"eventType": ["com.oraclecloud.computeapi.launchinstance.end"]}`

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to set up webhook URL: %w", err)
	}
	ruleName := fmt.Sprintf("superplane-compute-instance-created-%s", path.Base(webhookURL))
	rule, err := client.CreateEventsRule(
		config.CompartmentID,
		ruleName,
		condition,
		integrationMetadata.TopicID,
	)
	if err != nil {
		return fmt.Errorf("failed to create Events rule: %w", err)
	}

	if err := ctx.Metadata.Set(OnComputeInstanceCreatedMetadata{
		CompartmentID: config.CompartmentID,
		EventsRuleID:  rule.ID,
	}); err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	// Request a per-trigger HTTPS subscription to the shared topic.
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		CompartmentID: config.CompartmentID,
		TopicID:       integrationMetadata.TopicID,
	})
}

func (t *OnComputeInstanceCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnComputeInstanceCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnComputeInstanceCreated) Cleanup(ctx core.TriggerContext) error {
	var metadata OnComputeInstanceCreatedMetadata
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
func (t *OnComputeInstanceCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	cfg := OnComputeInstanceCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(ctx.Body, &envelope); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	// Handle ONS subscription confirmation handshake.
	// For CUSTOM_HTTPS subscriptions, OCI sends the confirmation URL in the
	// X-OCI-NS-ConfirmationURL HTTP header (not in the JSON body).
	// The endpoint must GET that URL to activate the subscription.
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

	if err := ctx.Events.Emit(OnComputeInstanceCreatedPayloadType, envelope); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil, nil
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
