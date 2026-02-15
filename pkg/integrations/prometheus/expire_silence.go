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

type ExpireSilence struct{}

type ExpireSilenceConfiguration struct {
	SilenceID string `json:"silenceID" mapstructure:"silenceID"`
}

func (c *ExpireSilence) Name() string {
	return "prometheus.expireSilence"
}

func (c *ExpireSilence) Label() string {
	return "Expire Silence"
}

func (c *ExpireSilence) Description() string {
	return "Expire (delete) a silence in Alertmanager"
}

func (c *ExpireSilence) Documentation() string {
	return `The Expire Silence component expires an active silence in Alertmanager, causing suppressed alerts to fire again.

## Configuration

- **Silence ID**: Required ID of the silence to expire (supports expressions)

## Output

Emits one ` + "`prometheus.silence`" + ` payload confirming the expired silence ID.`
}

func (c *ExpireSilence) Icon() string {
	return "prometheus"
}

func (c *ExpireSilence) Color() string {
	return "gray"
}

func (c *ExpireSilence) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ExpireSilence) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "silenceID",
			Label:       "Silence ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "silence-uuid",
			Description: "ID of the silence to expire",
		},
	}
}

func (c *ExpireSilence) Setup(ctx core.SetupContext) error {
	config := ExpireSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeExpireSilenceConfiguration(config)

	if config.SilenceID == "" {
		return fmt.Errorf("silenceID is required")
	}

	return nil
}

func (c *ExpireSilence) Execute(ctx core.ExecutionContext) error {
	config := ExpireSilenceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeExpireSilenceConfiguration(config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	if err := client.ExpireSilence(config.SilenceID); err != nil {
		return fmt.Errorf("failed to expire silence: %w", err)
	}

	payload := map[string]any{
		"silenceID": config.SilenceID,
		"state":     "expired",
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PrometheusSilencePayloadType,
		[]any{payload},
	)
}

func (c *ExpireSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ExpireSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ExpireSilence) Actions() []core.Action {
	return []core.Action{}
}

func (c *ExpireSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ExpireSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ExpireSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeExpireSilenceConfiguration(config ExpireSilenceConfiguration) ExpireSilenceConfiguration {
	config.SilenceID = strings.TrimSpace(config.SilenceID)
	return config
}
