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

const CostAllocationPayloadType = "opencost.costAllocation"

type GetCostAllocation struct{}

type GetCostAllocationConfiguration struct {
	Window    string `json:"window" mapstructure:"window"`
	Aggregate string `json:"aggregate" mapstructure:"aggregate"`
	Step      string `json:"step" mapstructure:"step"`
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
	return `The Get Cost Allocation component queries the OpenCost ` + "`/allocation/compute`" + ` API and returns cost data.

## Configuration

- **Window**: Time window to query (e.g., ` + "`1d`" + `, ` + "`7d`" + `, ` + "`today`" + `, ` + "`lastweek`" + `)
- **Aggregate**: Field to aggregate costs by (e.g., ` + "`namespace`" + `, ` + "`cluster`" + `, ` + "`controller`" + `, ` + "`pod`" + `)
- **Step** (optional): Duration for each allocation set (e.g., ` + "`1d`" + `)

## Output

Emits one ` + "`opencost.costAllocation`" + ` payload with allocations grouped by the selected aggregate field.`
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
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "1d",
			Description: "Duration of each allocation set (optional, e.g., 1d, 1h)",
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

	data, err := client.GetAllocationWithStep(config.Window, config.Aggregate, config.Step)
	if err != nil {
		return fmt.Errorf("failed to fetch cost allocation: %w", err)
	}

	payload := buildCostAllocationPayload(data, config)

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

func sanitizeGetCostAllocationConfiguration(config GetCostAllocationConfiguration) GetCostAllocationConfiguration {
	config.Window = strings.TrimSpace(config.Window)
	config.Aggregate = strings.ToLower(strings.TrimSpace(config.Aggregate))
	config.Step = strings.TrimSpace(config.Step)
	return config
}

func buildCostAllocationPayload(data []map[string]AllocationEntry, config GetCostAllocationConfiguration) map[string]any {
	allocations := make([]map[string]any, 0)

	for _, step := range data {
		for name, entry := range step {
			allocations = append(allocations, map[string]any{
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
				"properties":      entry.Properties,
			})
		}
	}

	return map[string]any{
		"window":      config.Window,
		"aggregate":   config.Aggregate,
		"allocations": allocations,
	}
}
