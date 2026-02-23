package opencost

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CostExceedsThresholdPayloadType = "opencost.costExceedsThreshold"

const (
	CostExceedsThresholdPollAction   = "poll"
	CostExceedsThresholdPollInterval = 5 * time.Minute
)

type CostExceedsThreshold struct{}

type CostExceedsThresholdConfiguration struct {
	Window    string  `json:"window" mapstructure:"window"`
	Aggregate string  `json:"aggregate" mapstructure:"aggregate"`
	Threshold float64 `json:"threshold" mapstructure:"threshold"`
}

func (t *CostExceedsThreshold) Name() string {
	return "opencost.costExceedsThreshold"
}

func (t *CostExceedsThreshold) Label() string {
	return "Cost Exceeds Threshold"
}

func (t *CostExceedsThreshold) Description() string {
	return "Trigger when OpenCost allocation cost exceeds a configured threshold"
}

func (t *CostExceedsThreshold) Documentation() string {
	return `The Cost Exceeds Threshold trigger fires when any allocation item from OpenCost exceeds the configured cost threshold.

## Use Cases

- **Budget alerts**: Send a Slack notification when a namespace exceeds its cost budget
- **Incident creation**: Automatically create a ticket when cluster spend spikes
- **Cost governance**: Trigger approval workflows when spend crosses a limit

## Configuration

- **Window**: Time window to query (e.g. ` + "`1d`" + `, ` + "`7d`" + `, ` + "`today`" + `)
- **Aggregate**: Dimension to group costs by (e.g. ` + "`namespace`" + `, ` + "`cluster`" + `)
- **Threshold**: Cost threshold in USD. The trigger fires when any allocation item's total cost exceeds this value.

## How It Works

SuperPlane polls the OpenCost allocation API every 5 minutes. When any allocation item's total cost exceeds the configured threshold, the trigger emits an event containing the item details and the threshold that was exceeded.`
}

func (t *CostExceedsThreshold) Icon() string {
	return "dollar-sign"
}

func (t *CostExceedsThreshold) Color() string {
	return "gray"
}

func (t *CostExceedsThreshold) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "window",
			Label:       "Window",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1d",
			Placeholder: "1d",
			Description: "Time window to query (e.g. 1d, 7d, today, lastweek)",
		},
		{
			Name:        "aggregate",
			Label:       "Aggregate",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "namespace",
			Description: "Dimension to group costs by",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: aggregateOptions,
				},
			},
		},
		{
			Name:        "threshold",
			Label:       "Threshold (USD)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     100,
			Description: "Cost threshold in USD. Trigger fires when any item's total cost exceeds this value.",
		},
	}
}

func decodeCostExceedsThresholdConfiguration(value any) (CostExceedsThresholdConfiguration, error) {
	config := CostExceedsThresholdConfiguration{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return CostExceedsThresholdConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Window = strings.TrimSpace(config.Window)
	if config.Window == "" {
		return CostExceedsThresholdConfiguration{}, fmt.Errorf("window is required")
	}

	config.Aggregate = strings.TrimSpace(config.Aggregate)
	if config.Aggregate == "" {
		return CostExceedsThresholdConfiguration{}, fmt.Errorf("aggregate is required")
	}

	if config.Threshold <= 0 {
		return CostExceedsThresholdConfiguration{}, fmt.Errorf("threshold must be greater than 0")
	}

	return config, nil
}

func (t *CostExceedsThreshold) ExampleData() map[string]any {
	return exampleDataCostExceedsThresholdParsed()
}

func (t *CostExceedsThreshold) Setup(ctx core.TriggerContext) error {
	_, err := decodeCostExceedsThresholdConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if ctx.Requests != nil {
		if err := scheduleCostExceedsThresholdPoll(ctx.Requests); err != nil {
			return fmt.Errorf("failed to schedule polling action: %w", err)
		}
	}

	return nil
}

func (t *CostExceedsThreshold) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *CostExceedsThreshold) Actions() []core.Action {
	return []core.Action{
		{
			Name:           CostExceedsThresholdPollAction,
			Description:    "Poll OpenCost allocation data and check against threshold",
			UserAccessible: false,
		},
	}
}

func (t *CostExceedsThreshold) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case CostExceedsThresholdPollAction:
		return nil, t.poll(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (t *CostExceedsThreshold) poll(ctx core.TriggerActionContext) error {
	config, err := decodeCostExceedsThresholdConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to create OpenCost client: %v", err)
		}
		return scheduleCostExceedsThresholdPoll(ctx.Requests)
	}

	query := url.Values{}
	query.Set("window", config.Window)
	query.Set("aggregate", config.Aggregate)

	response, err := client.GetAllocation(query)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to fetch OpenCost allocation data: %v", err)
		}
		return scheduleCostExceedsThresholdPoll(ctx.Requests)
	}

	for _, data := range response.Data {
		for name, item := range data {
			if item.TotalCost > config.Threshold {
				payload := map[string]any{
					"name":        name,
					"totalCost":   item.TotalCost,
					"threshold":   config.Threshold,
					"window":      config.Window,
					"aggregate":   config.Aggregate,
					"cpuCost":     item.CPUCost,
					"gpuCost":     item.GPUCost,
					"ramCost":     item.RAMCost,
					"pvCost":      item.PVCost,
					"networkCost": item.NetworkCost,
					"start":       item.Start,
					"end":         item.End,
				}

				if err := ctx.Events.Emit(CostExceedsThresholdPayloadType, payload); err != nil {
					return fmt.Errorf("failed to emit event: %w", err)
				}
			}
		}
	}

	return scheduleCostExceedsThresholdPoll(ctx.Requests)
}

func (t *CostExceedsThreshold) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func scheduleCostExceedsThresholdPoll(requests core.RequestContext) error {
	if requests == nil {
		return nil
	}

	return requests.ScheduleActionCall(
		CostExceedsThresholdPollAction,
		map[string]any{},
		CostExceedsThresholdPollInterval,
	)
}
