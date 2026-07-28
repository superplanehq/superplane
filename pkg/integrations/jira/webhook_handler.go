package jira

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration has no fields on purpose: Jira's dynamic webhook API accepts only one
// registered callback URL per OAuth connection ("Only a single URL per user is allowed to be
// registered via REST API"), so every jira.onIssue trigger under the same integration must
// share the one Jira-side registration, regardless of which project or events it configures.
// CompareConfig below always reports a match so the platform's webhook provisioner dedups them
// into a single webhook record instead of trying to register a distinct URL for each trigger.
type WebhookConfiguration struct{}

type WebhookMetadata struct {
	WebhookID *int64 `json:"webhookId,omitempty" mapstructure:"webhookId,omitempty"`
}

// allProjectsJQLFilter matches every issue in every project. An empty jqlFilter is rejected
// outright by Atlassian ("Empty JQL search not supported") even though the key itself must be
// present - this is the simplest clause confirmed (live, against a real site) to both be accepted
// and match unconditionally, needed since this single registration is shared by every
// jira.onIssue trigger on the integration regardless of project.
const allProjectsJQLFilter = "project != EMPTY"

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
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
