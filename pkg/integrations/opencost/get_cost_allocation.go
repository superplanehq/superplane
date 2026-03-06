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

const CostAllocationPayloadType = "opencost.allocation"

type GetCostAllocation struct{}

type GetCostAllocationConfiguration struct {
	Window     string `json:"window" mapstructure:"window"`
	Aggregate  string `json:"aggregate" mapstructure:"aggregate"`
	Step       string `json:"step" mapstructure:"step"`
	Resolution string `json:"resolution" mapstructure:"resolution"`
}

var aggregateOptions = []configuration.FieldOption{
	{Label: "Namespace", Value: "namespace"},
	{Label: "Pod", Value: "pod"},
	{Label: "Controller", Value: "controller"},
	{Label: "Service", Value: "service"},
	{Label: "Label", Value: "label"},
	{Label: "Cluster", Value: "cluster"},
	{Label: "Node", Value: "node"},
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
	return `The Get Cost Allocation component fetches cost allocation data from the OpenCost ` + "`/allocation/compute`" + ` endpoint.

## Configuration

- **Window**: Required time window for the query (e.g., ` + "`1h`" + `, ` + "`7d`" + `, ` + "`30d`" + `)
- **Aggregate**: Required dimension to group costs by (e.g., ` + "`namespace`" + `, ` + "`pod`" + `, ` + "`cluster`" + `)
- **Step**: Optional interval for data points (e.g., ` + "`1h`" + `, ` + "`1d`" + `)
- **Resolution**: Optional data granularity (e.g., ` + "`1m`" + `, ` + "`10m`" + `)

## Output

Emits one ` + "`opencost.allocation`" + ` payload containing the allocation data grouped by the specified dimension, with cost breakdowns including CPU, GPU, RAM, PV, network, and total costs.`
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
			Name:        "window",
			Label:       "Window",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1h",
			Placeholder: "e.g., 1h, 7d, 30d",
			Description: "Time window for the cost allocation query",
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
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 1h, 1d",
			Description: "Optional interval for data points",
		},
		{
			Name:        "resolution",
			Label:       "Resolution",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 1m, 10m",
			Description: "Optional data granularity",
		},
	}
}

func (c *GetCostAllocation) Setup(ctx core.SetupContext) error {
	config := GetCostAllocationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetCostAllocationConfig(config)

	if config.Window == "" {
		return fmt.Errorf("window is required")
	}

	if config.Aggregate == "" {
		return fmt.Errorf("aggregate is required")
	}

	return nil
}

func (c *GetCostAllocation) Execute(ctx core.ExecutionContext) error {
	config := GetCostAllocationConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeGetCostAllocationConfig(config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OpenCost client: %w", err)
	}

	response, err := client.GetAllocationWithParams(config.Window, config.Aggregate, config.Step, config.Resolution)
	if err != nil {
		return fmt.Errorf("failed to fetch cost allocation: %w", err)
	}

	payload := buildAllocationPayload(response, config)
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CostAllocationPayloadType,
		[]any{payload},
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

func sanitizeGetCostAllocationConfig(config GetCostAllocationConfiguration) GetCostAllocationConfiguration {
	config.Window = strings.TrimSpace(config.Window)
	config.Aggregate = strings.ToLower(strings.TrimSpace(config.Aggregate))
	config.Step = strings.TrimSpace(config.Step)
	config.Resolution = strings.TrimSpace(config.Resolution)
	return config
}

func buildAllocationPayload(response *AllocationResponse, config GetCostAllocationConfiguration) map[string]any {
	allocations := make([]map[string]any, 0)
	var totalCost float64

	for _, dataSet := range response.Data {
		for name, entry := range dataSet {
			totalCost += entry.TotalCost
			allocations = append(allocations, map[string]any{
				"name":        name,
				"cpuCost":     entry.CPUCost,
				"gpuCost":     entry.GPUCost,
				"ramCost":     entry.RAMCost,
				"pvCost":      entry.PVCost,
				"networkCost": entry.NetworkCost,
				"totalCost":   entry.TotalCost,
				"start":       entry.Start,
				"end":         entry.End,
				"minutes":     entry.Minutes,
				"properties":  entry.Properties,
			})
		}
	}

	return map[string]any{
		"window":      config.Window,
		"aggregate":   config.Aggregate,
		"allocations": allocations,
		"totalCost":   totalCost,
	}
}
