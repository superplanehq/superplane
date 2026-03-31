package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
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
	Timezone      *string `json:"timezone,omitempty"`
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

const grafanaDateTimeFormat = "2006-01-02T15:04"

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

- **Data Source**: The Grafana data source to query
- **Query**: The datasource query (PromQL, InfluxQL, etc.)
- **Time From / Time To**: Optional datetime picker values for the query range
- **Timezone**: Interprets datetime picker values using the selected timezone offset
- If omitted, SuperPlane defaults the query to the last 5 minutes
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
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Grafana data source to query",
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
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Optional start of the query time range",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
		},
		{
			Name:        "timeTo",
			Label:       "Time To",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Optional end of the query time range",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
		},
		{
			Name:        "timezone",
			Label:       "Timezone",
			Type:        configuration.FieldTypeTimezone,
			Required:    false,
			Default:     "current",
			Description: "Timezone offset used for Time From / Time To picker values. Relative Grafana values like now-1h ignore this field.",
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

func (q *QueryDataSource) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (q *QueryDataSource) Cleanup(ctx core.SetupContext) error {
	return nil
}

func defaultTimeRange() (string, string) {
	now := time.Now().UTC()
	from := now.Add(-5 * time.Minute)
	return fmt.Sprintf("%d", from.UnixMilli()), fmt.Sprintf("%d", now.UnixMilli())
}

func resolveQueryTimeValue(value string, timezone *string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	if parsed, ok, err := parseGrafanaQueryTime(trimmed, timezone); err != nil {
		return "", err
	} else if ok {
		return fmt.Sprintf("%d", parsed.UTC().UnixMilli()), nil
	}

	// Preserve Grafana-supported raw values like "now-1h".
	return trimmed, nil
}

func parseGrafanaQueryTime(value string, timezone *string) (time.Time, bool, error) {
	for _, format := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04Z07:00",
	} {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed, true, nil
		}
	}

	for _, format := range []string{"2006-01-02T15:04:05", grafanaDateTimeFormat} {
		if _, err := time.Parse(format, value); err != nil {
			continue
		}

		location, err := parseGrafanaQueryTimezone(timezone)
		if err != nil {
			return time.Time{}, false, err
		}

		parsed, err := time.ParseInLocation(format, value, location)
		if err != nil {
			return time.Time{}, false, err
		}

		return parsed, true, nil
	}

	return time.Time{}, false, nil
}

func parseGrafanaQueryTimezone(timezone *string) (*time.Location, error) {
	if timezone == nil || strings.TrimSpace(*timezone) == "" {
		return nil, errors.New("timezone is required for datetime-local values")
	}

	trimmed := strings.TrimSpace(*timezone)
	if trimmed == "current" {
		return nil, errors.New("timezone value 'current' must be resolved before execution")
	}

	offsetHours, err := strconv.ParseFloat(strings.TrimPrefix(trimmed, "+"), 64)
	if err != nil {
		return nil, fmt.Errorf("expected numeric offset like -5, 0, 5.5, or +8")
	}

	if offsetHours < -12 || offsetHours > 14 {
		return nil, fmt.Errorf("offset must be between -12 and +14 hours")
	}

	offsetSeconds := int(math.Round(offsetHours * 3600))
	return time.FixedZone(fmt.Sprintf("GMT%+.1f", offsetHours), offsetSeconds), nil
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
	if err := validateQueryTimeValue(spec.TimeFrom, spec.Timezone); err != nil {
		return fmt.Errorf("timeFrom: %w", err)
	}
	if err := validateQueryTimeValue(spec.TimeTo, spec.Timezone); err != nil {
		return fmt.Errorf("timeTo: %w", err)
	}

	return nil
}

func validateQueryTimeValue(value *string, timezone *string) error {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}

	_, _, err := parseGrafanaQueryTime(strings.TrimSpace(*value), timezone)
	return err
}
