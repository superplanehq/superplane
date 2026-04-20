package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteHTTPSyntheticCheck struct{}

type DeleteHTTPSyntheticCheckOutput struct {
	SyntheticCheck string `json:"syntheticCheck" mapstructure:"syntheticCheck"`
	Job            string `json:"job,omitempty" mapstructure:"job"`
	Target         string `json:"target,omitempty" mapstructure:"target"`
	Deleted        bool   `json:"deleted" mapstructure:"deleted"`
}

func (d *DeleteHTTPSyntheticCheck) Name() string {
	return "grafana.deleteHttpSyntheticCheck"
}

func (d *DeleteHTTPSyntheticCheck) Label() string {
	return "Delete HTTP Synthetic Check"
}

func (d *DeleteHTTPSyntheticCheck) Description() string {
	return "Delete a Grafana HTTP synthetic check"
}

func (d *DeleteHTTPSyntheticCheck) Documentation() string {
	return `The Delete HTTP Synthetic Check component deletes an existing Grafana synthetic check.

## Configuration

- **Synthetic Check**: The synthetic check to delete

## Output

Returns a compact confirmation payload for the deleted check.`
}

func (d *DeleteHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (d *DeleteHTTPSyntheticCheck) Color() string {
	return "red"
}

func (d *DeleteHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteHTTPSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "syntheticCheck",
			Label:       "Synthetic Check",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The synthetic check to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeSyntheticCheck,
				},
			},
		},
	}
}

func (d *DeleteHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := SyntheticCheckSelectionSpec{}
	if err := decodeSyntheticCheckSpec(ctx.Configuration, &spec); err != nil {
		return err
	}
	if err := validateSyntheticCheckSelection(spec); err != nil {
		return err
	}
	return resolveSyntheticCheckNodeMetadata(ctx, spec.SyntheticCheck, nil)
}

func (d *DeleteHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := SyntheticCheckSelectionSpec{}
	if err := decodeSyntheticCheckSpec(ctx.Configuration, &spec); err != nil {
		return err
	}
	if err := validateSyntheticCheckSelection(spec); err != nil {
		return err
	}

	client, err := NewSyntheticsClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating grafana synthetics client: %w", err)
	}

	check, err := client.GetCheck(spec.SyntheticCheck)
	if err != nil {
		return fmt.Errorf("error getting synthetic check: %w", err)
	}

	if _, err := client.DeleteCheck(spec.SyntheticCheck); err != nil {
		return fmt.Errorf("error deleting synthetic check: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.syntheticCheck.deleted",
		[]any{DeleteHTTPSyntheticCheckOutput{
			SyntheticCheck: spec.SyntheticCheck,
			Job:            check.Job,
			Target:         check.Target,
			Deleted:        true,
		}},
	)
}

func (d *DeleteHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteHTTPSyntheticCheck) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteHTTPSyntheticCheck) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (d *DeleteHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
