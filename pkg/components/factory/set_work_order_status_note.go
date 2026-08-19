package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const SetWorkOrderStatusNoteComponentName = "setWorkOrderStatusNote"

func init() {
	registry.RegisterAction(SetWorkOrderStatusNoteComponentName, &SetWorkOrderStatusNote{})
}

type SetWorkOrderStatusNote struct{}

type SetWorkOrderStatusNoteConfiguration struct {
	OrderID  string `json:"orderId" mapstructure:"orderId"`
	Headline string `json:"headline" mapstructure:"headline"`
	Body     string `json:"body" mapstructure:"body"`
	CtaLabel string `json:"ctaLabel" mapstructure:"ctaLabel"`
	CtaURL   string `json:"ctaUrl" mapstructure:"ctaUrl"`
}

func (c *SetWorkOrderStatusNote) Name() string {
	return SetWorkOrderStatusNoteComponentName
}

func (c *SetWorkOrderStatusNote) Label() string {
	return "Set Work Order Status Note"
}

func (c *SetWorkOrderStatusNote) Description() string {
	return "Announce what a waiting work order is blocked on and what resolves it"
}

func (c *SetWorkOrderStatusNote) Documentation() string {
	return `The Set Work Order Status Note component announces what a waiting work order is blocked on and what resolves it. The note shows as a "next step" panel on the work order page while the order waits — for example, a PR watcher can announce "Review the pull request: when it merges, this work order completes automatically."

A work order carries at most one note: setting a new one replaces the previous one, and any state change (close, reopen, back to draft) clears it. This keeps the note always about the current wait. The work order must be open.

- ` + "`headline`" + ` is the short instruction shown as the panel title (e.g. "Review the pull request").
- ` + "`body`" + ` is an optional markdown paragraph with the details — what happens on each outcome.
- ` + "`ctaLabel`" + ` and ` + "`ctaUrl`" + ` render an optional action button that links to where the wait resolves (e.g. the pull request). Set both or neither.

` + "`orderId`" + ` explicitly targets the work order — it defaults to ` + "`{{ order().id }}`" + `, the work order driving the current run. This component can only be used in factory-owned apps.`
}

func (c *SetWorkOrderStatusNote) Icon() string {
	return "factory"
}

func (c *SetWorkOrderStatusNote) Color() string {
	return "blue"
}

func (c *SetWorkOrderStatusNote) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.statusNoteSet",
		"data": map[string]any{
			"statusNote": map[string]any{
				"kind":     "info",
				"headline": "Review the pull request",
				"body":     "When PR #42 merges, this work order completes automatically.",
				"ctaLabel": "Review PR #42",
				"ctaUrl":   "https://github.com/acme/app/pull/42",
			},
		},
	}
}

func (c *SetWorkOrderStatusNote) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SetWorkOrderStatusNote) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Work Order ID",
			Description: "Work order to target. Defaults to the work order driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "headline",
			Label:       "Headline",
			Description: "Short instruction shown as the panel title (e.g. Review the pull request)",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "body",
			Label:       "Body",
			Description: "Optional markdown paragraph with the details — what the order waits on and what happens on each outcome",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "ctaLabel",
			Label:       "Action Label",
			Description: "Label for the action button (e.g. Review PR #42). Set together with the action URL.",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "ctaUrl",
			Label:       "Action URL",
			Description: "Absolute http(s) URL the action button opens — where the wait resolves (e.g. the pull request)",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
	}
}

func (c *SetWorkOrderStatusNote) Execute(ctx core.ExecutionContext) error {
	config := SetWorkOrderStatusNoteConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	note, err := ctx.Factory.SetWorkOrderStatusNote(core.SetWorkOrderStatusNoteParams{
		OrderID:  config.OrderID,
		Headline: config.Headline,
		Body:     config.Body,
		CtaLabel: config.CtaLabel,
		CtaURL:   config.CtaURL,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.statusNoteSet",
		[]any{map[string]any{
			"statusNote": note,
		}},
	)
}

func (c *SetWorkOrderStatusNote) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *SetWorkOrderStatusNote) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SetWorkOrderStatusNote) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SetWorkOrderStatusNote) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SetWorkOrderStatusNote) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *SetWorkOrderStatusNote) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
