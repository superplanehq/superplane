package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryDataSource struct{}

type QueryDataSourceSpec struct {
	DataSourceUID string  `json:"dataSourceUid"`
	Query         string  `json:"query"`
	TimeFrom      *string `json:"timeFrom,omitempty"`
	TimeTo        *string `json:"timeTo,omitempty"`
	Format        *string `json:"format,omitempty"`
}

type grafanaQueryRequest struct {
	Queries []grafanaQuery `json:"queries"`
	From    string         `json:"from,omitempty"`
	To      string         `json:"to,omitempty"`
}

type grafanaQuery struct {
	RefID      string `json:"refId"`
	Datasource any    `json:"datasource,omitempty"`
	Expr       string `json:"expr,omitempty"`
	Query      string `json:"query,omitempty"`
	Format     string `json:"format,omitempty"`
}

func (q *QueryDataSource) Name() string {
	return "grafana.queryDataSource"
}

func (q *QueryDataSource) Label() string {
	return "Query Data Source"
}

func (q *QueryDataSource) Description() string {
	return "Execute a query against a Grafana data source and return the result"
}

func (q *QueryDataSource) Documentation() string {
	return `The Query Data Source component executes a query against a Grafana data source using the Grafana Query API.

## Use Cases

- **Metrics investigation**: Run PromQL or other datasource queries from workflows
- **Alert validation**: Validate alert conditions before escalation
- **Incident context**: Pull current metrics into incident workflows

## Configuration

- **Data Source UID**: The Grafana datasource UID to query
- **Query**: The datasource query (PromQL, InfluxQL, etc.)
- **Time From / Time To**: Optional time range (relative like "now-5m" or absolute)
- **Format**: Optional query format (depends on the datasource)

## Output

Returns the Grafana query API response JSON.
`
}

func (q *QueryDataSource) Icon() string {
	return "database"
}

func (q *QueryDataSource) Color() string {
	return "blue"
}

func (q *QueryDataSource) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QueryDataSource) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dataSourceUid",
			Label:       "Data Source UID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana datasource UID to query",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDataSource,
				},
			},
		},
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The datasource query (PromQL, InfluxQL, etc.)",
			Placeholder: "sum(rate(http_requests_total[5m]))",
		},
		{
			Name:        "timeFrom",
			Label:       "Time From",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Start time (e.g. now-5m or 2024-01-01T00:00:00Z)",
			Placeholder: "now-5m",
		},
		{
			Name:        "timeTo",
			Label:       "Time To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "End time (e.g. now or 2024-01-01T01:00:00Z)",
			Placeholder: "now",
		},
		{
			Name:        "format",
			Label:       "Format",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional format passed to the datasource query",
		},
	}
}

func (q *QueryDataSource) Setup(ctx core.SetupContext) error {
	spec, err := decodeQueryDataSourceSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return validateQueryDataSourceSpec(spec)
}

func (q *QueryDataSource) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeQueryDataSourceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateQueryDataSourceSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	request := grafanaQueryRequest{
		Queries: []grafanaQuery{
			{
				RefID:      "A",
				Datasource: map[string]string{"uid": strings.TrimSpace(spec.DataSourceUID)},
				Expr:       strings.TrimSpace(spec.Query),
				Query:      strings.TrimSpace(spec.Query),
			},
		},
	}

	if spec.TimeFrom != nil && strings.TrimSpace(*spec.TimeFrom) != "" {
		request.From = strings.TrimSpace(*spec.TimeFrom)
	}

	if spec.TimeTo != nil && strings.TrimSpace(*spec.TimeTo) != "" {
		request.To = strings.TrimSpace(*spec.TimeTo)
	}

	if request.From == "" || request.To == "" {
		from, to := defaultTimeRange()
		if request.From == "" {
			request.From = from
		}
		if request.To == "" {
			request.To = to
		}
	}

	if spec.Format != nil && strings.TrimSpace(*spec.Format) != "" {
		request.Queries[0].Format = strings.TrimSpace(*spec.Format)
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, status, err := client.execRequest(http.MethodPost, "/api/ds/query", bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("error querying data source: %v", err)
	}

	if status < 200 || status >= 300 {
		return fmt.Errorf("grafana query failed with status %d: %s", status, string(responseBody))
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.query.result",
		[]any{response},
	)
}

func (q *QueryDataSource) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (q *QueryDataSource) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QueryDataSource) Actions() []core.Action {
	return []core.Action{}
}

func (q *QueryDataSource) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (q *QueryDataSource) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (q *QueryDataSource) Cleanup(ctx core.SetupContext) error {
	return nil
}

func defaultTimeRange() (string, string) {
	now := time.Now().UTC()
	from := now.Add(-5 * time.Minute)
	return fmt.Sprintf("%d", from.UnixMilli()), fmt.Sprintf("%d", now.UnixMilli())
}

func decodeQueryDataSourceSpec(configuration any) (QueryDataSourceSpec, error) {
	spec := QueryDataSourceSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return QueryDataSourceSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	return spec, nil
}

func validateQueryDataSourceSpec(spec QueryDataSourceSpec) error {
	if strings.TrimSpace(spec.DataSourceUID) == "" {
		return errors.New("dataSourceUid is required")
	}
	if strings.TrimSpace(spec.Query) == "" {
		return errors.New("query is required")
	}

	return nil
}
