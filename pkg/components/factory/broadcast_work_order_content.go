package factory

import (
	"net/http"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const BroadcastWorkOrderContentComponentName = "broadcastWorkOrderContent"

func init() {
	registry.RegisterAction(BroadcastWorkOrderContentComponentName, &BroadcastWorkOrderContent{})
}

type BroadcastWorkOrderContent struct{}

type BroadcastWorkOrderContentConfiguration struct {
	OrderID  string `json:"orderId" mapstructure:"orderId"`
	Summary  string `json:"summary" mapstructure:"summary"`
	Body     string `json:"body" mapstructure:"body"`
	URL      string `json:"url" mapstructure:"url"`
	URLLabel string `json:"urlLabel" mapstructure:"urlLabel"`
}

func (c *BroadcastWorkOrderContent) Name() string {
	return BroadcastWorkOrderContentComponentName
}

func (c *BroadcastWorkOrderContent) Label() string {
	return "Broadcast Task Content"
}

func (c *BroadcastWorkOrderContent) Description() string {
	return "Post an expandable update on a task activity log"
}

func (c *BroadcastWorkOrderContent) Documentation() string {
	return `The Broadcast Task Content component posts an update on a task activity log.

The activity log shows ` + "`summary`" + ` as one line. Select that line to expand the update. The expanded block shows ` + "`body`" + ` and, when set, ` + "`url`" + `.

Use this when an automation has content a person needs from the activity log, such as a preview environment URL. The URL can also stay on the task as a link artifact. This component puts that content on the activity line itself.

Set ` + "`body`" + `, ` + "`url`" + `, or both. ` + "`url`" + ` must be an absolute http or https URL. ` + "`urlLabel`" + ` is the link text. When it is empty, the activity log shows the URL.

` + "`orderId`" + ` explicitly targets the task — it defaults to ` + "`{{ order().id }}`" + `, the task driving the current run, which only resolves when the flow was dispatched from a factory line. In a flow triggered by an external event, replace it with e.g. ` + "`{{ previous().data.workOrder.id }}`" + ` after a ` + "`findWorkOrder`" + ` step. This component can only be used in factory-owned apps.`
}

func (c *BroadcastWorkOrderContent) Icon() string {
	return "factory"
}

func (c *BroadcastWorkOrderContent) Color() string {
	return "blue"
}

func (c *BroadcastWorkOrderContent) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.contentBroadcast",
		"data": map[string]any{
			"summary":  "Preview environment is ready",
			"body":     "The preview environment is available.",
			"url":      "https://preview.example.com",
			"urlLabel": "Preview",
		},
	}
}

func (c *BroadcastWorkOrderContent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *BroadcastWorkOrderContent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Task ID",
			Description: "Task to target. Defaults to the task driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Description: "Text shown on the activity log. Select it to expand the update.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "body",
			Label:       "Content",
			Description: "Markdown shown when the update is expanded. Set content, a link URL, or both.",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "url",
			Label:       "Link URL",
			Description: "Link shown in the expanded update. Use an absolute http or https URL, for example a preview environment URL.",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "urlLabel",
			Label:       "Link label",
			Description: "Text for the link. The URL is used when this is empty.",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
	}
}

func (c *BroadcastWorkOrderContent) Execute(ctx core.ExecutionContext) error {
	config := BroadcastWorkOrderContentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	summary := strings.TrimSpace(config.Summary)
	body := strings.TrimSpace(config.Body)
	rawURL := strings.TrimSpace(config.URL)
	urlLabel := strings.TrimSpace(config.URLLabel)
	if rawURL == "" {
		urlLabel = ""
	}

	err := ctx.Factory.BroadcastWorkOrderContent(core.BroadcastWorkOrderContentParams{
		OrderID:  config.OrderID,
		Summary:  summary,
		Body:     body,
		URL:      rawURL,
		URLLabel: urlLabel,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.contentBroadcast",
		[]any{map[string]any{
			"summary":  summary,
			"body":     body,
			"url":      rawURL,
			"urlLabel": urlLabel,
		}},
	)
}

func (c *BroadcastWorkOrderContent) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *BroadcastWorkOrderContent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *BroadcastWorkOrderContent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *BroadcastWorkOrderContent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *BroadcastWorkOrderContent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *BroadcastWorkOrderContent) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
