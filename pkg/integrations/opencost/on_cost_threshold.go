package opencost

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CostThresholdPayloadType = "opencost.costThreshold"

	PollAction   = "poll"
	PollInterval = 5 * time.Minute
)

type OnCostThreshold struct{}

type OnCostThresholdConfiguration struct {
	Window    string  `json:"window" mapstructure:"window"`
	Aggregate string  `json:"aggregate" mapstructure:"aggregate"`
	Threshold float64 `json:"threshold" mapstructure:"threshold"`
}

type OnCostThresholdMetadata struct {
	LastEmittedKey string `json:"lastEmittedKey,omitempty" mapstructure:"lastEmittedKey"`
}

func (t *OnCostThreshold) Name() string {
	return "opencost.onCostThreshold"
}

func (t *OnCostThreshold) Label() string {
	return "Cost Exceeds Threshold"
}

func (t *OnCostThreshold) Description() string {
	return "Trigger when OpenCost allocation costs exceed a configured threshold"
}

func (t *OnCostThreshold) Documentation() string {
	return `The Cost Exceeds Threshold trigger polls OpenCost and fires when any allocation's total cost exceeds the configured threshold.

## How it works

SuperPlane polls the OpenCost ` + "`/allocation/compute`" + ` API every 5 minutes. When any allocation entry's ` + "`totalCost`" + ` exceeds the configured threshold, it emits an event with the cost data.

## Configuration

- **Window**: Time window to query (e.g., ` + "`1d`" + `, ` + "`7d`" + `, ` + "`today`" + `)
- **Aggregate By**: Field to aggregate costs by (e.g., ` + "`namespace`" + `, ` + "`cluster`" + `)
- **Threshold**: Cost threshold in dollars. Any allocation above this value triggers an event.

## Event Data

Each event includes:
- **name**: The allocation name (e.g., namespace name)
- **totalCost**: The total cost that exceeded the threshold
- **cpuCost**, **ramCost**, **gpuCost**, etc.: Cost breakdown
- **window**: The queried time window
- **threshold**: The configured threshold value`
}

func (t *OnCostThreshold) Icon() string {
	return "opencost"
}

func (t *OnCostThreshold) Color() string {
	return "gray"
}

func (t *OnCostThreshold) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "window",
			Label:       "Window",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1d",
			Placeholder: "1d",
			Description: "Time window to query (e.g., 1d, 7d, today, lastweek)",
		},
		{
			Name:     "aggregate",
			Label:    "Aggregate By",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "namespace",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Namespace", Value: "namespace"},
						{Label: "Cluster", Value: "cluster"},
						{Label: "Node", Value: "node"},
						{Label: "Controller Kind", Value: "controllerKind"},
						{Label: "Controller", Value: "controller"},
						{Label: "Service", Value: "service"},
						{Label: "Pod", Value: "pod"},
						{Label: "Container", Value: "container"},
					},
				},
			},
		},
		{
			Name:        "threshold",
			Label:       "Cost Threshold ($)",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "100",
			Placeholder: "100",
			Description: "Total cost threshold in dollars. Any allocation above this value triggers an event.",
		},
	}
}

func (t *OnCostThreshold) Setup(ctx core.TriggerContext) error {
	config, err := parseAndValidateOnCostThresholdConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	_ = config

	if ctx.Requests != nil {
		if err := schedulePoll(ctx.Requests); err != nil {
			return fmt.Errorf("failed to schedule polling action: %w", err)
		}
	}

	return nil
}

func (t *OnCostThreshold) Actions() []core.Action {
	return []core.Action{
		{
			Name:           PollAction,
			Description:    "Poll OpenCost for cost data exceeding threshold",
			UserAccessible: false,
		},
	}
}

func (t *OnCostThreshold) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case PollAction:
		return nil, t.poll(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (t *OnCostThreshold) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnCostThreshold) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnCostThreshold) poll(ctx core.TriggerActionContext) error {
	config, err := parseAndValidateOnCostThresholdConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	data, err := client.GetAllocation(config.Window, config.Aggregate)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to poll OpenCost: %v", err)
		}
		return schedulePoll(ctx.Requests)
	}

	for _, step := range data {
		for name, entry := range step {
			if entry.TotalCost <= config.Threshold {
				continue
			}

			payload := map[string]any{
				"name":            name,
				"cpuCost":         entry.CPUCost,
				"gpuCost":         entry.GPUCost,
				"ramCost":         entry.RAMCost,
				"pvCost":          entry.PVCost,
				"networkCost":     entry.NetworkCost,
				"totalCost":       entry.TotalCost,
				"cpuEfficiency":   entry.CPUEfficiency,
				"ramEfficiency":   entry.RAMEfficiency,
				"totalEfficiency": entry.TotalEfficiency,
				"start":           entry.Start,
				"end":             entry.End,
				"window":          config.Window,
				"aggregate":       config.Aggregate,
				"threshold":       config.Threshold,
				"properties":      entry.Properties,
			}

			if err := ctx.Events.Emit(CostThresholdPayloadType, payload); err != nil {
				return fmt.Errorf("failed to emit cost threshold event: %w", err)
			}
		}
	}

	return schedulePoll(ctx.Requests)
}

func parseAndValidateOnCostThresholdConfiguration(raw any) (OnCostThresholdConfiguration, error) {
	config := OnCostThresholdConfiguration{}
	if err := mapstructure.WeakDecode(raw, &config); err != nil {
		return OnCostThresholdConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Window = strings.TrimSpace(config.Window)
	if config.Window == "" {
		return OnCostThresholdConfiguration{}, fmt.Errorf("window is required")
	}

	config.Aggregate = strings.ToLower(strings.TrimSpace(config.Aggregate))
	if config.Aggregate == "" {
		return OnCostThresholdConfiguration{}, fmt.Errorf("aggregate is required")
	}

	if config.Threshold == 0 {
		thresholdStr := extractStringField(raw, "threshold")
		if thresholdStr != "" {
			parsed, err := strconv.ParseFloat(thresholdStr, 64)
			if err != nil {
				return OnCostThresholdConfiguration{}, fmt.Errorf("threshold must be a valid number")
			}
			config.Threshold = parsed
		}
	}

	if config.Threshold < 0 {
		return OnCostThresholdConfiguration{}, fmt.Errorf("threshold must be a non-negative number")
	}

	return config, nil
}

func extractStringField(raw any, field string) string {
	m, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	v, ok := m[field]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func schedulePoll(requests core.RequestContext) error {
	if requests == nil {
		return nil
	}

	return requests.ScheduleActionCall(PollAction, map[string]any{}, PollInterval)
}
