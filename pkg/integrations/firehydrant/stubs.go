package firehydrant

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIncident struct{}

func (t *OnIncident) Name() string {
	return "firehydrant.onIncident"
}

func (t *OnIncident) Label() string {
	return "On New Incident"
}

func (t *OnIncident) Description() string {
	return "Runs when a new incident is created in FireHydrant"
}

func (t *OnIncident) Documentation() string {
	return ""
}

func (t *OnIncident) Icon() string {
	return "flame"
}

func (t *OnIncident) Color() string {
	return "gray"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// Stub for ExampleData - will be implemented with embedded JSON
func (t *OnIncident) ExampleData() map[string]any {
	return map[string]any{}
}

var _ core.Trigger = (*OnIncident)(nil)
var _ core.Component = (*CreateIncident)(nil)

type CreateIncident struct{}

func (c *CreateIncident) Name() string {
	return "firehydrant.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in FireHydrant"
}

func (c *CreateIncident) Documentation() string {
	return ""
}

func (c *CreateIncident) Icon() string {
	return "flame"
}

func (c *CreateIncident) Color() string {
	return "gray"
}

func (c *CreateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIncident) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateIncident) ExampleOutput() map[string]any {
	return map[string]any{}
}
