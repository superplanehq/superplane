package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AddIncidentActivity struct{}

type AddIncidentActivitySpec struct {
	Incident string `json:"incident" mapstructure:"incident"`
	Body     string `json:"body" mapstructure:"body"`
}

func (a *AddIncidentActivity) Name() string {
	return "grafana.addIncidentActivity"
}

func (a *AddIncidentActivity) Label() string {
	return "Add Incident Activity"
}

func (a *AddIncidentActivity) Description() string {
	return "Add a note to a Grafana IRM incident activity timeline"
}

func (a *AddIncidentActivity) Documentation() string {
	return `The Add Incident Activity component posts a user note to a Grafana IRM incident timeline.

## Configuration

- **Incident**: The incident to update (required)
- **Body**: Note body (required)

## Output

Returns the created activity item.`
}

func (a *AddIncidentActivity) Icon() string {
	return "message-square"
}

func (a *AddIncidentActivity) Color() string {
	return "blue"
}

func (a *AddIncidentActivity) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (a *AddIncidentActivity) Configuration() []configuration.Field {
	return []configuration.Field{
		incidentResourceField("incident", "Incident", "The incident to update"),
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Activity note body",
		},
	}
}

func (a *AddIncidentActivity) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[AddIncidentActivitySpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateAddIncidentActivitySpec(spec); err != nil {
		return err
	}
	return resolveIncidentNodeMetadata(ctx, spec.Incident)
}

func (a *AddIncidentActivity) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[AddIncidentActivitySpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateAddIncidentActivitySpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	activity, err := client.AddIncidentActivity(spec.Incident, spec.Body)
	if err != nil {
		return fmt.Errorf("error adding incident activity: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "grafana.incident.activityAdded", []any{activity})
}

func (a *AddIncidentActivity) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (a *AddIncidentActivity) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *AddIncidentActivity) Hooks() []core.Hook {
	return []core.Hook{}
}

func (a *AddIncidentActivity) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (a *AddIncidentActivity) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *AddIncidentActivity) Cleanup(ctx core.SetupContext) error {
	return nil
}
