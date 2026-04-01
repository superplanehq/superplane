package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateSilence struct{}

type CreateSilenceSpec struct {
	Matchers  []SilenceMatcherInput `json:"matchers" mapstructure:"matchers"`
	StartsAt  string                `json:"startsAt" mapstructure:"startsAt"`
	EndsAt    string                `json:"endsAt" mapstructure:"endsAt"`
	Comment   string                `json:"comment" mapstructure:"comment"`
	CreatedBy string                `json:"createdBy" mapstructure:"createdBy"`
}

type SilenceMatcherInput struct {
	Name    string `json:"name" mapstructure:"name"`
	Value   string `json:"value" mapstructure:"value"`
	IsRegex bool   `json:"isRegex" mapstructure:"isRegex"`
}

type CreateSilenceOutput struct {
	SilenceID string `json:"silenceId"`
}

func (c *CreateSilence) Name() string {
	return "grafana.createSilence"
}

func (c *CreateSilence) Label() string {
	return "Create Silence"
}

func (c *CreateSilence) Description() string {
	return "Create a new silence in the Grafana Alertmanager to suppress alert notifications"
}

func (c *CreateSilence) Documentation() string {
	return `The Create Silence component creates a new Alertmanager silence in Grafana, suppressing alert notifications that match the configured matchers during the specified time window.

## Use Cases

- **Deploy window**: Suppress noisy alerts during a planned maintenance or deployment window
- **Incident management**: Prevent alert storms from flooding on-call channels while an incident is being worked on
- **Testing**: Silence alerts during load tests or chaos experiments

## Configuration

- **Matchers**: One or more label matchers that identify which alerts to silence (required)
- **Starts At**: The start of the silence window (required)
- **Ends At**: The end of the silence window (required)
- **Comment**: A description of why the silence is being created (required)
- **Created By**: The author name recorded on the silence (optional, defaults to "superplane")

## Output

Returns the ID of the newly created silence.
`
}

func (c *CreateSilence) Icon() string {
	return "bell-off"
}

func (c *CreateSilence) Color() string {
	return "blue"
}

func (c *CreateSilence) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSilence) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "matchers",
			Label:       "Matchers",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Alert label matchers",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Matcher",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Label Name",
								Type:     configuration.FieldTypeString,
								Required: true,
								Description: "Label name",
							},
							{
								Name:     "value",
								Label:    "Label Value",
								Type:     configuration.FieldTypeString,
								Required: true,
								Description: "Label value",
							},
							{
								Name:        "isRegex",
								Label:       "Match as Regex",
								Type:        configuration.FieldTypeBool,
								Required:    false,
								Description: "Match as regex",
							},
						},
					},
				},
			},
		},
		{
			Name:        "startsAt",
			Label:       "Starts At",
			Type:        configuration.FieldTypeDateTime,
			Required:    true,
			Description: "Silence start time",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
		},
		{
			Name:        "endsAt",
			Label:       "Ends At",
			Type:        configuration.FieldTypeDateTime,
			Required:    true,
			Description: "Silence end time",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Reason for the silence",
		},
		{
			Name:        "createdBy",
			Label:       "Created By",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Author name for the silence record",
		},
	}
}

func (c *CreateSilence) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateCreateSilenceSpec(spec)
}

func (c *CreateSilence) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateCreateSilenceSpec(spec); err != nil {
		return err
	}

	const dtFormat = "2006-01-02T15:04"
	startsAt, err := time.Parse(dtFormat, strings.TrimSpace(spec.StartsAt))
	if err != nil {
		return fmt.Errorf("invalid startsAt %q: %w", spec.StartsAt, err)
	}

	endsAt, err := time.Parse(dtFormat, strings.TrimSpace(spec.EndsAt))
	if err != nil {
		return fmt.Errorf("invalid endsAt %q: %w", spec.EndsAt, err)
	}

	matchers := make([]SilenceMatcher, 0, len(spec.Matchers))
	for _, m := range spec.Matchers {
		matchers = append(matchers, SilenceMatcher{
			Name:    m.Name,
			Value:   m.Value,
			IsRegex: m.IsRegex,
			IsEqual: true,
		})
	}

	createdBy := strings.TrimSpace(spec.CreatedBy)
	if createdBy == "" {
		createdBy = "superplane"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	silenceID, err := client.CreateSilence(
		matchers,
		startsAt.UTC().Format(time.RFC3339),
		endsAt.UTC().Format(time.RFC3339),
		strings.TrimSpace(spec.Comment),
		createdBy,
	)
	if err != nil {
		return fmt.Errorf("error creating silence: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.silence.created",
		[]any{CreateSilenceOutput{SilenceID: silenceID}},
	)
}

func (c *CreateSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSilence) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeCreateSilenceSpec(config any) (CreateSilenceSpec, error) {
	spec := CreateSilenceSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return CreateSilenceSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return CreateSilenceSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateCreateSilenceSpec(spec CreateSilenceSpec) error {
	if len(spec.Matchers) == 0 {
		return errors.New("at least one matcher is required")
	}
	if strings.TrimSpace(spec.StartsAt) == "" {
		return errors.New("startsAt is required")
	}
	if strings.TrimSpace(spec.EndsAt) == "" {
		return errors.New("endsAt is required")
	}
	if strings.TrimSpace(spec.Comment) == "" {
		return errors.New("comment is required")
	}
	return nil
}
