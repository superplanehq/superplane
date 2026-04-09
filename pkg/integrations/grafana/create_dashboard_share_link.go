package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateDashboardShareLink struct{}

type CreateDashboardShareLinkSpec struct {
	DashboardUID string `json:"dashboardUid" mapstructure:"dashboardUid"`
	PanelID      *int   `json:"panelId,omitempty" mapstructure:"panelId"`
	From         string `json:"from" mapstructure:"from"`
	To           string `json:"to" mapstructure:"to"`
}

type CreateDashboardShareLinkOutput struct {
	URL            string `json:"url" mapstructure:"url"`
	DashboardTitle string `json:"dashboardTitle" mapstructure:"dashboardTitle"`
	DashboardUID   string `json:"dashboardUid" mapstructure:"dashboardUid"`
}

func (c *CreateDashboardShareLink) Name() string {
	return "grafana.createDashboardShareLink"
}

func (c *CreateDashboardShareLink) Label() string {
	return "Create Dashboard Share Link"
}

func (c *CreateDashboardShareLink) Description() string {
	return "Construct a shareable URL for a Grafana dashboard or panel"
}

func (c *CreateDashboardShareLink) Documentation() string {
	return `The Create Dashboard Share Link component constructs a shareable URL for a Grafana dashboard or panel.

## Use Cases

- **Incident response**: include a direct link to the relevant dashboard in notifications or tickets
- **Workflow enrichment**: embed dashboard links in Slack messages, Jira issues, or approval steps
- **Panel deep links**: link directly to a specific panel with a preset time range

## Configuration

- **Dashboard**: The Grafana dashboard UID for the share link
- **Panel**: If set, link opens the dashboard at this specific panel
- **From**: Optional expression for the start of the time range (e.g. ` + "`{{now() - duration(\"1h\")}}`" + `)
- **To**: Optional expression for the end of the time range (e.g. ` + "`{{now()}}`" + `)

## Output

Returns the constructed shareable URL along with the dashboard title and UID.`
}

func (c *CreateDashboardShareLink) Icon() string {
	return "layout-dashboard"
}

func (c *CreateDashboardShareLink) Color() string {
	return "blue"
}

func (c *CreateDashboardShareLink) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDashboardShareLink) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dashboardUid",
			Label:       "Dashboard",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana dashboard to create a share link for",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDashboard,
				},
			},
		},
		{
			Name:        "panelId",
			Label:       "Panel",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "If set, link opens the dashboard at this specific panel",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypePanel,
					Parameters: []configuration.ParameterRef{
						{Name: "dashboardUid", ValueFrom: &configuration.ParameterValueFrom{Field: "dashboardUid"}},
					},
				},
			},
		},
		{
			Name:        "from",
			Label:       "From",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Start of the time range",
			Placeholder: "e.g. {{now() - duration(\"1h\")}}",
		},
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "End of the time range",
			Placeholder: "e.g. {{now()}}",
		},
	}
}

func (c *CreateDashboardShareLink) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateDashboardShareLinkSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateCreateDashboardShareLinkSpec(spec); err != nil {
		return err
	}

	storeDashboardNodeMetadata(ctx, spec.DashboardUID, spec.PanelID)
	return nil
}

func (c *CreateDashboardShareLink) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateDashboardShareLinkSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateCreateDashboardShareLinkSpec(spec); err != nil {
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

	slug := dashboardURLPathSlug(dashboard)
	baseURL := fmt.Sprintf("%s/d/%s/%s", strings.TrimSuffix(client.BaseURL, "/"), spec.DashboardUID, slug)

	params := url.Values{}
	if spec.PanelID != nil {
		params.Set("viewPanel", fmt.Sprintf("%d", *spec.PanelID))
	}
	if strings.TrimSpace(spec.From) != "" {
		params.Set("from", spec.From)
	}
	if strings.TrimSpace(spec.To) != "" {
		params.Set("to", spec.To)
	}

	shareURL := baseURL
	if len(params) > 0 {
		shareURL = baseURL + "?" + params.Encode()
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.dashboard.shareLink",
		[]any{CreateDashboardShareLinkOutput{
			URL:            shareURL,
			DashboardTitle: dashboard.Title,
			DashboardUID:   spec.DashboardUID,
		}},
	)
}

func (c *CreateDashboardShareLink) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDashboardShareLink) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDashboardShareLink) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateDashboardShareLink) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateDashboardShareLink) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateDashboardShareLink) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeCreateDashboardShareLinkSpec(input any) (CreateDashboardShareLinkSpec, error) {
	spec := CreateDashboardShareLinkSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return CreateDashboardShareLinkSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := decoder.Decode(input); err != nil {
		return CreateDashboardShareLinkSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.DashboardUID = strings.TrimSpace(spec.DashboardUID)
	spec.From = strings.TrimSpace(spec.From)
	spec.To = strings.TrimSpace(spec.To)
	return spec, nil
}

func validateCreateDashboardShareLinkSpec(spec CreateDashboardShareLinkSpec) error {
	if spec.DashboardUID == "" {
		return errors.New("dashboardUid is required")
	}

	return nil
}
