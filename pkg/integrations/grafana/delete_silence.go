package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteSilence struct{}

type DeleteSilenceSpec struct {
	SilenceID string `json:"silenceId" mapstructure:"silenceId"`
}

type DeleteSilenceOutput struct {
	SilenceID string `json:"silenceId"`
	Deleted   bool   `json:"deleted"`
}

func (d *DeleteSilence) Name() string {
	return "grafana.deleteSilence"
}

func (d *DeleteSilence) Label() string {
	return "Delete Silence"
}

func (d *DeleteSilence) Description() string {
	return "Expire (delete) an existing Grafana Alertmanager silence by ID"
}

func (d *DeleteSilence) Documentation() string {
	return `The Delete Silence component expires an existing silence in Grafana Alertmanager.

## Use Cases

- **End a maintenance window early**: Remove a silence once deployment or maintenance completes ahead of schedule
- **Automated cleanup**: Expire silences created by automation after the condition they covered has resolved

## Configuration

- **Silence**: The silence to expire (required)

## Output

Returns the silence ID and a confirmation that the silence was deleted.
`
}

func (d *DeleteSilence) Icon() string {
	return "bell-off"
}

func (d *DeleteSilence) Color() string {
	return "blue"
}

func (d *DeleteSilence) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteSilence) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "silenceId",
			Label:       "Silence",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The silence to expire",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeSilence,
				},
			},
		},
	}
}

func (d *DeleteSilence) Setup(ctx core.SetupContext) error {
	spec, err := decodeDeleteSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeleteSilenceSpec(spec); err != nil {
		return err
	}

	return resolveSilenceNodeMetadata(ctx, spec.SilenceID)
}

func (d *DeleteSilence) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeleteSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeleteSilenceSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	silenceID := strings.TrimSpace(spec.SilenceID)
	if err := client.DeleteSilence(silenceID); err != nil {
		return fmt.Errorf("error deleting silence: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.silence.deleted",
		[]any{DeleteSilenceOutput{SilenceID: silenceID, Deleted: true}},
	)
}

func (d *DeleteSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteSilence) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeDeleteSilenceSpec(config any) (DeleteSilenceSpec, error) {
	spec := DeleteSilenceSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return DeleteSilenceSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return DeleteSilenceSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateDeleteSilenceSpec(spec DeleteSilenceSpec) error {
	if strings.TrimSpace(spec.SilenceID) == "" {
		return errors.New("silenceId is required")
	}
	return nil
}
