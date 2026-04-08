package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryLogs struct{}

type QueryLogsSpec struct {
	DataSourceUID string  `json:"dataSourceUid" mapstructure:"dataSourceUid"`
	Query         string  `json:"query" mapstructure:"query"`
	TimeFrom      *string `json:"timeFrom,omitempty" mapstructure:"timeFrom"`
	TimeTo        *string `json:"timeTo,omitempty" mapstructure:"timeTo"`
	Timezone      *string `json:"timezone,omitempty" mapstructure:"timezone"`
	Limit         *int    `json:"limit,omitempty" mapstructure:"limit"`
}

func (q *QueryLogs) Name() string {
	return "grafana.queryLogs"
}

func (q *QueryLogs) Label() string {
	return "Query Logs"
}

func (q *QueryLogs) Description() string {
	return "Run a LogQL query against a Loki data source and return structured log results"
}

func (q *QueryLogs) Documentation() string {
	return `The Query Logs component executes a LogQL query against a Loki-backed Grafana data source.

## Use Cases

- **Incident investigation**: Search logs for errors or anomalies during an incident response workflow
- **Deploy validation**: Confirm absence of error patterns following a deployment
- **Log enrichment**: Pull relevant log lines into a workflow for summarization or downstream notification

## Configuration

	- **Data Source**: The Loki data source to query (required)
	- **Query**: A LogQL query expression (required), e.g. ` + "`{app=\"myservice\"} |= \"error\"`" + `
	- **Time From / Time To**: Optional log query range. Supports absolute values like ` + "`2026-04-08T15:30Z`" + ` and relative Grafana values like ` + "`now-15m`" + ` or ` + "`now+2h`" + `
	- **Limit**: Maximum number of log lines to return (optional)

## Output

Returns the Grafana query API response containing matching log frames.
`
}

func (q *QueryLogs) Icon() string {
	return "file-text"
}

func (q *QueryLogs) Color() string {
	return "blue"
}

func (q *QueryLogs) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QueryLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dataSourceUid",
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Loki data source to query",
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
			Description: "LogQL query expression",
			Placeholder: `{app="myservice"} |= "error"`,
		},
		{
			Name:        "timeFrom",
			Label:       "Time From",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional start of the query time range",
			Placeholder: "now-15m or 2026-04-08T15:30",
		},
		{
			Name:        "timeTo",
			Label:       "Time To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional end of the query time range",
			Placeholder: "now or now+2h",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Maximum number of log lines to return",
			Placeholder: "1000",
		},
	}
}

func (q *QueryLogs) Setup(ctx core.SetupContext) error {
	spec, err := decodeQueryLogsSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateQueryLogsSpec(spec)
}

func (q *QueryLogs) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeQueryLogsSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateQueryLogsSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	lokiQuery := grafanaQuery{
		RefID:      "A",
		Datasource: map[string]string{"uid": strings.TrimSpace(spec.DataSourceUID)},
		Expr:       strings.TrimSpace(spec.Query),
		Query:      strings.TrimSpace(spec.Query),
	}

	if spec.Limit != nil && *spec.Limit > 0 {
		lokiQuery.MaxLines = *spec.Limit
	}

	request := grafanaQueryRequest{
		Queries: []grafanaQuery{lokiQuery},
	}

	if spec.TimeFrom != nil && strings.TrimSpace(*spec.TimeFrom) != "" {
		request.From, err = resolveQueryTimeValue(*spec.TimeFrom, spec.Timezone)
		if err != nil {
			return fmt.Errorf("invalid timeFrom value %q: %w", strings.TrimSpace(*spec.TimeFrom), err)
		}
	}

	if spec.TimeTo != nil && strings.TrimSpace(*spec.TimeTo) != "" {
		request.To, err = resolveQueryTimeValue(*spec.TimeTo, spec.Timezone)
		if err != nil {
			return fmt.Errorf("invalid timeTo value %q: %w", strings.TrimSpace(*spec.TimeTo), err)
		}
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

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, status, err := client.execRequest(http.MethodPost, "/api/ds/query", bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("error querying logs: %v", err)
	}

	if status < 200 || status >= 300 {
		return fmt.Errorf("grafana log query failed with status %d: %s", status, string(responseBody))
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.logs.result",
		[]any{response},
	)
}

func (q *QueryLogs) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (q *QueryLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QueryLogs) Actions() []core.Action {
	return []core.Action{}
}

func (q *QueryLogs) HandleAction(_ core.ActionContext) error {
	return nil
}

func (q *QueryLogs) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (q *QueryLogs) Cleanup(_ core.SetupContext) error {
	return nil
}

func decodeQueryLogsSpec(config any) (QueryLogsSpec, error) {
	spec := QueryLogsSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return QueryLogsSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return QueryLogsSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateQueryLogsSpec(spec QueryLogsSpec) error {
	if strings.TrimSpace(spec.DataSourceUID) == "" {
		return errors.New("dataSourceUid is required")
	}
	if strings.TrimSpace(spec.Query) == "" {
		return errors.New("query is required")
	}
	if err := validateQueryTimeValue(spec.TimeFrom, spec.Timezone); err != nil {
		return fmt.Errorf("timeFrom: %w", err)
	}
	if err := validateQueryTimeValue(spec.TimeTo, spec.Timezone); err != nil {
		return fmt.Errorf("timeTo: %w", err)
	}
	return nil
}
