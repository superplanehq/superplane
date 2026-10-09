package bitbucket

import (
	"errors"
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	EventTypes     []string `json:"eventTypes"`
	RepositorySlug string   `json:"repositorySlug"`
}

type BitbucketWebhook struct {
	UUID string `json:"uuid"`
}

type BitbucketWebhookHandler struct{}

func (h *BitbucketWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	if configA.RepositorySlug != configB.RepositorySlug {
		return false, nil
	}

	// Check if A contains all events from B (A is superset of B)
	// This allows webhook sharing when existing webhook has more events than needed
	for _, eventB := range configB.EventTypes {
		if !slices.Contains(configA.EventTypes, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *BitbucketWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}
	requestedConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	// Webhooks are per repository. A different repository never merges.
	if currentConfig.RepositorySlug != requestedConfig.RepositorySlug {
		return current, false, nil
	}

	// ponytail: union event types so one repository webhook serves every
	// trigger; re-provision updates that remote hook with the union
	merged := WebhookConfiguration{
		EventTypes:     slices.Clone(currentConfig.EventTypes),
		RepositorySlug: currentConfig.RepositorySlug,
	}
	changed := false
	for _, event := range requestedConfig.EventTypes {
		if !slices.Contains(merged.EventTypes, event) {
			merged.EventTypes = append(merged.EventTypes, event)
			changed = true
		}
	}
	return merged, changed, nil
}

func (h *BitbucketWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(metadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	if config.RepositorySlug == "" {
		return nil, fmt.Errorf("repository is required")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	existing := BitbucketWebhook{}
	if raw := ctx.Webhook.GetMetadata(); raw != nil {
		if err := mapstructure.Decode(raw, &existing); err != nil {
			return nil, fmt.Errorf("failed to decode webhook metadata: %w", err)
		}
	}
	if existing.UUID != "" {
		hook, err := client.UpdateWebhook(
			metadata.Workspace.Slug,
			config.RepositorySlug,
			existing.UUID,
			ctx.Webhook.GetURL(),
			string(secret),
			config.EventTypes,
		)
		if err == nil {
			if hook.UUID == "" {
				hook.UUID = existing.UUID
			}
			return &BitbucketWebhook{UUID: hook.UUID}, nil
		}
		if !errors.Is(err, errBitbucketWebhookNotFound) {
			return nil, fmt.Errorf("error updating webhook: %w", err)
		}
	}

	hook, err := client.CreateWebhook(
		metadata.Workspace.Slug,
		config.RepositorySlug,
		ctx.Webhook.GetURL(),
		string(secret),
		config.EventTypes,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %w", err)
	}

	return &BitbucketWebhook{UUID: hook.UUID}, nil
}

func (h *BitbucketWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(metadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	webhook := BitbucketWebhook{}
	err = mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook)
	if err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	// If the webhook was never created (Setup failed), there's nothing to clean up.
	if webhook.UUID == "" {
		return nil
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	err = client.DeleteWebhook(metadata.Workspace.Slug, config.RepositorySlug, webhook.UUID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %w", err)
	}

	return nil
}
