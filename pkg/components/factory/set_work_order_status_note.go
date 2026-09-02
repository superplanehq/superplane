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
	OrderID             string `json:"orderId" mapstructure:"orderId"`
	NoteKey             string `json:"noteKey" mapstructure:"noteKey"`
	Headline            string `json:"headline" mapstructure:"headline"`
	Body                string `json:"body" mapstructure:"body"`
	CtaLabel            string `json:"ctaLabel" mapstructure:"ctaLabel"`
	CtaURL              string `json:"ctaUrl" mapstructure:"ctaUrl"`
	ShowOnlyWhenWaiting bool   `json:"showOnlyWhenWaiting" mapstructure:"showOnlyWhenWaiting"`
}

func (c *SetWorkOrderStatusNote) Name() string {
	return SetWorkOrderStatusNoteComponentName
}

func (c *SetWorkOrderStatusNote) Label() string {
	return "Set Task Status Note"
}

func (c *SetWorkOrderStatusNote) Description() string {
	return "Announce what a waiting task is blocked on and what resolves it"
}

func (c *SetWorkOrderStatusNote) Documentation() string {
	return `The Set Task Status Note component announces what a waiting task is blocked on and what resolves it. The note shows as a "next step" panel on the task page — for example, a PR watcher can announce "Review the pull request: when it merges, this task completes automatically."

Each note is identified by its ` + "`noteKey`" + ` (for example ` + "`pr-closure`" + `). The first set creates the note. A later set with the same key updates that note in place. A different key sits beside it, so one task can carry several waits at once (a PR review and a later decision prompt). Any state change (close, reopen, back to draft) clears every note. The task must be open.

- ` + "`headline`" + ` is the short instruction shown as the panel title (e.g. "Review the pull request").
- ` + "`body`" + ` is an optional markdown paragraph with the details — what happens on each outcome.
- ` + "`ctaLabel`" + ` and ` + "`ctaUrl`" + ` render an optional action button that links to where the wait resolves (e.g. the pull request). Set both or neither.
- ` + "`showOnlyWhenWaiting`" + ` hides the note while a line is running. The default is off, so the note stays visible during a run.

` + "`orderId`" + ` explicitly targets the task — it defaults to ` + "`{{ order().id }}`" + `, the task driving the current run. This component can only be used in factory-owned apps.

Setting a note also emails the task's owners and creator (excluding whoever triggered the run), so they know it's waiting on their review — every set sends a fresh email, including one that just updates an existing note's body. Recipients can turn this off in their notification settings without affecting other task emails.`
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
				"key":      "pr-closure",
				"kind":     "info",
				"headline": "Review the pull request",
				"body":     "When PR #42 merges, this task completes automatically.",
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
			Label:       "Task ID",
			Description: "Task to target. Defaults to the task driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "noteKey",
			Label:       "Note Key",
			Description: "Stable identifier for this note on the task (e.g. pr-closure). Sets with the same key update the same note.",
			Type:        configuration.FieldTypeString,
			Required:    true,
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
			Description: "Optional markdown paragraph with the details — what the task waits on and what happens on each outcome",
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
		{
			Name:        "showOnlyWhenWaiting",
			Label:       "Show only while waiting",
			Description: "When this option is on, SuperPlane hides the note while a line is running.",
			Type:        configuration.FieldTypeBool,
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
		OrderID:             config.OrderID,
		NoteKey:             config.NoteKey,
		Headline:            config.Headline,
		Body:                config.Body,
		CtaLabel:            config.CtaLabel,
		CtaURL:              config.CtaURL,
		ShowOnlyWhenWaiting: config.ShowOnlyWhenWaiting,
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
