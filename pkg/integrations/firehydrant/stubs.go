package firehydrant

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

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
