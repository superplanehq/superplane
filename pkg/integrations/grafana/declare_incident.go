package grafana

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeclareIncident struct{}

func (d *DeclareIncident) Name() string {
	return "grafana.declareIncident"
}

func (d *DeclareIncident) Label() string {
	return "Declare Incident"
}

func (d *DeclareIncident) Description() string {
	return "Declare a Grafana IRM incident"
}

func (d *DeclareIncident) Documentation() string {
	return `The Declare Incident component creates a new incident in Grafana IRM.

## Use Cases

- **Automated incident declaration**: Open an incident when a deployment, alert, or workflow detects a production issue

## Configuration

- **Title**: Incident title (required)
- **Severity**: Pending, Critical, Major, or Minor (required)
- **Description**: Optional initial status update added to the incident
- **Labels**: Optional incident labels
- **Status**: Start the incident as active or resolved
- **Start Time**: Optional time when the incident began

## Output

Returns the created Grafana IRM incident.`
}

func (d *DeclareIncident) Icon() string {
	return "alert-triangle"
}

func (d *DeclareIncident) Color() string {
	return "blue"
}

func (d *DeclareIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeclareIncident) Configuration() []configuration.Field {
	return declareIncidentConfiguration(false)
}

func (d *DeclareIncident) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[declareIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	return validateDeclareIncidentSpec(spec)
}

func (d *DeclareIncident) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[declareIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeclareIncidentSpec(spec); err != nil {
		return err
	}

	isDrill := spec.IsDrill != nil && *spec.IsDrill
	return executeDeclareIncident(ctx, spec, isDrill)
}

func (d *DeclareIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeclareIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeclareIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeclareIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (d *DeclareIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeclareIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
