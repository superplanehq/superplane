package factory

import (
	"errors"
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const FindWorkOrderComponentName = "findWorkOrder"

func init() {
	registry.RegisterAction(FindWorkOrderComponentName, &FindWorkOrder{})
}

type FindWorkOrder struct{}

type FindWorkOrderConfiguration struct {
	By          string `json:"by" mapstructure:"by"`
	OrderID     string `json:"orderId" mapstructure:"orderId"`
	ArtifactKey string `json:"artifactKey" mapstructure:"artifactKey"`
}

func (c *FindWorkOrder) Name() string {
	return FindWorkOrderComponentName
}

func (c *FindWorkOrder) Label() string {
	return "Find Work Order"
}

func (c *FindWorkOrder) Description() string {
	return "Look up a work order by id or by an artifact's key"
}

func (c *FindWorkOrder) Documentation() string {
	return `The Find Work Order component resolves a work order without requiring the current run to be attached to one — the missing piece for flows that don't start on a factory line, e.g. a ` + "`github.onPullRequest`" + ` merged event that needs to close the order a pull request belongs to.

Lookup modes:

- **Work Order ID** (` + "`by: id`" + `): resolves ` + "`orderId`" + ` directly.
- **Artifact Key** (` + "`by: artifactKey`" + `): resolves the work order that owns the artifact tagged with ` + "`artifactKey`" + ` (set via ` + "`addWorkOrderArtifact`" + `'s ` + "`artifactKey`" + ` field — e.g. the pull request's URL).

On a match, emits ` + "`workOrder.found`" + ` with ` + "`{ workOrder }`" + ` on the default channel, so downstream components can target it with ` + "`orderId: {{ previous().data.workOrder.id }}`" + `. When nothing matches, the component passes silently (no emit, no error) — a PR merge unrelated to any tracked order shouldn't red a flow. Misconfiguration (invalid id, wrong factory, etc.) still fails the run. This component can only be used in factory-owned apps.`
}

func (c *FindWorkOrder) Icon() string {
	return "factory"
}

func (c *FindWorkOrder) Color() string {
	return "blue"
}

func (c *FindWorkOrder) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.found",
		"data": map[string]any{
			"workOrder": map[string]any{
				"id":    "wo-123",
				"title": "Work Order 1",
				"state": "open",
			},
		},
	}
}

func (c *FindWorkOrder) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *FindWorkOrder) Configuration() []configuration.Field {
	byID := []configuration.VisibilityCondition{{Field: "by", Values: []string{"id"}}}
	byArtifactKey := []configuration.VisibilityCondition{{Field: "by", Values: []string{"artifactKey"}}}

	return []configuration.Field{
		{
			Name:        "by",
			Label:       "Find by",
			Description: "How to look the work order up",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "id",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Work Order ID", Value: "id"},
						{Label: "Artifact Key", Value: "artifactKey"},
					},
				},
			},
		},
		{
			Name:                 "orderId",
			Label:                "Work Order ID",
			Description:          "The id of the work order to find",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: byID,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "by", Values: []string{"id"}},
			},
		},
		{
			Name:                 "artifactKey",
			Label:                "Artifact Key",
			Description:          "The key of an artifact attached to the work order (e.g. a pull request's URL)",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: byArtifactKey,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "by", Values: []string{"artifactKey"}},
			},
		},
	}
}

func (c *FindWorkOrder) Execute(ctx core.ExecutionContext) error {
	config := FindWorkOrderConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	order, err := ctx.Factory.FindWorkOrder(core.FindWorkOrderParams{
		By:          config.By,
		OrderID:     config.OrderID,
		ArtifactKey: config.ArtifactKey,
	})
	if err != nil {
		// A PR merge (or similar) unrelated to any tracked order is an
		// expected outcome, not a failure — let the run pass quietly so
		// it doesn't need its own error-handling branch.
		if errors.Is(err, core.ErrWorkOrderNotFound) {
			return ctx.ExecutionState.Pass()
		}
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.found",
		[]any{map[string]any{
			"workOrder": order,
		}},
	)
}

func (c *FindWorkOrder) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *FindWorkOrder) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *FindWorkOrder) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *FindWorkOrder) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *FindWorkOrder) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *FindWorkOrder) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *FindWorkOrder) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
