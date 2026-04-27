package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetIncident struct{}

type GetIncidentSpec struct {
	Incident string `json:"incident" mapstructure:"incident"`
}

func (g *GetIncident) Name() string {
	return "grafana.getIncident"
}

func (g *GetIncident) Label() string {
	return "Get Incident"
}

func (g *GetIncident) Description() string {
	return "Retrieve a Grafana IRM incident"
}

func (g *GetIncident) Documentation() string {
	return `The Get Incident component retrieves a single incident from Grafana IRM.

## Configuration

- **Incident**: The incident to retrieve (required)

## Output

Returns the full Grafana IRM incident object.`
}

func (g *GetIncident) Icon() string {
	return "alert-triangle"
}

func (g *GetIncident) Color() string {
	return "blue"
}

func (g *GetIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetIncident) Configuration() []configuration.Field {
	return []configuration.Field{incidentResourceField("incident", "Incident", "The incident to retrieve")}
}

func (g *GetIncident) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[GetIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateIncidentRequired(spec.Incident); err != nil {
		return err
	}
	return resolveIncidentNodeMetadata(ctx, spec.Incident)
}

func (g *GetIncident) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[GetIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateIncidentRequired(spec.Incident); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.GetIncident(spec.Incident)
	if err != nil {
		return fmt.Errorf("error getting incident: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "grafana.incident", []any{incident})
}

func (g *GetIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (g *GetIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
