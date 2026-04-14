package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListDataSources struct{}

type ListDataSourcesOutput struct {
	DataSources []DataSource `json:"dataSources"`
}

func (l *ListDataSources) Name() string {
	return "grafana.listDataSources"
}

func (l *ListDataSources) Label() string {
	return "List Data Sources"
}

func (l *ListDataSources) Description() string {
	return "List available Grafana data sources and their basic metadata"
}

func (l *ListDataSources) Documentation() string {
	return `The List Data Sources component retrieves all data sources configured in your Grafana instance.

## Use Cases

- **Discover sources**: Find the correct metrics, logs, or traces data source before running targeted queries
- **Validate sources**: Confirm a required data source exists before executing a query workflow
- **Dynamic routing**: Branch workflow logic based on the type or presence of a specific data source

## Output

Returns a list of data source objects including UID, name, type, and whether the source is the default.
`
}

func (l *ListDataSources) Icon() string {
	return "database"
}

func (l *ListDataSources) Color() string {
	return "blue"
}

func (l *ListDataSources) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListDataSources) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (l *ListDataSources) Setup(_ core.SetupContext) error {
	return nil
}

func (l *ListDataSources) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	sources, err := client.ListDataSources()
	if err != nil {
		return fmt.Errorf("error listing data sources: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.data-sources",
		[]any{ListDataSourcesOutput{DataSources: sources}},
	)
}

func (l *ListDataSources) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (l *ListDataSources) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListDataSources) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListDataSources) HandleAction(_ core.ActionContext) error {
	return nil
}

func (l *ListDataSources) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (l *ListDataSources) Cleanup(_ core.SetupContext) error {
	return nil
}
