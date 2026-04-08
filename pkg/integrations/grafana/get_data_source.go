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

type GetDataSource struct{}

type GetDataSourceSpec struct {
	DataSourceUID string `json:"dataSourceUid" mapstructure:"dataSourceUid"`
}

func (g *GetDataSource) Name() string {
	return "grafana.getDataSource"
}

func (g *GetDataSource) Label() string {
	return "Get Data Source"
}

func (g *GetDataSource) Description() string {
	return "Retrieve details for a specific Grafana data source by UID"
}

func (g *GetDataSource) Documentation() string {
	return `The Get Data Source component fetches the full details of a single Grafana data source by UID.

## Use Cases

- **Inspect source type**: Confirm the datasource type (Prometheus, Loki, Tempo, etc.) before deciding how to query it
- **Validate connectivity**: Confirm the data source URL and configuration are present before running downstream queries
- **Workflow routing**: Branch on datasource type to select the appropriate query component

## Configuration

- **Data Source**: The Grafana data source to inspect (required)

## Output

Returns the data source object including UID, name, type, URL, and whether it is the default.
`
}

func (g *GetDataSource) Icon() string {
	return "database"
}

func (g *GetDataSource) Color() string {
	return "blue"
}

func (g *GetDataSource) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDataSource) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dataSourceUid",
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana data source to inspect",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDataSource,
				},
			},
		},
	}
}

func (g *GetDataSource) Setup(ctx core.SetupContext) error {
	spec, err := decodeGetDataSourceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateGetDataSourceSpec(spec)
}

func (g *GetDataSource) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetDataSourceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateGetDataSourceSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	source, err := client.GetDataSource(strings.TrimSpace(spec.DataSourceUID))
	if err != nil {
		return fmt.Errorf("error getting data source: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.data-source",
		[]any{source},
	)
}

func (g *GetDataSource) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (g *GetDataSource) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetDataSource) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetDataSource) HandleAction(_ core.ActionContext) error {
	return nil
}

func (g *GetDataSource) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetDataSource) Cleanup(_ core.SetupContext) error {
	return nil
}

func decodeGetDataSourceSpec(config any) (GetDataSourceSpec, error) {
	spec := GetDataSourceSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return GetDataSourceSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return GetDataSourceSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateGetDataSourceSpec(spec GetDataSourceSpec) error {
	if strings.TrimSpace(spec.DataSourceUID) == "" {
		return errors.New("dataSourceUid is required")
	}
	return nil
}
