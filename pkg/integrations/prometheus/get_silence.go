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
	SilenceID string `json:"silenceID" mapstructure:"silenceID"`
}

type GetSilenceNodeMetadata struct {
	SilenceID string `json:"silenceID"`
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

- **Silence ID**: Required ID of the silence to retrieve (supports expressions, e.g. ` + "`{{ $['Create Silence'].silenceID }}`" + `)

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
			Name:        "silenceID",
			Label:       "Silence ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "{{ $['Create Silence'].silenceID }}",
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

	ctx.Metadata.Set(GetSilenceNodeMetadata{SilenceID: silence.ID})

	matchersData := make([]map[string]any, len(silence.Matchers))
	for i, m := range silence.Matchers {
		matchersData[i] = map[string]any{
			"name":    m.Name,
			"value":   m.Value,
			"isRegex": m.IsRegex,
			"isEqual": m.IsEqual,
		}
	}

	payload := map[string]any{
		"silenceID": silence.ID,
		"status":    silence.Status.State,
		"matchers":  matchersData,
		"startsAt":  silence.StartsAt,
		"endsAt":    silence.EndsAt,
		"createdBy": silence.CreatedBy,
		"comment":   silence.Comment,
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
	config.SilenceID = strings.TrimSpace(config.SilenceID)
	return config
}
