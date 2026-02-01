package rootly

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("rootly", &Rootly{})
}

type Rootly struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

type Metadata struct {
	Services []Service `json:"services"`
}

func (r *Rootly) Name() string {
	return "rootly"
}

func (r *Rootly) Label() string {
	return "Rootly"
}

func (r *Rootly) Icon() string {
	return "alert-triangle"
}

func (r *Rootly) Description() string {
	return "Manage and react to incidents in Rootly"
}

func (r *Rootly) Instructions() string {
	return ""
}

func (r *Rootly) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API key from Rootly. You can generate one in Configuration > API Keys.",
		},
	}
}

func (r *Rootly) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (r *Rootly) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncidentCreated{},
		&OnIncidentResolved{},
	}
}

func (r *Rootly) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (r *Rootly) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	services, err := client.ListServices()
	if err != nil {
		return fmt.Errorf("error listing services: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Services: services})
	ctx.Integration.Ready()
	return nil
}

func (r *Rootly) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

type WebhookConfiguration struct {
	Events []string `json:"events"`
}

type WebhookMetadata struct {
	EndpointID string `json:"endpointId"`
}

func (r *Rootly) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	// Reuse only when event sets are identical (e.g. two "On Incident Created" nodes share one webhook).
	// Different triggers (Created vs Resolved) have different event sets, so we create a new webhook
	// and both endpoints appear in Rootly.
	if len(configA.Events) != len(configB.Events) {
		return false, nil
	}
	for _, e := range configB.Events {
		if !slices.Contains(configA.Events, e) {
			return false, nil
		}
	}
	return true, nil
}

func (r *Rootly) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	// Rootly requires a unique name per endpoint; use webhook ID so multiple triggers get separate endpoints.
	name := "SuperPlane " + ctx.Webhook.GetID()
	endpoint, err := client.CreateWebhookEndpoint(name, ctx.Webhook.GetURL(), config.Events)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook endpoint: %v", err)
	}

	err = ctx.Webhook.SetSecret([]byte(endpoint.Secret))
	if err != nil {
		return nil, fmt.Errorf("error updating webhook secret: %v", err)
	}

	return WebhookMetadata{
		EndpointID: endpoint.ID,
	}, nil
}

func (r *Rootly) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	if ctx.Webhook.GetMetadata() != nil {
		if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
			return fmt.Errorf("error decoding webhook metadata: %v", err)
		}
	}

	// Nothing to delete if we never got an endpoint ID (e.g. webhook was never provisioned or is old).
	if metadata.EndpointID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteWebhookEndpoint(metadata.EndpointID)
	if err != nil {
		return fmt.Errorf("error deleting webhook endpoint: %v", err)
	}

	return nil
}

func (r *Rootly) Actions() []core.Action {
	return []core.Action{}
}

func (r *Rootly) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

// verifyWebhookSignature verifies the Rootly webhook signature.
// The signature format is: "t=<timestamp>, v1=<signature>"
// where signature = HMAC-SHA256(timestamp + body, secret)
func verifyWebhookSignature(signature string, body, secret []byte) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	// Parse the signature header
	// Format: "t=1492774588,v1=6657a869..."
	var timestamp, sig string
	parts := strings.Split(signature, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}

	if timestamp == "" || sig == "" {
		return fmt.Errorf("invalid signature format")
	}

	// Compute expected signature: HMAC-SHA256(timestamp + body, secret)
	payload := append([]byte(timestamp), body...)
	expectedSig := computeHMACSHA256(secret, payload)

	if !hmacEqual([]byte(sig), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// computeHMACSHA256 computes HMAC-SHA256 and returns hex-encoded result
func computeHMACSHA256(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// hmacEqual compares two HMAC values in constant time
func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
