package jira

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration describes the events and JQL filter for a per-node
// Jira webhook subscription. The webhook handler hands this to Jira's
// `/rest/api/3/webhook` endpoint.
type WebhookConfiguration struct {
	Events    []string `json:"events" mapstructure:"events"`
	JQLFilter string   `json:"jqlFilter,omitempty" mapstructure:"jqlFilter"`
}

// WebhookMetadata stores the webhook ID returned by Jira so we can later
// delete or refresh the subscription.
type WebhookMetadata struct {
	IDs []int `json:"ids" mapstructure:"ids"`
}

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, err := decodeWebhookConfiguration(a)
	if err != nil {
		return false, err
	}

	configB, err := decodeWebhookConfiguration(b)
	if err != nil {
		return false, err
	}

	if configA.JQLFilter != configB.JQLFilter {
		return false, nil
	}

	return sameStringSet(configA.Events, configB.Events), nil
}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	if reflect.DeepEqual(current, requested) {
		return current, false, nil
	}
	return requested, true, nil
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook config: %v", err)
	}

	if len(config.Events) == 0 {
		return nil, fmt.Errorf("webhook configuration must include at least one event")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %v", err)
	}

	ids, err := client.RegisterWebhooks(ctx.Webhook.GetURL(), []WebhookRegistration{
		{
			Events:    config.Events,
			JQLFilter: defaultJQL(config.JQLFilter),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error registering Jira webhook: %v", err)
	}

	return &WebhookMetadata{IDs: ids}, nil
}

func (h *JiraWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %v", err)
	}

	if len(metadata.IDs) == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %v", err)
	}

	if err := client.DeleteWebhooks(metadata.IDs); err != nil {
		return fmt.Errorf("error deleting Jira webhook: %v", err)
	}
	return nil
}

func decodeWebhookConfiguration(value any) (WebhookConfiguration, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return WebhookConfiguration{}, err
	}
	return config, nil
}

// defaultJQL returns a JQL string that always evaluates to true when the
// caller did not provide one. Jira's webhook endpoint requires a non-empty
// JQL filter for issue-related events.
func defaultJQL(value string) string {
	if value == "" {
		return "issuekey is not EMPTY"
	}
	return value
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
