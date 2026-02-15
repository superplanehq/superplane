package prometheus

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PrometheusSilencePayloadType = "prometheus.silence"

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
	IsRegex bool   `json:"isRegex" mapstructure:"isRegex"`
	IsEqual bool   `json:"isEqual" mapstructure:"isEqual"`
}

func (c *CreateSilence) Name() string {
	return "prometheus.createSilence"
}

func (c *CreateSilence) Label() string {
	return "Create Silence"
}

func (c *CreateSilence) Description() string {
	return "Create a silence in Alertmanager"
}

func (c *CreateSilence) Documentation() string {
	return `The Create Silence component creates a silence in Alertmanager to suppress matching alerts for a given duration.

## Configuration

- **Matchers**: Required list of label matchers for the silence
- **Duration**: Required duration for the silence (e.g., ` + "`1h`" + `, ` + "`30m`" + `, ` + "`2h30m`" + `)
- **Created By**: Required author of the silence (supports expressions)
- **Comment**: Required reason for the silence (supports expressions)

## Output

Emits one ` + "`prometheus.silence`" + ` payload with the silence ID, matchers, timing, and author fields.`
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
			Name:     "matchers",
			Label:    "Matchers",
			Type:     configuration.FieldTypeList,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Matcher",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "name",
								Label:              "Label Name",
								Type:               configuration.FieldTypeString,
								Required:           true,
								Placeholder:        "alertname",
								DisallowExpression: true,
							},
							{
								Name:        "value",
								Label:       "Label Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "HighLatency",
							},
							{
								Name:     "isRegex",
								Label:    "Regex Match",
								Type:     configuration.FieldTypeBool,
								Required: false,
								Default:  false,
							},
							{
								Name:     "isEqual",
								Label:    "Equal Match",
								Type:     configuration.FieldTypeBool,
								Required: false,
								Default:  true,
							},
						},
					},
				},
			},
			Default: []map[string]any{
				{"name": "alertname", "value": "MyAlert", "isRegex": false, "isEqual": true},
			},
			Description: "Label matchers for the silence",
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1h",
			Placeholder: "1h",
			Description: "Duration of the silence (e.g., 1h, 30m, 2h30m)",
		},
		{
			Name:        "createdBy",
			Label:       "Created By",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "SuperPlane",
			Placeholder: "SuperPlane",
			Description: "Author of the silence",
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "Silenced by SuperPlane workflow",
			Placeholder: "Reason for silencing",
			Description: "Reason for the silence",
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

	for i, m := range config.Matchers {
		if m.Name == "" {
			return fmt.Errorf("matcher %d: name is required", i+1)
		}
		if m.Value == "" {
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
		return fmt.Errorf("invalid duration %q: %w", config.Duration, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	now := time.Now().UTC()
	matchers := make([]SilenceMatcher, 0, len(config.Matchers))
	for _, m := range config.Matchers {
		matchers = append(matchers, SilenceMatcher{
			Name:    m.Name,
			Value:   m.Value,
			IsRegex: m.IsRegex,
			IsEqual: m.IsEqual,
		})
	}

	request := CreateSilenceRequest{
		Matchers:  matchers,
		StartsAt:  now.Format(time.RFC3339),
		EndsAt:    now.Add(duration).Format(time.RFC3339),
		CreatedBy: config.CreatedBy,
		Comment:   config.Comment,
	}

	silenceID, err := client.CreateSilence(request)
	if err != nil {
		return fmt.Errorf("failed to create silence: %w", err)
	}

	payload := buildSilencePayload(silenceID, request.Matchers, request.StartsAt, request.EndsAt, request.CreatedBy, request.Comment, "")
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PrometheusSilencePayloadType,
		[]any{payload},
	)
}

func (c *CreateSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
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
	config.Duration = strings.TrimSpace(config.Duration)
	config.CreatedBy = strings.TrimSpace(config.CreatedBy)
	config.Comment = strings.TrimSpace(config.Comment)

	for i := range config.Matchers {
		config.Matchers[i].Name = strings.TrimSpace(config.Matchers[i].Name)
		config.Matchers[i].Value = strings.TrimSpace(config.Matchers[i].Value)
	}

	return config
}

func buildSilencePayload(silenceID string, matchers []SilenceMatcher, startsAt, endsAt, createdBy, comment, state string) map[string]any {
	matcherMaps := make([]map[string]any, 0, len(matchers))
	for _, m := range matchers {
		matcherMaps = append(matcherMaps, map[string]any{
			"name":    m.Name,
			"value":   m.Value,
			"isRegex": m.IsRegex,
			"isEqual": m.IsEqual,
		})
	}

	payload := map[string]any{
		"silenceID": silenceID,
		"matchers":  matcherMaps,
		"startsAt":  startsAt,
		"endsAt":    endsAt,
		"createdBy": createdBy,
		"comment":   comment,
	}

	if state != "" {
		payload["state"] = state
	}

	return payload
}
