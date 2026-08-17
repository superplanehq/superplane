package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

// UpdateWorkOrderArtifact
//
// State / Merged / Draft accept expressions, so a flow can wire them
// straight to a `github.onPullRequest` payload without an if-node.
// After template resolution, values may arrive as bool, string, or
// number; `any` avoids brittle assumptions.

const UpdateWorkOrderArtifactComponentName = "updateWorkOrderArtifact"

func init() {
	registry.RegisterAction(UpdateWorkOrderArtifactComponentName, &UpdateWorkOrderArtifact{})
}

type UpdateWorkOrderArtifact struct{}

type UpdateWorkOrderArtifactConfiguration struct {
	OrderID     string `json:"orderId" mapstructure:"orderId"`
	ArtifactKey string `json:"artifactKey" mapstructure:"artifactKey"`
	State       any    `json:"state,omitempty" mapstructure:"state,omitempty"`
	Merged      any    `json:"merged,omitempty" mapstructure:"merged,omitempty"`
	Draft       any    `json:"draft,omitempty" mapstructure:"draft,omitempty"`
	Title       string `json:"title" mapstructure:"title"`
}

func (c *UpdateWorkOrderArtifact) Name() string {
	return UpdateWorkOrderArtifactComponentName
}

func (c *UpdateWorkOrderArtifact) Label() string {
	return "Update Work Order Artifact"
}

func (c *UpdateWorkOrderArtifact) Description() string {
	return "Update a pull request artifact already attached to a work order (e.g. its state)"
}

func (c *UpdateWorkOrderArtifact) Documentation() string {
	return `The Update Work Order Artifact component keeps an already-attached artifact's data current — today, a pull request's ` + "`state`" + `. It resolves the artifact by ` + "`artifactKey`" + `, the same key it was given at attach time via ` + "`addWorkOrderArtifact`" + `'s ` + "`artifactKey`" + ` field (typically the pull request's URL), and merges the fields set here into its existing data. Fields left blank are untouched — sending only ` + "`state`" + ` doesn't clobber ` + "`title`" + ` or ` + "`number`" + `.

This does not add a new timeline entry: the work order's "attached" line and the sidebar artifact list both update in place so the chip's icon/color track the pull request's current state without spamming the timeline on every open → draft → merged flip.

Typical wiring to stay in sync with GitHub: a ` + "`github.onPullRequest`" + ` trigger (actions: opened, ready_for_review, converted_to_draft, closed) → ` + "`findWorkOrder`" + ` (` + "`by: artifactKey`" + `, ` + "`artifactKey: {{ event.data.pull_request.html_url }}`" + `) → this component.

The ` + "`state`" + `, ` + "`merged`" + `, and ` + "`draft`" + ` fields all accept expressions, so you can pass the webhook payload through as-is instead of writing an if-node per branch. Set ` + "`state: \"{{ event.data.pull_request.state }}\"`" + ` and ` + "`merged: \"{{ event.data.pull_request.merged }}\"`" + ` (plus optional ` + "`draft: \"{{ event.data.pull_request.draft }}\"`" + `); the component folds a GitHub-shaped payload (` + "`state: \"closed\"`" + ` + ` + "`merged: true`" + `) into SuperPlane's ` + "`state: \"merged\"`" + ` before it hits the artifact, so the chip flips to purple as soon as the merge event lands.

` + "`orderId`" + ` explicitly targets the work order — it defaults to ` + "`{{ order().id }}`" + `, the work order driving the current run, which only resolves when the flow was dispatched from a factory line. In a flow triggered by an external event such as ` + "`github.onPullRequest`" + `, replace it with ` + "`{{ previous().data.workOrder.id }}`" + ` after a ` + "`findWorkOrder`" + ` step. This component can only be used in factory-owned apps.`
}

func (c *UpdateWorkOrderArtifact) Icon() string {
	return "factory"
}

func (c *UpdateWorkOrderArtifact) Color() string {
	return "blue"
}

func (c *UpdateWorkOrderArtifact) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.artifactUpdated",
		"data": map[string]any{
			"artifact": map[string]any{
				"id":   "art-123",
				"type": "pr",
				"data": map[string]any{
					"url":    "https://github.com/example/repo/pull/42",
					"number": "42",
					"title":  "Draft implementation",
					"state":  "merged",
				},
			},
		},
	}
}

func (c *UpdateWorkOrderArtifact) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateWorkOrderArtifact) Configuration() []configuration.Field {
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
			Name:        "artifactKey",
			Label:       "Artifact Key",
			Description: "The key the artifact was attached with (e.g. the pull request's URL) — see addWorkOrderArtifact's artifactKey field.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "state",
			Label:       "State",
			Description: "New pull request state — drives the artifact chip's icon/color. One of `open`, `draft`, `closed`, `merged`. Accepts an expression (e.g. `{{ root().data.pull_request.state }}`). Leave blank to update only the other fields.",
			Placeholder: "open",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "merged",
			Label:       "Merged",
			Description: "GitHub-native `pull_request.merged` flag. When truthy, the artifact is stored as merged even if `state` is blank or says `open`. Accepts an expression (e.g. `{{ root().data.pull_request.merged }}`).",
			Placeholder: "{{ root().data.pull_request.merged }}",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "draft",
			Label:       "Draft",
			Description: "GitHub-native `pull_request.draft` flag. When truthy and the PR is not merged, the artifact is stored as draft. Accepts an expression (e.g. `{{ root().data.pull_request.draft }}`).",
			Placeholder: "{{ root().data.pull_request.draft }}",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "title",
			Label:       "Title",
			Description: "New title, in case the pull request was retitled. Leave unset to keep the existing title.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
	}
}

func (c *UpdateWorkOrderArtifact) Execute(ctx core.ExecutionContext) error {
	config := UpdateWorkOrderArtifactConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	data := map[string]any{}
	if state := resolvePrArtifactState(config.State, config.Merged, config.Draft); state != "" {
		data["state"] = state
	}
	if config.Title != "" {
		data["title"] = config.Title
	}

	artifact, err := ctx.Factory.UpdateWorkOrderArtifact(core.UpdateWorkOrderArtifactParams{
		OrderID: config.OrderID,
		Key:     config.ArtifactKey,
		Data:    data,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.artifactUpdated",
		[]any{map[string]any{
			"artifact": artifact,
		}},
	)
}

func (c *UpdateWorkOrderArtifact) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateWorkOrderArtifact) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateWorkOrderArtifact) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateWorkOrderArtifact) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateWorkOrderArtifact) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateWorkOrderArtifact) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
