package jira

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration tracks the union of native Jira event names the shared webhook must
// deliver: Jira's dynamic webhook API accepts only one registered callback URL per OAuth
// connection
type WebhookConfiguration struct {
	Events []string `json:"events,omitempty" mapstructure:"events,omitempty"`
}

type WebhookMetadata struct {
	WebhookID *int64 `json:"webhookId,omitempty" mapstructure:"webhookId,omitempty"`
}

const allProjectsJQLFilter = "project != EMPTY"

var legacyIssueEvents = []string{issueEventCreated, issueEventUpdated, issueEventDeleted}

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
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
	if previous.WebhookID != nil {
		if err := client.DeleteIssueWebhooks([]int64{*previous.WebhookID}); err != nil {
			return nil, fmt.Errorf("failed to delete previous Jira webhook: %w", err)
		}
	}

	// This single registration must cover every project and every trigger type sharing it
	// (jira.onIssue, jira.onIssueComment) - each trigger filters to its own configured project
	// and events itself, in HandleWebhook.
	webhookID, err := client.CreateIssueWebhook(ctx.Webhook.GetURL(), allProjectsJQLFilter, events)
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
