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

type GetAlert struct{}

type GetAlertConfiguration struct {
	AlertName string `json:"alertName" mapstructure:"alertName"`
	State     string `json:"state" mapstructure:"state"`
}

func (c *GetAlert) Name() string {
	return "prometheus.getAlert"
}

func (c *GetAlert) Label() string {
	return "Get Alert"
}

func (c *GetAlert) Description() string {
	return "Get a Prometheus alert by name"
}

func (c *GetAlert) Documentation() string {
	return `The Get Alert component fetches active alerts from Prometheus (` + "`/api/v1/alerts`" + `) and returns the first alert that matches.

## Configuration

- **Alert Name**: Required ` + "`labels.alertname`" + ` value to search for (supports expressions)
- **State**: Optional filter (` + "`any`" + `, ` + "`firing`" + `, ` + "`pending`" + `, ` + "`inactive`" + `)

## Output

Emits one ` + "`prometheus.alert`" + ` payload with labels, annotations, state, and timing fields.`
}

func (c *GetAlert) Icon() string {
	return "prometheus"
}

func (c *GetAlert) Color() string {
	return "gray"
}

func (c *GetAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertName",
			Label:       "Alert Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "labels.alertname value to match",
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  AlertStateAny,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Any", Value: AlertStateAny},
						{Label: "Firing", Value: AlertStateFiring},
						{Label: "Pending", Value: AlertStatePending},
						{Label: "Inactive", Value: AlertStateInactive},
					},
				},
			},
		},
	}
}

func (c *GetAlert) Setup(ctx core.SetupContext) error {
	config := GetAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetAlertConfiguration(config)

	if config.AlertName == "" {
		return fmt.Errorf("alertName is required")
	}

	state := config.State
	if state == "" {
		return nil
	}

	switch state {
	case AlertStateAny, AlertStateFiring, AlertStatePending, AlertStateInactive:
		return nil
	default:
		return fmt.Errorf("invalid state %q", config.State)
	}
}

func (c *GetAlert) Execute(ctx core.ExecutionContext) error {
	config := GetAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetAlertConfiguration(config)

	alertName := config.AlertName
	state := config.State
	if state == "" {
		state = AlertStateAny
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	alerts, err := client.GetAlertsFromPrometheus()
	if err != nil {
		return fmt.Errorf("failed to fetch alerts: %w", err)
	}

	for _, alert := range alerts {
		if alert.Labels["alertname"] != alertName {
			continue
		}

		if state != AlertStateAny && !strings.EqualFold(alert.State, state) {
			continue
		}

		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			PrometheusAlertPayloadType,
			[]any{buildAlertPayloadFromPrometheusAlert(alert)},
		)
	}

	if state == AlertStateAny {
		return fmt.Errorf("alert %q was not found", alertName)
	}

	return fmt.Errorf("alert %q with state %q was not found", alertName, state)
}

func (c *GetAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetAlert) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetAlert) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeGetAlertConfiguration(config GetAlertConfiguration) GetAlertConfiguration {
	config.AlertName = strings.TrimSpace(config.AlertName)
	config.State = strings.ToLower(strings.TrimSpace(config.State))
	return config
}
