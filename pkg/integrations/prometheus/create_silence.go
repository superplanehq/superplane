package prometheus

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateSilence struct{}

type CreateSilenceConfiguration struct {
	Matchers  []MatcherConfiguration `json:"matchers" mapstructure:"matchers"`
	Duration  string                 `json:"duration" mapstructure:"duration"`
	CreatedBy string                 `json:"createdBy" mapstructure:"createdBy"`
	Comment   string                 `json:"comment" mapstructure:"comment"`
}

type MatcherConfiguration struct {
	Name    string `json:"name" mapstructure:"name"`
	Value   string `json:"value" mapstructure:"value"`
	IsRegex *bool  `json:"isRegex,omitempty" mapstructure:"isRegex"`
	IsEqual *bool  `json:"isEqual,omitempty" mapstructure:"isEqual"`
}

type CreateSilenceNodeMetadata struct {
	SilenceID string `json:"silenceID"`
}

func (c *CreateSilence) Name() string {
	return "prometheus.createSilence"
}

func (c *CreateSilence) Label() string {
	return "Create Silence"
}

func (c *CreateSilence) Description() string {
	return "Create a silence in Alertmanager to suppress alerts"
}

func (c *CreateSilence) Documentation() string {
	return `The Create Silence component creates a silence in Alertmanager (` + "`POST /api/v2/silences`" + `) to suppress matching alerts.

## Configuration

- **Matchers**: Required list of matchers. Each matcher has:
  - **Name**: Label name to match
  - **Value**: Label value to match
  - **Is Regex**: Whether value is a regex pattern (default: false)
  - **Is Equal**: Whether to match equality (true) or inequality (false) (default: true)
- **Duration**: Required duration string (e.g. ` + "`1h`" + `, ` + "`30m`" + `, ` + "`2h30m`" + `)
- **Created By**: Required name of who is creating the silence
- **Comment**: Required reason for the silence

## Output

Emits one ` + "`prometheus.silence`" + ` payload with silence ID, status, matchers, timing, and creator info.`
}

func (c *CreateSilence) Icon() string {
	return "prometheus"
}

func (c *CreateSilence) Color() string {
	return "gray"
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
			Default:     `[{"name":"alertname","value":"Watchdog","isRegex":false,"isEqual":true}]`,
			Description: "List of label matchers to select alerts",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Matcher",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "isRegex",
								Label:    "Is Regex",
								Type:     configuration.FieldTypeBool,
								Required: false,
								Default:  false,
							},
							{
								Name:     "isEqual",
								Label:    "Is Equal",
								Type:     configuration.FieldTypeBool,
								Required: false,
								Default:  true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "1h",
			Description: "Duration for the silence (e.g. 1h, 30m, 2h30m)",
		},
		{
			Name:        "createdBy",
			Label:       "Created By",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "SuperPlane",
			Description: "Name of the person or system creating the silence",
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Reason for creating the silence",
		},
	}
}

func (c *CreateSilence) Setup(ctx core.SetupContext) error {
	config := CreateSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeCreateSilenceConfiguration(config)

	if len(config.Matchers) == 0 {
		return fmt.Errorf("at least one matcher is required")
	}

	for i, matcher := range config.Matchers {
		if matcher.Name == "" {
			return fmt.Errorf("matcher %d: name is required", i+1)
		}
		if matcher.Value == "" {
			return fmt.Errorf("matcher %d: value is required", i+1)
		}
	}

	if config.Duration == "" {
		return fmt.Errorf("duration is required")
	}

	if _, err := time.ParseDuration(config.Duration); err != nil {
		return fmt.Errorf("invalid duration %q: %w", config.Duration, err)
	}

	if config.CreatedBy == "" {
		return fmt.Errorf("createdBy is required")
	}

	if config.Comment == "" {
		return fmt.Errorf("comment is required")
	}

	return nil
}

func (c *CreateSilence) Execute(ctx core.ExecutionContext) error {
	config := CreateSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeCreateSilenceConfiguration(config)

	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	now := time.Now().UTC()
	startsAt := now.Format(time.RFC3339)
	endsAt := now.Add(duration).Format(time.RFC3339)

	matchers := make([]Matcher, len(config.Matchers))
	for i, m := range config.Matchers {
		isRegex := false
		if m.IsRegex != nil {
			isRegex = *m.IsRegex
		}
		isEqual := true
		if m.IsEqual != nil {
			isEqual = *m.IsEqual
		}
		matchers[i] = Matcher{
			Name:    m.Name,
			Value:   m.Value,
			IsRegex: isRegex,
			IsEqual: isEqual,
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	silenceID, err := client.CreateSilence(SilencePayload{
		Matchers:  matchers,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		CreatedBy: config.CreatedBy,
		Comment:   config.Comment,
	})
	if err != nil {
		return fmt.Errorf("failed to create silence: %w", err)
	}

	ctx.Metadata.Set(CreateSilenceNodeMetadata{SilenceID: silenceID})

	matchersData := make([]map[string]any, len(matchers))
	for i, m := range matchers {
		matchersData[i] = map[string]any{
			"name":    m.Name,
			"value":   m.Value,
			"isRegex": m.IsRegex,
			"isEqual": m.IsEqual,
		}
	}

	payload := map[string]any{
		"silenceID": silenceID,
		"status":    "active",
		"matchers":  matchersData,
		"startsAt":  startsAt,
		"endsAt":    endsAt,
		"createdBy": config.CreatedBy,
		"comment":   config.Comment,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"prometheus.silence",
		[]any{payload},
	)
}

func (c *CreateSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateSilence) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeCreateSilenceConfiguration(config CreateSilenceConfiguration) CreateSilenceConfiguration {
	for i := range config.Matchers {
		config.Matchers[i].Name = strings.TrimSpace(config.Matchers[i].Name)
		config.Matchers[i].Value = strings.TrimSpace(config.Matchers[i].Value)
	}
	config.Duration = strings.TrimSpace(config.Duration)
	config.CreatedBy = strings.TrimSpace(config.CreatedBy)
	config.Comment = strings.TrimSpace(config.Comment)
	return config
}
