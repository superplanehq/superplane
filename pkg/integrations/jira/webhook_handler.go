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

	// No jqlFilter: this single registration must cover every project, since it is shared by
	// every jira.onIssue trigger on this integration - each one filters to its own configured
	// project and events itself, in HandleWebhook.
	webhookID, err := client.CreateIssueWebhook(ctx.Webhook.GetURL(), "", []string{
		issueEventCreated, issueEventUpdated, issueEventDeleted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira webhook: %w", err)
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

	return client.DeleteIssueWebhooks([]int64{*metadata.WebhookID})
}
