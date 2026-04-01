package grafana

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListSilences struct{}

type ListSilencesSpec struct {
	Filter string `json:"filter" mapstructure:"filter"`
}

type ListSilencesOutput struct {
	Silences []Silence `json:"silences"`
}

func (l *ListSilences) Name() string {
	return "grafana.listSilences"
}

func (l *ListSilences) Label() string {
	return "List Silences"
}

func (l *ListSilences) Description() string {
	return "List active and pending silences from the Grafana Alertmanager"
}

func (l *ListSilences) Documentation() string {
	return `The List Silences component retrieves silences from Grafana Alertmanager.

## Use Cases

- **Audit**: Review all currently active or pending silences in your Grafana instance
- **Detect if already muted**: Check whether a specific alert or label set is already silenced before creating a duplicate
- **Workflow logic**: Branch on silence state — e.g. skip escalation if an alert is already silenced

## Configuration

- **Filter**: Optional label matcher string to filter silences (e.g. ` + "`" + `alertname=~"High.*"` + "`" + `)

## Output

Returns a list of silence objects, each including ID, state, comment, matchers, start/end times, and the author.
`
}

func (l *ListSilences) Icon() string {
	return "bell-off"
}

func (l *ListSilences) Color() string {
	return "blue"
}

func (l *ListSilences) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListSilences) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "filter",
			Label:       "Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: `Label matcher filter (e.g. alertname=~"High.*")`,
		},
	}
}

func (l *ListSilences) Setup(ctx core.SetupContext) error {
	_, err := decodeListSilencesSpec(ctx.Configuration)
	return err
}

func (l *ListSilences) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeListSilencesSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	silences, err := client.ListSilences(strings.TrimSpace(spec.Filter))
	if err != nil {
		return fmt.Errorf("error listing silences: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.silences",
		[]any{ListSilencesOutput{Silences: silences}},
	)
}

func (l *ListSilences) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListSilences) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListSilences) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListSilences) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListSilences) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (l *ListSilences) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeListSilencesSpec(config any) (ListSilencesSpec, error) {
	spec := ListSilencesSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return ListSilencesSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return ListSilencesSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

