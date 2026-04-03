package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetDashboard struct{}

type GetDashboardSpec struct {
	DashboardUID string `json:"dashboardUid" mapstructure:"dashboardUid"`
}

func (c *GetDashboard) Name() string {
	return "grafana.getDashboard"
}

func (c *GetDashboard) Label() string {
	return "Get Dashboard"
}

func (c *GetDashboard) Description() string {
	return "Retrieve a Grafana dashboard by UID"
}

func (c *GetDashboard) Documentation() string {
	return `The Get Dashboard component fetches a Grafana dashboard using the Grafana Dashboards HTTP API.

## Use Cases

- **Dashboard inspection**: retrieve current dashboard configuration for review or downstream use
- **Workflow enrichment**: include dashboard details in notifications, tickets, or approvals
- **Panel discovery**: list panels available in a dashboard for subsequent rendering or linking

## Configuration

- **Dashboard**: The Grafana dashboard UID to retrieve

## Output

Returns the Grafana dashboard object, including title, slug, URL, folder, tags, and panel summaries.`
}

func (c *GetDashboard) Icon() string {
	return "layout-dashboard"
}

func (c *GetDashboard) Color() string {
	return "blue"
}

func (c *GetDashboard) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDashboard) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dashboardUid",
			Label:       "Dashboard",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana dashboard to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDashboard,
				},
			},
		},
	}
}

func (c *GetDashboard) Setup(ctx core.SetupContext) error {
	spec, err := decodeGetDashboardSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateGetDashboardSpec(spec); err != nil {
		return err
	}

	storeDashboardNodeMetadata(ctx, spec.DashboardUID)
	return nil
}

func (c *GetDashboard) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetDashboardSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateGetDashboardSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	dashboard, err := client.GetDashboard(spec.DashboardUID)
	if err != nil {
		return fmt.Errorf("error getting dashboard: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.dashboard",
		[]any{dashboard},
	)
}

func (c *GetDashboard) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDashboard) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDashboard) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetDashboard) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetDashboard) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetDashboard) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeGetDashboardSpec(input any) (GetDashboardSpec, error) {
	spec := GetDashboardSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return GetDashboardSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := decoder.Decode(input); err != nil {
		return GetDashboardSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.DashboardUID = strings.TrimSpace(spec.DashboardUID)
	return spec, nil
}

func validateGetDashboardSpec(spec GetDashboardSpec) error {
	if spec.DashboardUID == "" {
		return errors.New("dashboardUid is required")
	}

	return nil
}
