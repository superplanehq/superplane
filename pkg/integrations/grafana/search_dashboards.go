package grafana

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SearchDashboards struct{}

type SearchDashboardsSpec struct {
	Query     string `json:"query" mapstructure:"query"`
	FolderUID string `json:"folderUID" mapstructure:"folderUID"`
	Tag       string `json:"tag" mapstructure:"tag"`
	Limit     int    `json:"limit" mapstructure:"limit"`
}

type SearchDashboardsOutput struct {
	Dashboards []DashboardSummary `json:"dashboards" mapstructure:"dashboards"`
}

func (c *SearchDashboards) Name() string {
	return "grafana.searchDashboards"
}

func (c *SearchDashboards) Label() string {
	return "Search Dashboards"
}

func (c *SearchDashboards) Description() string {
	return "Search Grafana dashboards by title, folder, or tag"
}

func (c *SearchDashboards) Documentation() string {
	return `The Search Dashboards component searches Grafana dashboards using the Grafana Search HTTP API.

## Use Cases

- **Dashboard discovery**: find dashboards by title, folder, or tag for downstream automation
- **Workflow enrichment**: include dashboard lists in notifications, tickets, or approval steps
- **Audit and inventory**: enumerate dashboards matching specific criteria for review or cleanup

## Configuration

- **Query**: Optional title filter for dashboards
- **Folder**: Optional folder to filter dashboards by
- **Tag**: Optional tag to filter dashboards by
- **Limit**: Optional maximum number of dashboards to return

## Output

Returns an object containing the list of matching Grafana dashboard summaries, including each dashboard UID, title, URL, folder, and tags.`
}

func (c *SearchDashboards) Icon() string {
	return "layout-dashboard"
}

func (c *SearchDashboards) Color() string {
	return "blue"
}

func (c *SearchDashboards) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SearchDashboards) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter dashboards by title",
		},
		{
			Name:        "folderUID",
			Label:       "Folder",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter dashboards by folder",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeFolder,
				},
			},
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter dashboards by tag",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Maximum number of dashboards to return",
		},
	}
}

func (c *SearchDashboards) Setup(ctx core.SetupContext) error {
	_, err := decodeSearchDashboardsSpec(ctx.Configuration)
	return err
}

func (c *SearchDashboards) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSearchDashboardsSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	results, err := client.SearchDashboards(spec.Query, spec.FolderUID, spec.Tag, spec.Limit)
	if err != nil {
		return fmt.Errorf("error searching dashboards: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.dashboards",
		[]any{SearchDashboardsOutput{
			Dashboards: results,
		}},
	)
}

func (c *SearchDashboards) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SearchDashboards) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SearchDashboards) Actions() []core.Action {
	return []core.Action{}
}

func (c *SearchDashboards) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *SearchDashboards) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SearchDashboards) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeSearchDashboardsSpec(input any) (SearchDashboardsSpec, error) {
	spec := SearchDashboardsSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return SearchDashboardsSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := decoder.Decode(input); err != nil {
		return SearchDashboardsSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.Query = strings.TrimSpace(spec.Query)
	spec.FolderUID = strings.TrimSpace(spec.FolderUID)
	spec.Tag = strings.TrimSpace(spec.Tag)
	return spec, nil
}
