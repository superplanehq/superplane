package railway

import (
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
}

type WebhookMetadata struct {
	// Railway webhooks are configured via UI, so we don't have a webhook ID
	// This is kept for potential future use if Railway adds API support
	Project string `json:"project" mapstructure:"project"`
}

type RailwayWebhookHandler struct{}

// CompareConfig checks if two webhook configurations are equivalent.
// Webhooks with the same project can be shared.
func (h *RailwayWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Project == configB.Project, nil
}

// Merge combines current and requested webhook configurations.
// Since Railway webhooks are manually configured, we just return the current config.
func (h *RailwayWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// Setup is called when a webhook needs to be created.
// Since Railway webhooks are UI-only, we just return metadata.
// The webhook URL will be displayed to the user for manual setup.
func (h *RailwayWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, err
	}

	// Railway doesn't have an API for creating webhooks
	// Users must manually configure the webhook URL in Railway UI
	// We just return metadata indicating the project
	return WebhookMetadata{
		Project: config.Project,
	}, nil
}

// Cleanup is called when a webhook should be deleted.
// Since Railway webhooks are UI-only, we cannot delete them programmatically.
// Users must manually remove the webhook from Railway UI.
func (h *RailwayWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	// Railway doesn't have an API for deleting webhooks
	// Users must manually remove the webhook from Railway UI
	return nil
}
