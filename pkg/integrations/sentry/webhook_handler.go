package sentry

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration defines the configuration for a Sentry webhook
type WebhookConfiguration struct {
	Project string `json:"project"`
}

// WebhookMetadata stores metadata about the created service hook
type WebhookMetadata struct {
	HookID  string `json:"hookId"`
	Project string `json:"project"`
	Secret  string `json:"secret"`
}

type SentryWebhookHandler struct{}

// CompareConfig determines if two webhook configurations are equivalent
func (h *SentryWebhookHandler) CompareConfig(a, b any) (bool, error) {
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

// Merge merges webhook configurations
func (h *SentryWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// Setup creates a service hook in Sentry via the API
// POST /api/0/projects/{organization_slug}/{project_slug}/hooks/
func (h *SentryWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	configuration := WebhookConfiguration{}
	err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	if configuration.Project == "" {
		return nil, fmt.Errorf("project is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	// Create a service hook via Sentry API
	// Service hooks support: event.alert, event.created
	hook, err := client.CreateServiceHook(
		configuration.Project,
		ctx.Webhook.GetURL(),
		[]string{"event.created", "event.alert"},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating service hook: %v", err)
	}

	// Store the hook secret in the webhook context for signature verification
	if hook.Secret != "" {
		err = ctx.Webhook.SetSecret([]byte(hook.Secret))
		if err != nil {
			// Try to cleanup the created hook if we can't store the secret
			_ = client.DeleteServiceHook(configuration.Project, hook.ID)
			return nil, fmt.Errorf("error storing webhook secret: %v", err)
		}
	}

	return WebhookMetadata{
		HookID:  hook.ID,
		Project: configuration.Project,
		Secret:  hook.Secret,
	}, nil
}

// Cleanup removes a service hook from Sentry via the API
// DELETE /api/0/projects/{organization_slug}/{project_slug}/hooks/{hook_id}/
func (h *SentryWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	if metadata.HookID == "" || metadata.Project == "" {
		// No hook to cleanup
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteServiceHook(metadata.Project, metadata.HookID)
	if err != nil {
		return fmt.Errorf("error deleting service hook: %v", err)
	}

	return nil
}
