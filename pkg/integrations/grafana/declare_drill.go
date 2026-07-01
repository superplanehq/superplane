package grafana

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeclareDrill struct{}

func (d *DeclareDrill) Name() string {
	return "grafana.declareDrill"
}

func (d *DeclareDrill) Label() string {
	return "Declare Drill"
}

func (d *DeclareDrill) Description() string {
	return "Declare a Grafana IRM drill incident"
}

func (d *DeclareDrill) Documentation() string {
	return `The Declare Drill component creates a new drill incident in Grafana IRM.

## Use Cases

- **Operational exercises**: Run incident response drills without affecting production metrics
- **Process validation**: Test runbooks, roles, and integrations in a safe environment

## Configuration

- **Title**: Drill title (required)
- **Severity**: Pending, Critical, Major, or Minor (required)
- **Description**: Optional initial status update added to the drill
- **Labels**: Optional drill labels
- **Status**: Start the drill as active or resolved
- **Start Time**: Optional time when the drill began

## Output

Returns the created Grafana IRM incident.`
}

func (d *DeclareDrill) Icon() string {
	return "shield-alert"
}

func (d *DeclareDrill) Color() string {
	return "blue"
}

func (d *DeclareDrill) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeclareDrill) Configuration() []configuration.Field {
	return declareIncidentConfiguration(false)
}

func (d *DeclareDrill) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[declareIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}

	return validateDeclareIncidentSpec(spec)
}

func (d *DeclareDrill) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[declareIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeclareIncidentSpec(spec); err != nil {
		return err
	}

	return executeDeclareIncident(ctx, spec, true)
}

func (d *DeclareDrill) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeclareDrill) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeclareDrill) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeclareDrill) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (d *DeclareDrill) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeclareDrill) Cleanup(ctx core.SetupContext) error {
	return nil
}
