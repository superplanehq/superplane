package linear

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config passed to RequestWebhook for Linear webhooks.
type WebhookConfiguration struct {
	TeamID         string   `json:"teamId"`
	AllPublicTeams bool     `json:"allPublicTeams"`
	ResourceTypes  []string `json:"resourceTypes"`
}

// WebhookMetadata stored after creating a webhook.
type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
}

type LinearWebhookHandler struct{}

func (h *LinearWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *LinearWebhookHandler) CompareConfig(a, b any) (bool, error) {
	var configA, configB WebhookConfiguration
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}
	if configA.AllPublicTeams != configB.AllPublicTeams {
		return false, nil
	}
	if configA.TeamID != configB.TeamID {
		return false, nil
	}
	if len(configA.ResourceTypes) != len(configB.ResourceTypes) {
		return false, nil
	}
	for _, r := range configB.ResourceTypes {
		if !slices.Contains(configA.ResourceTypes, r) {
			return false, nil
		}
	}
	return true, nil
}

func (h *LinearWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	var config WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("decode webhook configuration: %w", err)
	}

	var teamID *string
	if config.TeamID != "" {
		teamID = &config.TeamID
	}
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

func (h *LinearWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var metadata WebhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("decode webhook metadata: %w", err)
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	return client.WebhookDelete(metadata.WebhookID)
}
