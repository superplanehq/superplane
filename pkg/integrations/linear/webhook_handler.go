package linear

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// Linear returns this string in API errors when the webhook was already deleted.
const linearErrEntityNotFound = "Entity not found"

// WebhookConfiguration is the config passed to RequestWebhook for Linear webhooks.
type WebhookConfiguration struct {
	TeamID         string   `json:"teamId"`
	AllPublicTeams bool     `json:"allPublicTeams"`
	ResourceTypes  []string `json:"resourceTypes"`
}

// WebhookMetadata is stored after creating a webhook (used by Cleanup).
type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
}

// LinearWebhookHandler implements core.WebhookHandler for Linear (create/delete webhooks via GraphQL).
type LinearWebhookHandler struct{}

// Merge returns current config unchanged; Linear webhooks do not support merging.
func (h *LinearWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// CompareConfig returns true if both configs describe the same webhook (team, allPublicTeams, resourceTypes).
func (h *LinearWebhookHandler) CompareConfig(a, b any) (bool, error) {
	var configA, configB WebhookConfiguration
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}
	if configA.AllPublicTeams != configB.AllPublicTeams || configA.TeamID != configB.TeamID {
		return false, nil
	}
	return resourceTypesEqual(configA.ResourceTypes, configB.ResourceTypes), nil
}

func resourceTypesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, r := range b {
		if !slices.Contains(a, r) {
			return false
		}
	}
	return true
}

// Setup creates the webhook in Linear and stores its ID (and optional secret) in metadata.
func (h *LinearWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	var config WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("decode webhook configuration: %w", err)
	}

	teamID := ptrIfNonEmpty(config.TeamID)
	resourceTypes := config.ResourceTypes
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"Issue"}
	}

	webhookID, secret, err := client.WebhookCreate(ctx.Webhook.GetURL(), teamID, config.AllPublicTeams, resourceTypes)
	if err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}

	if len(secret) > 0 {
		if err := ctx.Webhook.SetSecret(secret); err != nil {
			return nil, fmt.Errorf("set webhook secret: %w", err)
		}
	}

	return WebhookMetadata{WebhookID: webhookID}, nil
}

func ptrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Cleanup deletes the webhook in Linear. Treats "Entity not found" as success so the cleanup worker can remove our record.
func (h *LinearWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var metadata WebhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("decode webhook metadata: %w", err)
	}
	if metadata.WebhookID == "" {
		return nil
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.WebhookDelete(metadata.WebhookID); err != nil && !isLinearNotFound(err) {
		return err
	}
	return nil
}

func isLinearNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), linearErrEntityNotFound)
}
