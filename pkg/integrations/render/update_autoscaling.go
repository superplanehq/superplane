package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateAutoscalingPayloadType = "render.autoscaling.updated"

type UpdateAutoscaling struct{}

type UpdateAutoscalingConfiguration struct {
	Service       string `json:"service" mapstructure:"service"`
	Enabled       bool   `json:"enabled" mapstructure:"enabled"`
	MinInstances  int    `json:"minInstances" mapstructure:"minInstances"`
	MaxInstances  int    `json:"maxInstances" mapstructure:"maxInstances"`
	CPUPercent    int    `json:"cpuPercent" mapstructure:"cpuPercent"`
	MemoryPercent int    `json:"memoryPercent" mapstructure:"memoryPercent"`
}

func (c *UpdateAutoscaling) Name() string { return "render.updateAutoscaling" }

func (c *UpdateAutoscaling) Label() string { return "Update Autoscaling" }

func (c *UpdateAutoscaling) Description() string {
	return "Update Render autoscaling settings for a web service"
}

func (c *UpdateAutoscaling) Documentation() string {
	return `Update autoscaling minimum, maximum, CPU target, and memory target for a Render web service.`
}

func (c *UpdateAutoscaling) Icon() string { return "trending-up" }

func (c *UpdateAutoscaling) Color() string { return "gray" }

func (c *UpdateAutoscaling) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateAutoscaling) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceField("Render web service to update"),
		{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Required: true, Default: true},
		{Name: "minInstances", Label: "Min Instances", Type: configuration.FieldTypeNumber, Required: true, Default: 1},
		{Name: "maxInstances", Label: "Max Instances", Type: configuration.FieldTypeNumber, Required: true, Default: 3},
		{Name: "cpuPercent", Label: "Target CPU Percent", Type: configuration.FieldTypeNumber, Required: false, Default: 70},
		{Name: "memoryPercent", Label: "Target Memory Percent", Type: configuration.FieldTypeNumber, Required: false, Default: 75},
	}
}

func decodeUpdateAutoscalingConfiguration(configuration any) (UpdateAutoscalingConfiguration, error) {
	spec := UpdateAutoscalingConfiguration{Enabled: true, MinInstances: 1, MaxInstances: 3, CPUPercent: 70, MemoryPercent: 75}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return UpdateAutoscalingConfiguration{}, err
	}
	spec.Service = strings.TrimSpace(spec.Service)
	if spec.Service == "" {
		return UpdateAutoscalingConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.MinInstances < 1 {
		return UpdateAutoscalingConfiguration{}, fmt.Errorf("minInstances must be greater than 0")
	}
	if spec.MaxInstances < spec.MinInstances {
		return UpdateAutoscalingConfiguration{}, fmt.Errorf("maxInstances must be greater than or equal to minInstances")
	}
	return spec, nil
}

func (c *UpdateAutoscaling) Setup(ctx core.SetupContext) error {
	_, err := decodeUpdateAutoscalingConfiguration(ctx.Configuration)
	return err
}

func (c *UpdateAutoscaling) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeUpdateAutoscalingConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	response, err := client.UpdateAutoscaling(spec.Service, spec.Enabled, spec.MinInstances, spec.MaxInstances, spec.CPUPercent, spec.MemoryPercent)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UpdateAutoscalingPayloadType, []any{map[string]any{
		"serviceId":      spec.Service,
		"enabled":        spec.Enabled,
		"minInstances":   spec.MinInstances,
		"maxInstances":   spec.MaxInstances,
		"cpuPercent":     spec.CPUPercent,
		"memoryPercent":  spec.MemoryPercent,
		"status":         "updated",
		"renderResponse": response,
	}})
}

func (c *UpdateAutoscaling) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *UpdateAutoscaling) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *UpdateAutoscaling) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *UpdateAutoscaling) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *UpdateAutoscaling) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *UpdateAutoscaling) HandleHook(ctx core.ActionHookContext) error { return nil }
