package opencost

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CostExceedsThresholdPayloadType = "opencost.costExceedsThreshold"
	PollAction                      = "poll"
	PollInterval                    = 5 * time.Minute
)

type OnCostExceedsThreshold struct{}

type OnCostExceedsThresholdConfiguration struct {
	Window    string  `json:"window" mapstructure:"window"`
	Aggregate string  `json:"aggregate" mapstructure:"aggregate"`
	Threshold float64 `json:"threshold" mapstructure:"threshold"`
}

type OnCostExceedsThresholdMetadata struct {
	LastTriggeredAt string `json:"lastTriggeredAt,omitempty" mapstructure:"lastTriggeredAt"`
}

func (t *OnCostExceedsThreshold) Name() string {
	return "opencost.onCostExceedsThreshold"
}

func (t *OnCostExceedsThreshold) Label() string {
	return "Cost Exceeds Threshold"
}

func (t *OnCostExceedsThreshold) Description() string {
	return "Trigger when cost allocation from OpenCost exceeds a threshold"
}

func (t *OnCostExceedsThreshold) Documentation() string {
	return `The Cost Exceeds Threshold trigger polls OpenCost and fires when the total cost for the configured window and aggregate dimension exceeds the specified threshold.

## Use Cases

- **Budget alerts**: Notify Slack when namespace costs exceed a daily budget
- **Cost anomaly detection**: Create tickets when unexpected cost spikes occur
- **Resource optimization**: Trigger workflows to scale down resources when costs are too high

## Configuration

- **Window**: Time window for the cost query (e.g., ` + "`1h`" + `, ` + "`24h`" + `, ` + "`7d`" + `)
- **Aggregate By**: Dimension to group costs by (e.g., ` + "`namespace`" + `, ` + "`pod`" + `, ` + "`cluster`" + `)
- **Cost Threshold**: The cost threshold in dollars. The trigger fires when total cost exceeds this value.

## Polling

This trigger polls the OpenCost API every 5 minutes to check if costs exceed the configured threshold.`
}

func (t *OnCostExceedsThreshold) Icon() string {
	return "opencost"
}

func (t *OnCostExceedsThreshold) Color() string {
	return "gray"
}

func (t *OnCostExceedsThreshold) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "window",
			Label:       "Window",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "24h",
			Placeholder: "e.g., 1h, 24h, 7d",
			Description: "Time window for the cost query",
		},
		{
			Name:     "aggregate",
			Label:    "Aggregate By",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "namespace",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: aggregateOptions,
				},
			},
			Description: "Dimension to group cost data by",
		},
		{
			Name:        "threshold",
			Label:       "Cost Threshold ($)",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., 100.00",
			Description: "Total cost threshold in dollars. Triggers when exceeded.",
		},
	}
}

func (t *OnCostExceedsThreshold) Setup(ctx core.TriggerContext) error {
	config, err := parseAndValidateThresholdConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	_ = config

	if ctx.Requests != nil {
		if err := ctx.Requests.ScheduleActionCall(PollAction, map[string]any{}, PollInterval); err != nil {
			return fmt.Errorf("failed to schedule polling action: %w", err)
		}
	}

	return nil
}

func (t *OnCostExceedsThreshold) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnCostExceedsThreshold) Actions() []core.Action {
	return []core.Action{
		{
			Name:           PollAction,
			Description:    "Poll OpenCost to check if costs exceed the threshold",
			UserAccessible: false,
		},
	}
}

func (t *OnCostExceedsThreshold) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case PollAction:
		return nil, t.poll(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (t *OnCostExceedsThreshold) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnCostExceedsThreshold) poll(ctx core.TriggerActionContext) error {
	config, err := parseAndValidateThresholdConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OpenCost client: %w", err)
	}

	response, err := client.GetAllocation(config.Window, config.Aggregate)
	if err != nil {
		return t.rescheduleAndReturn(ctx, fmt.Errorf("failed to fetch allocation: %w", err))
	}

	var totalCost float64
	exceedingItems := make([]map[string]any, 0)

	for _, dataSet := range response.Data {
		for name, entry := range dataSet {
			totalCost += entry.TotalCost
			if entry.TotalCost > config.Threshold {
				exceedingItems = append(exceedingItems, map[string]any{
					"name":      name,
					"totalCost": entry.TotalCost,
					"cpuCost":   entry.CPUCost,
					"gpuCost":   entry.GPUCost,
					"ramCost":   entry.RAMCost,
					"pvCost":    entry.PVCost,
				})
			}
		}
	}

	if totalCost > config.Threshold {
		payload := map[string]any{
			"totalCost":      totalCost,
			"threshold":      config.Threshold,
			"window":         config.Window,
			"aggregate":      config.Aggregate,
			"exceedingItems": exceedingItems,
		}

		if err := ctx.Events.Emit(CostExceedsThresholdPayloadType, payload); err != nil {
			return t.rescheduleAndReturn(ctx, fmt.Errorf("failed to emit event: %w", err))
		}
	}

	if ctx.Requests != nil {
		_ = ctx.Requests.ScheduleActionCall(PollAction, map[string]any{}, PollInterval)
	}

	return nil
}

func (t *OnCostExceedsThreshold) rescheduleAndReturn(ctx core.TriggerActionContext, err error) error {
	if ctx.Requests != nil {
		_ = ctx.Requests.ScheduleActionCall(PollAction, map[string]any{}, PollInterval)
	}
	return err
}

func parseAndValidateThresholdConfig(configuration any) (OnCostExceedsThresholdConfiguration, error) {
	config := OnCostExceedsThresholdConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return OnCostExceedsThresholdConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = sanitizeThresholdConfig(config)

	if config.Window == "" {
		return OnCostExceedsThresholdConfiguration{}, fmt.Errorf("window is required")
	}

	if config.Aggregate == "" {
		return OnCostExceedsThresholdConfiguration{}, fmt.Errorf("aggregate is required")
	}

	if config.Threshold <= 0 {
		return OnCostExceedsThresholdConfiguration{}, fmt.Errorf("threshold must be greater than zero")
	}

	return config, nil
}

func sanitizeThresholdConfig(config OnCostExceedsThresholdConfiguration) OnCostExceedsThresholdConfiguration {
	config.Window = strings.TrimSpace(config.Window)
	config.Aggregate = strings.ToLower(strings.TrimSpace(config.Aggregate))
	return config
}
