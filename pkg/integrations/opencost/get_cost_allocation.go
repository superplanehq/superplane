package opencost

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetCostAllocation struct{}

type GetCostAllocationConfiguration struct {
	Window    string `json:"window" mapstructure:"window"`
	Aggregate string `json:"aggregate" mapstructure:"aggregate"`
	Filter    string `json:"filter" mapstructure:"filter"`
}

func (c *GetCostAllocation) Name() string {
	return "opencost.getCostAllocation"
}

func (c *GetCostAllocation) Label() string {
	return "Get Cost Allocation"
}

func (c *GetCostAllocation) Description() string {
	return "Fetch cost allocation data from OpenCost"
}

func (c *GetCostAllocation) Documentation() string {
	return `The Get Cost Allocation component fetches cost allocation data from the OpenCost API.

## Configuration

- **Window**: Time window for the allocation query (e.g., 1h, 1d, 7d)
- **Aggregate By**: How to group cost data (namespace, cluster, controller, service, deployment)
- **Filter** (optional): Only return allocation items matching this name

## Output

Emits ` + "`opencost.costAllocation`" + ` payloads with cost breakdown fields including cpuCost, gpuCost, ramCost, pvCost, networkCost, and totalCost.

## Use Cases

- Post a daily cost report to Slack grouped by namespace
- Feed cost data into a workflow that compares spend over time
- Generate periodic cost reports for specific services`
}

func (c *GetCostAllocation) Icon() string {
	return "opencost"
}

func (c *GetCostAllocation) Color() string {
	return "gray"
}

func (c *GetCostAllocation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCostAllocation) Configuration() []configuration.Field {
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
			Name:        "filter",
			Label:       "Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only return allocation items matching this name",
			Placeholder: "my-namespace",
		},
	}
}

func (c *GetCostAllocation) Setup(ctx core.SetupContext) error {
	config := GetCostAllocationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetCostAllocationConfiguration(config)

	if config.Window == "" {
		return fmt.Errorf("window is required")
	}

	if config.Aggregate == "" {
		return fmt.Errorf("aggregate is required")
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
		return fmt.Errorf("invalid window %q", config.Window)
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
		return fmt.Errorf("invalid aggregate %q", config.Aggregate)
	}

	return nil
}

func (c *GetCostAllocation) Execute(ctx core.ExecutionContext) error {
	config := GetCostAllocationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetCostAllocationConfiguration(config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OpenCost client: %w", err)
	}

	data, err := client.GetAllocation(config.Window, config.Aggregate)
	if err != nil {
		return fmt.Errorf("failed to fetch allocation data: %w", err)
	}

	payloads := []any{}
	for _, windowData := range data {
		for name, allocation := range windowData {
			if config.Filter != "" && !strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(config.Filter)) {
				continue
			}

			payload := buildCostAllocationPayload(allocation)
			payload["window"] = config.Window
			payload["aggregate"] = config.Aggregate
			payloads = append(payloads, payload)
		}
	}

	if len(payloads) == 0 {
		if config.Filter != "" {
			return fmt.Errorf("no allocation data found for filter %q", config.Filter)
		}
		return fmt.Errorf("no allocation data found")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CostAllocationPayloadType,
		payloads,
	)
}

func (c *GetCostAllocation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCostAllocation) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetCostAllocation) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetCostAllocation) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetCostAllocation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCostAllocation) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeGetCostAllocationConfiguration(config GetCostAllocationConfiguration) GetCostAllocationConfiguration {
	config.Window = strings.TrimSpace(config.Window)
	config.Aggregate = strings.ToLower(strings.TrimSpace(config.Aggregate))
	config.Filter = strings.TrimSpace(config.Filter)
	return config
}
