package grafana

import (
	"encoding/base64"
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
	DashboardUID string `json:"dashboardUid" mapstructure:"dashboardUid"`
	PanelID      int    `json:"panelId" mapstructure:"panelId"`
	Width        int    `json:"width" mapstructure:"width"`
	Height       int    `json:"height" mapstructure:"height"`
	From         string `json:"from" mapstructure:"from"`
	To           string `json:"to" mapstructure:"to"`
}

type RenderPanelOutput struct {
	ImageData    string `json:"imageData" mapstructure:"imageData"`
	DashboardUID string `json:"dashboardUid" mapstructure:"dashboardUid"`
	PanelID      int    `json:"panelId" mapstructure:"panelId"`
}

func (c *RenderPanel) Name() string {
	return "grafana.renderPanel"
}

func (c *RenderPanel) Label() string {
	return "Render Panel"
}

func (c *RenderPanel) Description() string {
	return "Render a Grafana dashboard panel as a PNG image"
}

func (c *RenderPanel) Documentation() string {
	return `The Render Panel component renders a Grafana dashboard panel as a PNG image using the Grafana Image Renderer.

## Use Cases

- **Incident snapshots**: attach a rendered panel image to incident tickets or Slack notifications
- **Scheduled reports**: capture visual metric snapshots at regular intervals
- **Workflow enrichment**: include panel images in approval or review workflows

## Configuration

- **Dashboard**: The Grafana dashboard containing the panel to render
- **Panel ID**: The ID of the panel to render
- **Width**: Image width in pixels (default 1000)
- **Height**: Image height in pixels (default 500)
- **From**: Start of the time range (e.g. now-1h)
- **To**: End of the time range (e.g. now)

## Output

Returns a base64-encoded PNG image along with the dashboard UID and panel ID.`
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
			Name:        "dashboardUid",
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
			Name:        "panelId",
			Label:       "Panel ID",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "The ID of the panel to render",
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
			Description: "Start of the time range (e.g. now-1h)",
		},
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "End of the time range (e.g. now)",
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

	storeDashboardNodeMetadata(ctx, spec.DashboardUID)
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

	width := spec.Width
	if width == 0 {
		width = 1000
	}

	height := spec.Height
	if height == 0 {
		height = 500
	}

	imageBytes, err := client.RenderPanel(spec.DashboardUID, dashboardURLPathSlug(dashboard), spec.PanelID, width, height, spec.From, spec.To)
	if err != nil {
		return fmt.Errorf("error rendering panel: %w", err)
	}

	imageData := base64.StdEncoding.EncodeToString(imageBytes)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.panel.image",
		[]any{RenderPanelOutput{
			ImageData:    imageData,
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

func (c *RenderPanel) Actions() []core.Action {
	return []core.Action{}
}

func (c *RenderPanel) HandleAction(ctx core.ActionContext) error {
	return nil
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
		return errors.New("dashboardUid is required")
	}
	if spec.PanelID == 0 {
		return errors.New("panelId is required")
	}

	return nil
}
