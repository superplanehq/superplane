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
	DefaultPollInterval = 5 * time.Minute

	AggregateNamespace  = "namespace"
	AggregateCluster    = "cluster"
	AggregateController = "controller"
	AggregateService    = "service"
	AggregateLabel      = "label"
	AggregateDeployment = "deployment"

	WindowOneHour   = "1h"
	WindowOneDay    = "1d"
	WindowTwoDays   = "2d"
	WindowSevenDays = "7d"
)

type OnCostExceedsThreshold struct{}

type OnCostExceedsThresholdConfiguration struct {
	Window    string  `json:"window" mapstructure:"window"`
	Aggregate string  `json:"aggregate" mapstructure:"aggregate"`
	Threshold float64 `json:"threshold" mapstructure:"threshold"`
	Filter    string  `json:"filter" mapstructure:"filter"`
}

type OnCostExceedsThresholdMetadata struct {
	Configured bool `json:"configured" mapstructure:"configured"`
}

func (t *OnCostExceedsThreshold) Name() string {
	return "opencost.onCostExceedsThreshold"
}

func (t *OnCostExceedsThreshold) Label() string {
	return "Cost Exceeds Threshold"
}

func (t *OnCostExceedsThreshold) Description() string {
	return "Trigger when OpenCost allocation exceeds a cost threshold"
}

func (t *OnCostExceedsThreshold) Documentation() string {
	return `The Cost Exceeds Threshold trigger starts a workflow execution when the total cost for any allocation item exceeds the configured threshold.

## What this trigger does

- Periodically polls the OpenCost allocation API
- Compares the total cost of each allocation item against the threshold
- Emits one event per allocation item that exceeds the threshold as ` + "`opencost.costAllocation`" + `

## Configuration

- **Window**: Time window for the allocation query (e.g., 1h, 1d, 7d)
- **Aggregate**: How to group cost data (namespace, cluster, controller, service, deployment)
- **Threshold**: Cost value (USD) that triggers the event
- **Filter** (optional): Only emit events for allocation items matching this name

## Use Cases

- Notify a Slack channel when a namespace's daily cost exceeds $50
- Create a Jira ticket when cluster costs spike above a threshold
- Trigger an automated scaling workflow when spend is too high`
}

func (t *OnCostExceedsThreshold) Icon() string {
	return "opencost"
}

func (t *OnCostExceedsThreshold) Color() string {
	return "gray"
}

func (t *OnCostExceedsThreshold) ExampleData() map[string]any {
	return exampleDataOnCostExceedsThreshold()
}

func (t *OnCostExceedsThreshold) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "window",
			Label:    "Window",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  WindowOneDay,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "1 Hour", Value: WindowOneHour},
						{Label: "1 Day", Value: WindowOneDay},
						{Label: "2 Days", Value: WindowTwoDays},
						{Label: "7 Days", Value: WindowSevenDays},
					},
				},
			},
			Description: "Time window for the cost allocation query",
		},
		{
			Name:     "aggregate",
			Label:    "Aggregate By",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AggregateNamespace,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Namespace", Value: AggregateNamespace},
						{Label: "Cluster", Value: AggregateCluster},
						{Label: "Controller", Value: AggregateController},
						{Label: "Service", Value: AggregateService},
						{Label: "Deployment", Value: AggregateDeployment},
					},
				},
			},
			Description: "How to group the cost data",
		},
		{
			Name:        "threshold",
			Label:       "Threshold (USD)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     50.0,
			Description: "Total cost threshold in USD that triggers the event",
		},
		{
			Name:        "filter",
			Label:       "Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only trigger for allocation items matching this name (e.g., a specific namespace)",
			Placeholder: "my-namespace",
		},
	}
}

func (t *OnCostExceedsThreshold) Setup(ctx core.TriggerContext) error {
	config, err := parseAndValidateThresholdConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := OnCostExceedsThresholdMetadata{}
	if ctx.Metadata != nil && ctx.Metadata.Get() != nil {
		_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	}

	if !metadata.Configured {
		metadata.Configured = true
		if ctx.Metadata != nil {
			if err := ctx.Metadata.Set(metadata); err != nil {
				return fmt.Errorf("failed to set metadata: %w", err)
			}
		}
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{
		"window":    config.Window,
		"aggregate": config.Aggregate,
		"threshold": config.Threshold,
		"filter":    config.Filter,
	}, DefaultPollInterval)
}

