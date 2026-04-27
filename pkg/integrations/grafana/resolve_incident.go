package grafana

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ResolveIncident struct{}

type ResolveIncidentSpec struct {
	Incident string `json:"incident" mapstructure:"incident"`
	Summary  string `json:"summary" mapstructure:"summary"`
}

func (r *ResolveIncident) Name() string {
	return "grafana.resolveIncident"
}

func (r *ResolveIncident) Label() string {
	return "Resolve Incident"
}

func (r *ResolveIncident) Description() string {
	return "Resolve a Grafana IRM incident"
}

func (r *ResolveIncident) Documentation() string {
	return `The Resolve Incident component marks an existing Grafana IRM incident as resolved.

## Configuration

- **Incident**: The incident to resolve (required)
- **Summary**: Optional resolution note added to the incident activity before resolving

## Output

Returns the resolved Grafana IRM incident.`
}

func (r *ResolveIncident) Icon() string {
	return "alert-triangle"
}

func (r *ResolveIncident) Color() string {
	return "blue"
}

func (r *ResolveIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (r *ResolveIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		incidentResourceField("incident", "Incident", "The incident to resolve"),
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Resolution note to add before resolving",
		},
	}
}

func (r *ResolveIncident) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[ResolveIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateIncidentRequired(spec.Incident); err != nil {
		return err
	}
	return resolveIncidentNodeMetadata(ctx, spec.Incident)
}

func (r *ResolveIncident) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[ResolveIncidentSpec](ctx.Configuration)
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

	if strings.TrimSpace(spec.Summary) != "" {
		if _, err := client.AddIncidentActivity(spec.Incident, spec.Summary); err != nil {
			return fmt.Errorf("error adding incident resolution summary: %w", err)
		}
	}

	incident, err := client.ResolveIncident(spec.Incident)
	if err != nil {
		return fmt.Errorf("error resolving incident: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "grafana.incident.resolved", []any{incident})
}

func (r *ResolveIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *ResolveIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *ResolveIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (r *ResolveIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (r *ResolveIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (r *ResolveIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
