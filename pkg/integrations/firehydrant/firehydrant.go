package firehydrant

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("firehydrant", &FireHydrant{}, &FireHydrantWebhookHandler{})
}

type FireHydrant struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

type Metadata struct {
	Services []Service `json:"services"`
}

func (f *FireHydrant) Name() string {
	return "firehydrant"
}

func (f *FireHydrant) Label() string {
	return "FireHydrant"
}

func (f *FireHydrant) Icon() string {
	return "flame"
}

func (f *FireHydrant) Description() string {
	return "Manage and react to incidents in FireHydrant"
}

func (f *FireHydrant) Instructions() string {
	return ""
}

func (f *FireHydrant) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API key from FireHydrant. You can generate one in Organization Settings > API Keys.",
		},
	}
}

func (f *FireHydrant) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (f *FireHydrant) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
	}
}

func (f *FireHydrant) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (f *FireHydrant) Sync(ctx core.SyncContext) error {
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

func (f *FireHydrant) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (f *FireHydrant) Actions() []core.Action {
	return []core.Action{}
}

func (f *FireHydrant) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (f *FireHydrant) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "service":
		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(metadata.Services))
		for _, service := range metadata.Services {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: service.Name,
				ID:   service.ID,
			})
		}
		return resources, nil

	case "severity":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		severities, err := client.ListSeverities()
		if err != nil {
			return nil, fmt.Errorf("failed to list severities: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(severities))
		for _, severity := range severities {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: severity.Slug,
				ID:   severity.ID,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

// verifyWebhookSignature verifies the FireHydrant webhook signature.
// The signature is sent in the X-FireHydrant-Signature header.
// FireHydrant uses HMAC-SHA256 to sign the request body.
func verifyWebhookSignature(signature string, body, secret []byte) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	// Compute expected signature: HMAC-SHA256(body, secret)
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
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
