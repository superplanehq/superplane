package jira

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// webhookKindAlert marks a WebhookConfiguration as a dedicated JSM Ops alert webhook rather than
// the shared Jira issue/comment webhook (the default, empty Kind) - the two use entirely
// different Atlassian APIs and need different registration, dedup, and cleanup behavior.
const webhookKindAlert = "alert"

// WebhookConfiguration covers two unrelated registrations: an empty Kind for the shared Jira
// issue/comment webhook (see allProjectsJQLFilter), and Kind "alert" for a dedicated JSM Ops
// alert webhook (see webhookKindAlert) - CompareConfig decides which of the two gets deduped.
type WebhookConfiguration struct {
	Kind   string `json:"kind,omitempty" mapstructure:"kind,omitempty"`
	TeamID string `json:"teamId,omitempty" mapstructure:"teamId,omitempty"`
}

type WebhookMetadata struct {
	WebhookID *int64 `json:"webhookId,omitempty" mapstructure:"webhookId,omitempty"`
}

// AlertWebhookMetadata is the JSM Ops integration id for a dedicated jira.onAlert webhook.
type AlertWebhookMetadata struct {
	IntegrationID string `json:"integrationId,omitempty" mapstructure:"integrationId,omitempty"`
}

// allProjectsJQLFilter matches every issue in every project. An empty jqlFilter is rejected
// outright by Atlassian ("Empty JQL search not supported") even though the key itself must be
// present - this is the simplest clause confirmed (live, against a real site) to both be accepted
// and match unconditionally, needed since this single registration is shared by every
// jira.onIssue trigger on the integration regardless of project.
const allProjectsJQLFilter = "project != EMPTY"

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, configB := WebhookConfiguration{}, WebhookConfiguration{}
	_ = mapstructure.Decode(a, &configA)
	_ = mapstructure.Decode(b, &configB)

	if configA.Kind == webhookKindAlert || configB.Kind == webhookKindAlert {
		return false, nil
	}

	return true, nil
}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	_ = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)

	if config.Kind == webhookKindAlert {
		return h.setupAlertWebhook(ctx, config)
	}

	return h.setupIssueWebhook(ctx)
}

func (h *JiraWebhookHandler) setupAlertWebhook(ctx core.WebhookHandlerContext, config WebhookConfiguration) (any, error) {
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	name := fmt.Sprintf("SuperPlane (%s)", ctx.Webhook.GetID())
	integration, err := client.CreateAlertWebhookIntegration(cloudID, name, ctx.Webhook.GetURL(), config.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSM Ops alert webhook: %w", err)
	}

	return &AlertWebhookMetadata{IntegrationID: integration.ID}, nil
}

func (h *JiraWebhookHandler) setupIssueWebhook(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// This single registration must cover every project, since it is shared by every
	// jira.onIssue trigger on this integration - each one filters to its own configured project
	// and events itself, in HandleWebhook.
	webhookID, err := client.CreateIssueWebhook(ctx.Webhook.GetURL(), allProjectsJQLFilter, []string{
		issueEventCreated, issueEventUpdated, issueEventDeleted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira webhook: %w", err)
	}

	// Atlassian expires dynamic webhooks 30 days after creation unless refreshed. Mirror the id
	// onto the integration itself - the refreshWebhook hook (see jira.go) only has access to the
	// integration, not this webhook record - and kick off the self-rescheduling refresh loop.
	integrationMetadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata)
	integrationMetadata.WebhookID = &webhookID
	ctx.Integration.SetMetadata(integrationMetadata)

	if err := ctx.Integration.ScheduleActionCall(refreshWebhookHookName, map[string]any{}, webhookRefreshInterval); err != nil {
		// Jira allows only one registered URL per OAuth connection. If we leave this registration
		// behind after a schedule failure, Setup retries hit that limit, never persist WebhookID
		// into the SuperPlane webhook record, and Cleanup has nothing to delete - orphan forever.
		if delErr := client.DeleteIssueWebhooks([]int64{webhookID}); delErr != nil {
			return nil, fmt.Errorf("failed to schedule webhook refresh: %w (also failed to delete orphaned webhook %d: %v)", err, webhookID, delErr)
		}
		integrationMetadata.WebhookID = nil
		ctx.Integration.SetMetadata(integrationMetadata)
		return nil, fmt.Errorf("failed to schedule webhook refresh: %w", err)
	}

	return &WebhookMetadata{WebhookID: &webhookID}, nil
}

func (h *JiraWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	config := WebhookConfiguration{}
	_ = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)

	if config.Kind == webhookKindAlert {
		return h.cleanupAlertWebhook(ctx)
	}

	return h.cleanupIssueWebhook(ctx)
}

func (h *JiraWebhookHandler) cleanupAlertWebhook(ctx core.WebhookHandlerContext) error {
	metadata := AlertWebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}
	if metadata.IntegrationID == "" {
		return nil
	}

	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return client.DeleteAlertWebhookIntegration(cloudID, metadata.IntegrationID)
}

func (h *JiraWebhookHandler) cleanupIssueWebhook(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	// If Setup never completed successfully, there is nothing registered to delete.
	if metadata.WebhookID == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.DeleteIssueWebhooks([]int64{*metadata.WebhookID}); err != nil {
		return err
	}

	// Clear the mirrored id so a refreshWebhook hook call still in flight finds nothing to
	// refresh, instead of retrying forever against a webhook that no longer exists.
	integrationMetadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata)
	integrationMetadata.WebhookID = nil
	ctx.Integration.SetMetadata(integrationMetadata)

	return nil
}
