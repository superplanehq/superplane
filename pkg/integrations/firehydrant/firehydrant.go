package firehydrant

import (
	"crypto/hmac"
	"crypto/sha256"
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
	return "alert-triangle"
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
			Description: "API key from FireHydrant. You can create one in Settings > Bot tokens.",
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
		&OnNewIncident{},
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
}

func (f *FireHydrant) Actions() []core.Action {
	return []core.Action{}
}

func (f *FireHydrant) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

// verifyWebhookSignature verifies the FireHydrant webhook signature.
// FireHydrant signs webhooks with HMAC-SHA256: signature = hex(HMAC-SHA256(secret, body)).
// The signature is sent in the "fh-signature" header.
func verifyWebhookSignature(signature string, body, secret []byte) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	expectedSig := computeHMACSHA256(secret, body)

	if !hmacEqual([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func computeHMACSHA256(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

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
