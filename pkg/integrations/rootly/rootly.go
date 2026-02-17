package rootly

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("rootly", &Rootly{}, &RootlyWebhookHandler{})
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
		&CreateEvent{},
		&UpdateIncident{},
		&GetIncident{},
	}
}

func (r *Rootly) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
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
