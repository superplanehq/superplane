package prometheus

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetSilence struct{}

type GetSilenceConfiguration struct {
	SilenceID string `json:"silenceID" mapstructure:"silenceID"`
}

func (c *GetSilence) Name() string {
	return "prometheus.getSilence"
}

func (c *GetSilence) Label() string {
	return "Get Silence"
}

func (c *GetSilence) Description() string {
	return "Get a silence by ID from Alertmanager"
}

func (c *GetSilence) Documentation() string {
	return `The Get Silence component retrieves a silence from Alertmanager by its ID.

## Configuration

- **Silence ID**: Required ID of the silence to retrieve (supports expressions)

## Output

Emits one ` + "`prometheus.silence`" + ` payload with the full silence details including matchers, timing, state, and author fields.`
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
			Name:        "silenceID",
			Label:       "Silence ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "silence-uuid",
			Description: "ID of the silence to retrieve",
		},
	}
}

func (c *GetSilence) Setup(ctx core.SetupContext) error {
	config := GetSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetSilenceConfiguration(config)

	if config.SilenceID == "" {
		return fmt.Errorf("silenceID is required")
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

	silence, err := client.GetSilence(config.SilenceID)
	if err != nil {
		return fmt.Errorf("failed to get silence: %w", err)
	}

	matchers := make([]SilenceMatcher, 0, len(silence.Matchers))
	for _, m := range silence.Matchers {
		matchers = append(matchers, SilenceMatcher{
			Name:    m.Name,
			Value:   m.Value,
			IsRegex: m.IsRegex,
			IsEqual: m.IsEqual,
		})
	}

	state := ""
	if silence.Status != nil {
		state = silence.Status.State
	}

	payload := buildSilencePayload(silence.ID, matchers, silence.StartsAt, silence.EndsAt, silence.CreatedBy, silence.Comment, state)
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PrometheusSilencePayloadType,
		[]any{payload},
	)
}

func (c *GetSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
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
	config.SilenceID = strings.TrimSpace(config.SilenceID)
	return config
}
