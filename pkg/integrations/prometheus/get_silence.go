package prometheus

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetSilence struct{}

type GetSilenceConfiguration struct {
	Silence         string `json:"silence" mapstructure:"silence"`
	LegacySilenceID string `json:"silenceID,omitempty" mapstructure:"silenceID"`
}

type GetSilenceNodeMetadata struct {
	SilenceID string `json:"silenceID"`
}

type getSilencePayload struct {
	SilenceID string    `mapstructure:"silenceID"`
	Status    string    `mapstructure:"status"`
	Matchers  []Matcher `mapstructure:"matchers"`
	StartsAt  string    `mapstructure:"startsAt"`
	EndsAt    string    `mapstructure:"endsAt"`
	CreatedBy string    `mapstructure:"createdBy"`
	Comment   string    `mapstructure:"comment"`
}

func (c *GetSilence) Name() string {
	return "prometheus.getSilence"
}

func (c *GetSilence) Label() string {
	return "Get Silence"
}

func (c *GetSilence) Description() string {
	return "Get a silence from Alertmanager by ID"
}

func (c *GetSilence) Documentation() string {
	return `The Get Silence component retrieves a silence from Alertmanager (` + "`GET /api/v2/silence/{silenceID}`" + `) by its ID.

## Configuration

- **Silence**: Required silence to retrieve (supports expressions, e.g. ` + "`{{ $['Create Silence'].silenceID }}`" + `)

## Output

Emits one ` + "`prometheus.silence`" + ` payload with silence ID, status, matchers, timing, and creator info.`
}

func (c *GetSilence) Icon() string {
	return "prometheus"
}

func (c *GetSilence) Color() string {
	return "gray"
}

func (c *GetSilence) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetSilence) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "silence",
			Label:       "Silence",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Silence to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSilence,
				},
			},
		},
	}
}

func (c *GetSilence) Setup(ctx core.SetupContext) error {
	config := GetSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetSilenceConfiguration(config)

	if config.Silence == "" {
		return fmt.Errorf("silence is required")
	}

	return nil
}

func (c *GetSilence) Execute(ctx core.ExecutionContext) error {
	config := GetSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetSilenceConfiguration(config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	silence, err := client.GetSilence(config.Silence)
	if err != nil {
		return fmt.Errorf("failed to get silence: %w", err)
	}

	ctx.Metadata.Set(GetSilenceNodeMetadata{SilenceID: silence.ID})

	p := getSilencePayload{
		SilenceID: silence.ID,
		Status:    silence.Status.State,
		Matchers:  silence.Matchers,
		StartsAt:  silence.StartsAt,
		EndsAt:    silence.EndsAt,
		CreatedBy: silence.CreatedBy,
		Comment:   silence.Comment,
	}

	var payload map[string]any
	if err := mapstructure.Decode(p, &payload); err != nil {
		return fmt.Errorf("failed to decode silence payload: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"prometheus.silence",
		[]any{payload},
	)
}

func (c *GetSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetSilence) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeGetSilenceConfiguration(config GetSilenceConfiguration) GetSilenceConfiguration {
	config.Silence = strings.TrimSpace(config.Silence)
	config.LegacySilenceID = strings.TrimSpace(config.LegacySilenceID)
	if config.Silence == "" {
		config.Silence = config.LegacySilenceID
	}
	return config
}
