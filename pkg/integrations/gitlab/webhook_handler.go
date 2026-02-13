package gitlab

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type GitLabWebhookHandler struct{}

func (h *GitLabWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *GitLabWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	if configA.ProjectID != configB.ProjectID {
		return false, nil
	}

	if configA.EventType != configB.EventType {
		return false, nil
	}

	return true, nil
}

func (h *GitLabWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	hooksClient, err := NewHooksClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create hooks client: %v", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook config: %v", err)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	events := HookEvents{}
	switch config.EventType {
	case "issues":
		events.IssuesEvents = true
	case "merge_requests":
		events.MergeRequestsEvents = true
	case "push":
		events.PushEvents = true
	case "tag_push":
		events.TagPushEvents = true
	case "note":
		events.NoteEvents = true
	case "pipeline":
		events.PipelineEvents = true
	case "releases":
		events.ReleasesEvents = true
	case "milestone":
		events.MilestoneEvents = true
	case "vulnerability":
		events.VulnerabilityEvents = true
	}

	hook, err := hooksClient.CreateHook(config.ProjectID, ctx.Webhook.GetURL(), string(secret), events)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return &WebhookMetadata{ID: hook.ID}, nil
}

func (h *GitLabWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	hooksClient, err := NewHooksClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create hooks client: %v", err)
	}

	webhook := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %v", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return fmt.Errorf("failed to decode webhook config: %v", err)
	}

	if err := hooksClient.DeleteHook(config.ProjectID, webhook.ID); err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}
