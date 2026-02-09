package jira

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListWebhooks struct{}

func (c *ListWebhooks) Name() string {
	return "jira.listWebhooks"
}

func (c *ListWebhooks) Label() string {
	return "List Webhooks"
}

func (c *ListWebhooks) Description() string {
	return "List all webhooks registered with Jira for this OAuth app"
}

func (c *ListWebhooks) Documentation() string {
	return "Lists all webhooks registered via the Jira REST API for the current OAuth application."
}

func (c *ListWebhooks) Icon() string {
	return "jira"
}

func (c *ListWebhooks) Color() string {
	return "blue"
}

func (c *ListWebhooks) ExampleOutput() map[string]any {
	return map[string]any{
		"total": 1,
		"webhooks": []map[string]any{
			{"id": 12345, "jqlFilter": "project = TEST", "events": []string{"jira:issue_created"}},
		},
	}
}

func (c *ListWebhooks) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListWebhooks) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *ListWebhooks) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListWebhooks) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListWebhooks) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	webhooks, err := client.ListWebhooks()
	if err != nil {
		return err
	}

	results := make([]map[string]any, len(webhooks.Values))
	for i, w := range webhooks.Values {
		results[i] = map[string]any{
			"id":        w.ID,
			"jqlFilter": w.JQLFilter,
			"events":    w.Events,
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"jira.webhookList",
		[]any{map[string]any{
			"total":    webhooks.Total,
			"webhooks": results,
		}},
	)
}

func (c *ListWebhooks) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListWebhooks) Actions() []core.Action {
	return nil
}

func (c *ListWebhooks) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListWebhooks) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ListWebhooks) Cleanup(ctx core.SetupContext) error {
	return nil
}
