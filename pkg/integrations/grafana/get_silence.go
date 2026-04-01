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

type GetSilence struct{}

type GetSilenceSpec struct {
	SilenceID string `json:"silenceId" mapstructure:"silenceId"`
}

func (g *GetSilence) Name() string {
	return "grafana.getSilence"
}

func (g *GetSilence) Label() string {
	return "Get Silence"
}

func (g *GetSilence) Description() string {
	return "Retrieve a single Grafana Alertmanager silence by ID"
}

func (g *GetSilence) Documentation() string {
	return `The Get Silence component fetches the details of a single silence from Grafana Alertmanager using its ID.

## Use Cases

- **Inspect a silence**: Retrieve full details of a silence including state, comment, matchers, and times
- **Verify a silence**: Confirm a silence is still active before taking action in a workflow

## Configuration

- **Silence ID**: The unique ID of the silence to retrieve (required)

## Output

Returns the silence object including ID, state, comment, matchers, start/end times, and the author.
`
}

func (g *GetSilence) Icon() string {
	return "bell-off"
}

func (g *GetSilence) Color() string {
	return "blue"
}

func (g *GetSilence) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetSilence) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "silenceId",
			Label:       "Silence ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the silence to retrieve",
		},
	}
}

func (g *GetSilence) Setup(ctx core.SetupContext) error {
	spec, err := decodeGetSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateGetSilenceSpec(spec)
}

func (g *GetSilence) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateGetSilenceSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	silence, err := client.GetSilence(strings.TrimSpace(spec.SilenceID))
	if err != nil {
		return fmt.Errorf("error getting silence: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.silence",
		[]any{silence},
	)
}

func (g *GetSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetSilence) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeGetSilenceSpec(config any) (GetSilenceSpec, error) {
	spec := GetSilenceSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return GetSilenceSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return GetSilenceSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateGetSilenceSpec(spec GetSilenceSpec) error {
	if strings.TrimSpace(spec.SilenceID) == "" {
		return errors.New("silenceId is required")
	}
	return nil
}
