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

type RenderPanel struct{}

type RenderPanelSpec struct {
	DashboardUID string `json:"dashboard" mapstructure:"dashboard"`
	PanelID      int    `json:"panel" mapstructure:"panel"`
	Width        *int   `json:"width,omitempty" mapstructure:"width"`
	Height       *int   `json:"height,omitempty" mapstructure:"height"`
	From         string `json:"from" mapstructure:"from"`
	To           string `json:"to" mapstructure:"to"`
}

type RenderPanelOutput struct {
	URL          string `json:"url" mapstructure:"url"`
	DashboardUID string `json:"dashboard" mapstructure:"dashboard"`
	PanelID      int    `json:"panel" mapstructure:"panel"`
}

func (c *RenderPanel) Name() string {
	return "grafana.renderPanel"
}

func (c *RenderPanel) Label() string {
	return "Render Panel"
}

func (c *RenderPanel) Description() string {
	return "Construct a Grafana image render URL for a dashboard panel"
}

func (c *RenderPanel) Documentation() string {
	return `The Render Panel component constructs a Grafana image render URL for a dashboard panel using the Grafana Image Renderer.

## Use Cases

- **Incident snapshots**: attach or link a rendered panel image in tickets or notifications
- **Scheduled reports**: generate a reusable render URL for panel snapshots
- **Workflow enrichment**: pass a compact panel image URL through workflow steps

## Configuration

	- **Dashboard**: The Grafana dashboard containing the panel to render
	- **Panel**: The panel to render
	- **Width**: Image width in pixels (default 1000)
	- **Height**: Image height in pixels (default 500)
	- **From**: Optional start of the time range. Examples: ` + "`{{ now() - duration(\"1h\") }}`" + ` or ` + "`now-1h`" + `
	- **To**: Optional end of the time range. Examples: ` + "`{{ now() }}`" + ` or ` + "`now`" + `

## Output

Returns the Grafana render URL along with the dashboard UID and panel.
`
}

func (c *RenderPanel) Icon() string {
	return "layout-dashboard"
}

func (c *RenderPanel) Color() string {
	return "blue"
}

func (c *RenderPanel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RenderPanel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dashboard",
			Label:       "Dashboard",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana dashboard containing the panel to render",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDashboard,
				},
			},
		},
		{
			Name:        "panel",
			Label:       "Panel",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana panel to render",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypePanel,
					Parameters: []configuration.ParameterRef{
						{Name: "dashboard", ValueFrom: &configuration.ParameterValueFrom{Field: "dashboard"}},
					},
				},
			},
		},
		{
			Name:        "width",
			Label:       "Width",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Image width in pixels (default 1000)",
		},
		{
			Name:        "height",
			Label:       "Height",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Image height in pixels (default 500)",
		},
		{
			Name:        "from",
			Label:       "From",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Start of the time range (expression text)",
			Placeholder: `{{ now() - duration("1h") }}`,
		},
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "End of the time range (expression text)",
			Placeholder: `{{ now() }}`,
		},
	}
}

func (c *RenderPanel) Setup(ctx core.SetupContext) error {
	spec, err := decodeRenderPanelSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRenderPanelSpec(spec); err != nil {
		return err
	}

	storeDashboardNodeMetadata(ctx, spec.DashboardUID, &spec.PanelID)
	return nil
}

func (c *RenderPanel) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRenderPanelSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRenderPanelSpec(spec); err != nil {
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

	width := 1000
	if spec.Width != nil {
		width = *spec.Width
	}

	height := 500
	if spec.Height != nil {
		height = *spec.Height
	}

	from, err := resolveGrafanaTimeInput(spec.From, nil, ctx.Expressions)
	if err != nil {
		return fmt.Errorf("invalid from value %q: %w", strings.TrimSpace(spec.From), err)
	}

	to, err := resolveGrafanaTimeInput(spec.To, nil, ctx.Expressions)
	if err != nil {
		return fmt.Errorf("invalid to value %q: %w", strings.TrimSpace(spec.To), err)
	}

	renderURL := client.RenderPanelURL(
		spec.DashboardUID,
		dashboardURLPathSlug(dashboard),
		spec.PanelID,
		width,
		height,
		from,
		to,
	)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.panel.image",
		[]any{RenderPanelOutput{
			URL:          renderURL,
			DashboardUID: spec.DashboardUID,
			PanelID:      spec.PanelID,
		}},
	)
}

func (c *RenderPanel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RenderPanel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RenderPanel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RenderPanel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeRenderPanelSpec(input any) (RenderPanelSpec, error) {
	spec := RenderPanelSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return RenderPanelSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := decoder.Decode(input); err != nil {
		return RenderPanelSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.DashboardUID = strings.TrimSpace(spec.DashboardUID)
	spec.From = strings.TrimSpace(spec.From)
	spec.To = strings.TrimSpace(spec.To)
	return spec, nil
}

func validateRenderPanelSpec(spec RenderPanelSpec) error {
	if spec.DashboardUID == "" {
		return errors.New("dashboard is required")
	}
	if spec.PanelID == 0 {
		return errors.New("panel is required")
	}
	if spec.Width != nil {
		if *spec.Width <= 0 {
			return errors.New("width must be greater than 0")
		}
	}
	if spec.Height != nil {
		if *spec.Height <= 0 {
			return errors.New("height must be greater than 0")
		}
	}

	return nil
}

func (c *RenderPanel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RenderPanel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
