package factory

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const BroadcastTaskActivityComponentName = "broadcastTaskActivity"
const broadcastTaskActivityEventType = "workOrder.activityBroadcast"

func init() {
	registry.RegisterAction(BroadcastTaskActivityComponentName, &BroadcastTaskActivity{})
}

type BroadcastTaskActivity struct{}

type BroadcastTaskActivityConfiguration struct {
	OrderID string `json:"orderId" mapstructure:"orderId"`
	Title   string `json:"title" mapstructure:"title"`
	Body    string `json:"body" mapstructure:"body"`
	URL     string `json:"url" mapstructure:"url"`
}

func (c *BroadcastTaskActivity) Name() string {
	return BroadcastTaskActivityComponentName
}

func (c *BroadcastTaskActivity) Label() string {
	return "Broadcast Task Activity"
}

func (c *BroadcastTaskActivity) Description() string {
	return "Post expandable content on the task activity log"
}

func (c *BroadcastTaskActivity) Documentation() string {
	return `The Broadcast Task Activity component posts content on the task activity log.

The log shows ` + "`title`" + ` as one row. Click that row to open a block with ` + "`body`" + ` and ` + "`url`" + `. Use this when an automation creates something people need to open, such as a preview environment URL. The URL can also stay on the task as an artifact.

Set ` + "`body`" + `, ` + "`url`" + `, or both. ` + "`url`" + ` must be an absolute http or https URL. ` + "`body`" + ` is Markdown.

` + "`orderId`" + ` targets the task. It defaults to ` + "`{{ order().id }}`" + `, the task that drives the current run. That value resolves only when a factory line dispatched the flow. In a flow that an external event starts, replace it with an id from an earlier step, for example ` + "`{{ previous().data.workOrder.id }}`" + ` after Find Task. This component can only be used in factory-owned apps.`
}

func (c *BroadcastTaskActivity) Icon() string {
	return "factory"
}

func (c *BroadcastTaskActivity) Color() string {
	return "blue"
}

func (c *BroadcastTaskActivity) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      broadcastTaskActivityEventType,
		"data": map[string]any{
			"title": "Preview environment ready",
			"body":  "Open the preview for this change.",
			"url":   "https://preview.example.com/pr/42",
		},
	}
}

func (c *BroadcastTaskActivity) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *BroadcastTaskActivity) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Task ID",
			Description: "Task to target. Defaults to the task that drives the current run. That value resolves only when a factory line dispatched this flow. Replace it with an id from an earlier step otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "title",
			Label:       "Title",
			Description: "Text on the activity row. Click the row to open the content.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "body",
			Label:       "Content",
			Description: "Markdown shown when the activity row is open. Set content, URL, or both.",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "url",
			Label:       "URL",
			Description: "Link shown when the activity row is open. Use an absolute http or https URL, such as a preview environment URL.",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
	}
}

func (c *BroadcastTaskActivity) ValidateNodeConfiguration(config map[string]any) error {
	decoded := BroadcastTaskActivityConfiguration{}
	if err := mapstructure.Decode(config, &decoded); err != nil {
		return err
	}
	return validateBroadcastTaskActivity(decoded)
}

func (c *BroadcastTaskActivity) Execute(ctx core.ExecutionContext) error {
	config := BroadcastTaskActivityConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if err := validateBroadcastTaskActivity(config); err != nil {
		return err
	}

	err := ctx.Factory.BroadcastWorkOrderActivity(core.BroadcastWorkOrderActivityParams{
		OrderID: config.OrderID,
		Title:   config.Title,
		Body:    config.Body,
		URL:     config.URL,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		broadcastTaskActivityEventType,
		[]any{map[string]any{
			"title": strings.TrimSpace(config.Title),
			"body":  strings.TrimSpace(config.Body),
			"url":   strings.TrimSpace(config.URL),
		}},
	)
}

func validateBroadcastTaskActivity(config BroadcastTaskActivityConfiguration) error {
	if strings.TrimSpace(config.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(config.Body) == "" && strings.TrimSpace(config.URL) == "" {
		return fmt.Errorf("content or URL is required")
	}
	if rawURL := strings.TrimSpace(config.URL); rawURL != "" && !isBroadcastHTTPURL(rawURL) {
		return fmt.Errorf("URL must be an absolute http or https URL")
	}
	return nil
}

func isBroadcastHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") && parsed.Host != ""
}

func (c *BroadcastTaskActivity) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *BroadcastTaskActivity) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *BroadcastTaskActivity) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *BroadcastTaskActivity) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *BroadcastTaskActivity) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *BroadcastTaskActivity) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
