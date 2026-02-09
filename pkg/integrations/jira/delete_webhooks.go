package jira

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteWebhooks struct{}

type DeleteWebhooksConfiguration struct {
	WebhookID *int64 `json:"webhookId" mapstructure:"webhookId"`
	DeleteAll bool   `json:"deleteAll" mapstructure:"deleteAll"`
}

func (c *DeleteWebhooks) Name() string {
	return "jira.deleteWebhooks"
}

func (c *DeleteWebhooks) Label() string {
	return "Delete Webhooks"
}

func (c *DeleteWebhooks) Description() string {
	return "Delete webhooks registered with Jira"
}

func (c *DeleteWebhooks) Documentation() string {
	return `Delete webhooks registered via the Jira REST API.

You can either:
- Delete a specific webhook by providing its ID
- Delete all webhooks by enabling "Delete All"`
}

func (c *DeleteWebhooks) Icon() string {
	return "jira"
}

func (c *DeleteWebhooks) Color() string {
	return "blue"
}

func (c *DeleteWebhooks) ExampleOutput() map[string]any {
	return map[string]any{
		"message": "All webhooks deleted",
	}
}

func (c *DeleteWebhooks) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteWebhooks) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "webhookId",
			Label:       "Webhook ID",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "The Jira webhook ID to delete (leave empty if deleting all)",
		},
		{
			Name:        "deleteAll",
			Label:       "Delete All",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Delete all webhooks registered for this OAuth app",
		},
	}
}

func (c *DeleteWebhooks) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteWebhooks) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteWebhooks) Execute(ctx core.ExecutionContext) error {
	var config DeleteWebhooksConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	var result map[string]any

	if config.DeleteAll {
		err := client.DeleteAllWebhooks()
		if err != nil {
			return err
		}
		result = map[string]any{
			"message": "All webhooks deleted",
		}
	} else if config.WebhookID != nil {
		err := client.DeleteWebhookByID(*config.WebhookID)
		if err != nil {
			return err
		}
		result = map[string]any{
			"message":   "Webhook deleted",
			"webhookId": *config.WebhookID,
		}
	} else {
		return fmt.Errorf("either webhookId or deleteAll must be specified")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"jira.webhookDeleted",
		[]any{result},
	)
}

func (c *DeleteWebhooks) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteWebhooks) Actions() []core.Action {
	return nil
}

func (c *DeleteWebhooks) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteWebhooks) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteWebhooks) Cleanup(ctx core.SetupContext) error {
	return nil
}
