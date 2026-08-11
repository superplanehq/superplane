package jira

import (
	"fmt"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/retry"
)

// webhookKindAlert marks a WebhookConfiguration as a dedicated JSM Ops alert webhook rather than
// the shared Jira issue/comment webhook (the default, empty Kind) - the two use entirely
// different Atlassian APIs and need different registration, dedup, and cleanup behavior.
const webhookKindAlert = "alert"

// WebhookConfiguration covers two unrelated registrations: an empty Kind tracks the union of
// native Jira event names the shared issue/comment webhook must deliver (see allProjectsJQLFilter
// and legacyIssueEvents), while Kind "alert" is a dedicated JSM Ops alert webhook scoped by
// TeamID - CompareConfig decides which of the two gets deduped.
type WebhookConfiguration struct {
	Kind   string   `json:"kind,omitempty" mapstructure:"kind,omitempty"`
	TeamID string   `json:"teamId,omitempty" mapstructure:"teamId,omitempty"`
	Events []string `json:"events,omitempty" mapstructure:"events,omitempty"`
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

// legacyIssueEvents is what every shared Jira webhook registered before WebhookConfiguration
// tracked Events explicitly - an empty stored Events list means a row predates that change, not
// that the webhook currently delivers nothing, since jira.onIssue was the only trigger able to
// create it and always registered exactly these.
var legacyIssueEvents = []string{issueEventCreated, issueEventUpdated, issueEventDeleted}

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, configB := WebhookConfiguration{}, WebhookConfiguration{}
	_ = mapstructure.Decode(a, &configA)
	_ = mapstructure.Decode(b, &configB)

	// Alert webhooks are scoped by team, not shared like the issue/comment webhook - two configs
	// only match if both are alert configs for the same team, so publishing an unchanged
	// jira.onAlert trigger doesn't tear down and recreate its JSM Ops integration every time.
	if configA.Kind == webhookKindAlert || configB.Kind == webhookKindAlert {
		return configA.Kind == webhookKindAlert && configB.Kind == webhookKindAlert && configA.TeamID == configB.TeamID, nil
	}

	return true, nil
}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, requestedConfig := WebhookConfiguration{}, WebhookConfiguration{}
	_ = mapstructure.Decode(current, &currentConfig)
	_ = mapstructure.Decode(requested, &requestedConfig)

	baseline := currentConfig.Events
	if len(baseline) == 0 {
		baseline = legacyIssueEvents
	}

	merged := mergeEvents(baseline, requestedConfig.Events)
	if len(merged) == len(baseline) {
		return current, false, nil
	}

	return WebhookConfiguration{Events: merged}, true, nil
}

// mergeEvents returns the union of current and additional, preserving current's order so an
// unrelated Merge call doesn't reorder (and thus needlessly re-provision) an unchanged webhook.
func mergeEvents(current, additional []string) []string {
	merged := append([]string{}, current...)
	for _, event := range additional {
		if !slices.Contains(merged, event) {
			merged = append(merged, event)
		}
	}
	return merged
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	_ = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)

	if config.Kind == webhookKindAlert {
		return h.setupAlertWebhook(ctx, config)
	}

	return h.setupIssueWebhook(ctx, config)
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

func (h *JiraWebhookHandler) setupIssueWebhook(ctx core.WebhookHandlerContext, config WebhookConfiguration) (any, error) {
	events := config.Events
	if len(events) == 0 {
		events = legacyIssueEvents
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	previous := WebhookMetadata{}
	_ = mapstructure.Decode(ctx.Webhook.GetMetadata(), &previous)

	// This single registration must cover every project and every trigger type sharing it
	// (jira.onIssue, jira.onIssueComment, jira.onIncident) - each trigger filters to its own
	// configured project and events itself, in HandleWebhook.
	var webhookID int64
	createWebhook := func() error {
		var createErr error
		webhookID, createErr = client.CreateIssueWebhook(ctx.Webhook.GetURL(), allProjectsJQLFilter, events)
		return createErr
	}

	if previous.WebhookID == nil {
		if err := createWebhook(); err != nil {
			return nil, fmt.Errorf("failed to create Jira webhook: %w", err)
		}
	} else {
		if err := client.DeleteIssueWebhooks([]int64{*previous.WebhookID}); err != nil {
			return nil, fmt.Errorf("failed to delete previous Jira webhook: %w", err)
		}

		// Deleting a working registration before its replacement exists briefly silences every
		// trigger sharing it, since Jira allows only one registration per connection - retry the
		// create tightly so a transient failure recovers in seconds rather than waiting for the
		// provisioner's own, much slower retry cadence.
		retryErr := retry.WithConstantWait(createWebhook, retry.Options{
			Task:        "recreate Jira webhook after widening events",
			MaxAttempts: 2,
			Wait:        2 * time.Second,
		})
		if retryErr != nil {
			return nil, fmt.Errorf("failed to create Jira webhook: %w", retryErr)
		}
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
