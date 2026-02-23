package opencost

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetCostAllocationPayloadType = "opencost.costAllocation"

type GetCostAllocation struct{}

type GetCostAllocationConfiguration struct {
	Window    string `json:"window" mapstructure:"window"`
	Aggregate string `json:"aggregate" mapstructure:"aggregate"`
	Step      string `json:"step" mapstructure:"step"`
}

var aggregateOptions = []configuration.FieldOption{
	{Label: "Cluster", Value: "cluster"},
	{Label: "Namespace", Value: "namespace"},
	{Label: "Controller", Value: "controller"},
	{Label: "Controller Kind", Value: "controllerKind"},
	{Label: "Node", Value: "node"},
	{Label: "Pod", Value: "pod"},
	{Label: "Container", Value: "container"},
	{Label: "Service", Value: "service"},
	{Label: "Label", Value: "label"},
	{Label: "Annotation", Value: "annotation"},
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
	return `The Get Cost Allocation component fetches Kubernetes cost allocation data from OpenCost.

## Use Cases

- **Cost reporting**: Post a cost allocation breakdown to Slack on a schedule
- **Workflow enrichment**: Use cost data in downstream workflow steps for decision-making
- **Audit trails**: Collect and store cost snapshots for historical tracking

## Configuration

- **Window**: Time window to query (e.g. ` + "`1d`" + `, ` + "`7d`" + `, ` + "`today`" + `, ` + "`lastweek`" + `)
- **Aggregate**: Dimension to group costs by (e.g. ` + "`namespace`" + `, ` + "`cluster`" + `, ` + "`pod`" + `)
- **Step**: Optional step duration for splitting the window into intervals (e.g. ` + "`1d`" + `, ` + "`1h`" + `)

## Output

Emits a ` + "`opencost.costAllocation`" + ` payload containing the allocation data grouped by the selected aggregate dimension.`
}

func (c *GetCostAllocation) Icon() string {
	return "dollar-sign"
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
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Placeholder: "1d",
			Description: "Optional step duration for splitting the window into intervals",
		},
	}
}

func decodeGetCostAllocationConfiguration(value any) (GetCostAllocationConfiguration, error) {
	spec := GetCostAllocationConfiguration{}
	if err := mapstructure.Decode(value, &spec); err != nil {
		return GetCostAllocationConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Window = strings.TrimSpace(spec.Window)
	if spec.Window == "" {
		return GetCostAllocationConfiguration{}, fmt.Errorf("window is required")
	}

	spec.Aggregate = strings.TrimSpace(spec.Aggregate)
	if spec.Aggregate == "" {
		return GetCostAllocationConfiguration{}, fmt.Errorf("aggregate is required")
	}

	spec.Step = strings.TrimSpace(spec.Step)

	return spec, nil
}

func (c *GetCostAllocation) Setup(ctx core.SetupContext) error {
	_, err := decodeGetCostAllocationConfiguration(ctx.Configuration)
	return err
}

func (c *GetCostAllocation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCostAllocation) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetCostAllocationConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("window", spec.Window)
	query.Set("aggregate", spec.Aggregate)
	if spec.Step != "" {
		query.Set("step", spec.Step)
	}

	response, err := client.GetAllocation(query)
	if err != nil {
		return fmt.Errorf("failed to fetch cost allocation: %w", err)
	}

	allocations := buildAllocationPayload(response, spec)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetCostAllocationPayloadType,
		[]any{allocations},
	)
}

func buildAllocationPayload(response *AllocationResponse, spec GetCostAllocationConfiguration) map[string]any {
	items := []map[string]any{}

	for _, data := range response.Data {
		for name, item := range data {
			items = append(items, map[string]any{
				"name":            name,
				"cpuCost":         item.CPUCost,
				"gpuCost":         item.GPUCost,
				"ramCost":         item.RAMCost,
				"pvCost":          item.PVCost,
				"networkCost":     item.NetworkCost,
				"sharedCost":      item.SharedCost,
				"externalCost":    item.ExternalCost,
				"totalCost":       item.TotalCost,
				"totalEfficiency": item.TotalEfficiency,
				"cpuEfficiency":   item.CPUEfficiency,
				"ramEfficiency":   item.RAMEfficiency,
				"start":           item.Start,
				"end":             item.End,
			})
		}
	}

	return map[string]any{
		"window":    spec.Window,
		"aggregate": spec.Aggregate,
		"items":     items,
	}
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