func (t *OnCostExceedsThreshold) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", Description: "Poll OpenCost for cost data", UserAccessible: false},
	}
}

func (t *OnCostExceedsThreshold) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "poll" {
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return nil, t.poll(ctx)
}

func (t *OnCostExceedsThreshold) poll(ctx core.TriggerActionContext) error {
	config, err := parseAndValidateThresholdConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return t.scheduleNextPoll(ctx, config)
	}

	data, err := client.GetAllocation(config.Window, config.Aggregate)
	if err != nil {
		return t.scheduleNextPoll(ctx, config)
	}

	for _, windowData := range data {
		for name, allocation := range windowData {
			if config.Filter != "" && !strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(config.Filter)) {
				continue
			}

			if allocation.TotalCost >= config.Threshold {
				payload := buildCostAllocationPayload(allocation)
				payload["threshold"] = config.Threshold
				payload["window"] = config.Window
				payload["aggregate"] = config.Aggregate

				if err := ctx.Events.Emit(CostAllocationPayloadType, payload); err != nil {
					return fmt.Errorf("failed to emit cost event: %w", err)
				}
			}
		}
	}

	return t.scheduleNextPoll(ctx, config)
}

func (t *OnCostExceedsThreshold) scheduleNextPoll(ctx core.TriggerActionContext, config OnCostExceedsThresholdConfiguration) error {
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{
		"window":    config.Window,
		"aggregate": config.Aggregate,
		"threshold": config.Threshold,
		"filter":    config.Filter,
	}, DefaultPollInterval)
}

func (t *OnCostExceedsThreshold) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnCostExceedsThreshold) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func parseAndValidateThresholdConfiguration(configuration any) (OnCostExceedsThresholdConfiguration, error) {
	config := OnCostExceedsThresholdConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return OnCostExceedsThresholdConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = sanitizeThresholdConfiguration(config)

	if err := validateThresholdConfiguration(config); err != nil {
		return OnCostExceedsThresholdConfiguration{}, err
	}

	return config, nil
}

func sanitizeThresholdConfiguration(config OnCostExceedsThresholdConfiguration) OnCostExceedsThresholdConfiguration {
	config.Window = strings.TrimSpace(config.Window)
	config.Aggregate = strings.ToLower(strings.TrimSpace(config.Aggregate))
	config.Filter = strings.TrimSpace(config.Filter)
	return config
}

func validateThresholdConfiguration(config OnCostExceedsThresholdConfiguration) error {
	if config.Window == "" {
		return fmt.Errorf("window is required")
	}

	validWindows := []string{WindowOneHour, WindowOneDay, WindowTwoDays, WindowSevenDays}
	windowValid := false
	for _, w := range validWindows {
		if config.Window == w {
			windowValid = true
			break
		}
	}
	if !windowValid {
		return fmt.Errorf("invalid window %q, expected one of: %s", config.Window, strings.Join(validWindows, ", "))
	}

	if config.Aggregate == "" {
		return fmt.Errorf("aggregate is required")
	}

	validAggregates := []string{AggregateNamespace, AggregateCluster, AggregateController, AggregateService, AggregateDeployment}
	aggregateValid := false
	for _, a := range validAggregates {
		if config.Aggregate == a {
			aggregateValid = true
			break
		}
	}
	if !aggregateValid {
		return fmt.Errorf("invalid aggregate %q, expected one of: %s", config.Aggregate, strings.Join(validAggregates, ", "))
	}

	if config.Threshold <= 0 {
		return fmt.Errorf("threshold must be greater than zero")
	}

	return nil
}

func buildCostAllocationPayload(allocation AllocationSet) map[string]any {
	return map[string]any{
		"name":        allocation.Name,
		"start":       allocation.Start,
		"end":         allocation.End,
		"cpuCost":     allocation.CPUCost,
		"gpuCost":     allocation.GPUCost,
		"ramCost":     allocation.RAMCost,
		"pvCost":      allocation.PVCost,
		"networkCost": allocation.NetworkCost,
		"totalCost":   allocation.TotalCost,
	}
}
