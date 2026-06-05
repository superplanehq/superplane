package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListDeploysPayloadType = "render.deploys.listed"

type ListDeploys struct{}

type ListDeploysConfiguration struct {
	Service  string   `json:"service" mapstructure:"service"`
	Statuses []string `json:"statuses" mapstructure:"statuses"`
	Limit    int      `json:"limit" mapstructure:"limit"`
}

func (c *ListDeploys) Name() string { return "render.listDeploys" }

func (c *ListDeploys) Label() string { return "List Deploys" }

func (c *ListDeploys) Description() string {
	return "List recent deploys for a Render service"
}

func (c *ListDeploys) Documentation() string {
	return `List recent deploys for a Render service so workflows can inspect release state or choose rollback targets.`
}

func (c *ListDeploys) Icon() string { return "history" }

func (c *ListDeploys) Color() string { return "gray" }

func (c *ListDeploys) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListDeploys) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceField("Render service to list deploys for"),
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional deploy statuses to include, for example live or build_failed",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Status",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     10,
			Description: "Maximum deploys to return",
		},
	}
}

func decodeListDeploysConfiguration(configuration any) (ListDeploysConfiguration, error) {
	spec := ListDeploysConfiguration{Limit: 10}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return ListDeploysConfiguration{}, err
	}
	spec.Service = strings.TrimSpace(spec.Service)
	spec.Statuses = cleanStringList(spec.Statuses)
	if spec.Service == "" {
		return ListDeploysConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.Limit < 1 {
		spec.Limit = 10
	}
	return spec, nil
}

func (c *ListDeploys) Setup(ctx core.SetupContext) error {
	_, err := decodeListDeploysConfiguration(ctx.Configuration)
	return err
}

func (c *ListDeploys) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeListDeploysConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploys, err := client.ListDeploys(spec.Service, spec.Statuses, spec.Limit)
	if err != nil {
		return err
	}

	items := make([]map[string]any, 0, len(deploys))
	var latestSuccessful map[string]any
	for _, deploy := range deploys {
		item := deployDataFromDeployResponse(spec.Service, deploy)
		items = append(items, item)
		if latestSuccessful == nil && deploy.Status == "live" {
			latestSuccessful = item
		}
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ListDeploysPayloadType, []any{map[string]any{
		"serviceId":        spec.Service,
		"count":            len(items),
		"deploys":          items,
		"latestSuccessful": latestSuccessful,
	}})
}

func (c *ListDeploys) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *ListDeploys) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *ListDeploys) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *ListDeploys) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *ListDeploys) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *ListDeploys) HandleHook(ctx core.ActionHookContext) error { return nil }

func serviceField(description string) configuration.Field {
	return configuration.Field{
		Name:        "service",
		Label:       "Service",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{Type: "service"},
		},
	}
}
