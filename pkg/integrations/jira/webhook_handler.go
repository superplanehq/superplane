package jira

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/retry"
)

// Kind distinguishes stored legacy alert webhooks from the shared issue/comment webhook.
type WebhookConfiguration struct {
	Kind     string   `json:"kind,omitempty" mapstructure:"kind,omitempty"`
	Events   []string `json:"events,omitempty" mapstructure:"events,omitempty"`
	Projects []string `json:"projects,omitempty" mapstructure:"projects,omitempty"`
}

type WebhookMetadata struct {
	WebhookID *int64 `json:"webhookId,omitempty" mapstructure:"webhookId,omitempty"`
}

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

	return configA.Kind == "" && configB.Kind == "", nil
}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, requestedConfig := WebhookConfiguration{}, WebhookConfiguration{}
	_ = mapstructure.Decode(current, &currentConfig)
	_ = mapstructure.Decode(requested, &requestedConfig)

	baseline := currentConfig.Events
	if len(baseline) == 0 {
		baseline = legacyIssueEvents
	}

	mergedEvents := mergeUniqueStrings(baseline, requestedConfig.Events)
	mergedProjects := mergeUniqueStrings(currentConfig.Projects, requestedConfig.Projects)
	if len(mergedEvents) == len(baseline) && len(mergedProjects) == len(currentConfig.Projects) {
		return current, false, nil
	}

	merged := WebhookConfiguration{Events: mergedEvents}
	if len(mergedProjects) > 0 {
		merged.Projects = mergedProjects
	}
	return merged, true, nil
}

// mergeUniqueStrings returns the union of current and additional, preserving current's order
// so an unrelated Merge call doesn't reorder (and thus needlessly re-provision) an unchanged
// webhook.
func mergeUniqueStrings(current, additional []string) []string {
	merged := append([]string{}, current...)
	for _, value := range additional {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(merged, value) {
			continue
		}
		merged = append(merged, value)
	}
	return merged
}

// issueWebhookJQLFilter builds the dynamic-webhook JQL Atlassian accepts: only the project
// field, and only =, !=, IN, or NOT IN. project != EMPTY can register and still match
// nothing, so this lists the projects of every trigger that shares the webhook.
func issueWebhookJQLFilter(projects []string) (string, error) {
	keys := make([]string, 0, len(projects))
	for _, project := range projects {
		project = strings.TrimSpace(project)
		if project == "" {
			continue
		}
		keys = append(keys, project)
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("at least one project is required to register the Jira issue webhook")
	}

	quoted := make([]string, len(keys))
	for i, project := range keys {
		quoted[i] = `"` + jqlQuotedProjectKey(project) + `"`
	}
	if len(quoted) == 1 {
		return "project = " + quoted[0], nil
	}
	return "project IN (" + strings.Join(quoted, ",") + ")", nil
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	_ = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)

	if config.Kind != "" {
		return nil, fmt.Errorf("Jira Service Management webhooks are no longer supported")
	}

	return h.setupIssueWebhook(ctx, config)
}

func (h *JiraWebhookHandler) setupIssueWebhook(ctx core.WebhookHandlerContext, config WebhookConfiguration) (any, error) {
	events := config.Events
	if len(events) == 0 {
		events = legacyIssueEvents
	}

	jqlFilter, err := issueWebhookJQLFilter(config.Projects)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	previous := WebhookMetadata{}
	_ = mapstructure.Decode(ctx.Webhook.GetMetadata(), &previous)

	// This single registration must cover every project and every trigger type sharing it
	// (jira.onIssue, jira.onIssueComment) - each trigger filters to its own
	// configured project and events itself, in HandleWebhook.
	var webhookID int64
	createWebhook := func() error {
		var createErr error
		webhookID, createErr = h.createIssueWebhookRecoveringURLConflict(client, ctx, jqlFilter, events)
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

	if config.Kind != "" {
		return nil
	}

	return h.cleanupIssueWebhook(ctx)
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
