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

type QueryTraces struct{}

type QueryTracesSpec struct {
	DataSourceUID string  `json:"dataSourceUid" mapstructure:"dataSourceUid"`
	Query         string  `json:"query" mapstructure:"query"`
	TimeFrom      *string `json:"timeFrom,omitempty" mapstructure:"timeFrom"`
	TimeTo        *string `json:"timeTo,omitempty" mapstructure:"timeTo"`
	Timezone      *string `json:"timezone,omitempty" mapstructure:"timezone"`
}

func (q *QueryTraces) Name() string {
	return "grafana.queryTraces"
}

func (q *QueryTraces) Label() string {
	return "Query Traces"
}

func (q *QueryTraces) Description() string {
	return "Run a TraceQL query against a Tempo data source and return matching traces"
}

func (q *QueryTraces) Documentation() string {
	return `The Query Traces component executes a TraceQL query against a Tempo-backed Grafana data source.

## Use Cases

- **Incident triage**: Find traces for a failing service during an incident to identify slow or erroring spans
- **Deploy validation**: Confirm trace patterns look healthy after a deployment
- **Latency investigation**: Search for high-latency traces matching a specific service or operation

## Configuration

- **Data Source**: The Tempo data source to query (required)
- **Query**: A TraceQL query expression (required), e.g. ` + "`{ .http.status_code = 500 }`" + `
- **Time From / Time To**: Optional datetime range for the trace search window
- **Timezone**: Interprets datetime picker values using the selected timezone offset

## Output

Returns the Grafana query API response containing matching trace frames.
`
}

func (q *QueryTraces) Icon() string {
	return "git-branch"
}

func (q *QueryTraces) Color() string {
	return "blue"
}

func (q *QueryTraces) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QueryTraces) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dataSourceUid",
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Tempo data source to query",
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
			Description: "TraceQL query expression",
			Placeholder: `{ .http.status_code = 500 }`,
		},
		{
			Name:        "timeFrom",
			Label:       "Time From",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Optional start of the trace search time range",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: grafanaDateTimeFormat,
				},
			},
		},
		{
			Name:        "timeTo",
			Label:       "Time To",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Optional end of the trace search time range",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: grafanaDateTimeFormat,
				},
			},
		},
		{
			Name:        "timezone",
			Label:       "Timezone",
			Type:        configuration.FieldTypeTimezone,
			Required:    false,
			Default:     "current",
			Description: "Timezone offset used for Time From / Time To picker values",
		},
	}
}

func (q *QueryTraces) Setup(ctx core.SetupContext) error {
	spec, err := decodeQueryTracesSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateQueryTracesSpec(spec)
}

func (q *QueryTraces) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeQueryTracesSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateQueryTracesSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dataSourceUID := strings.TrimSpace(spec.DataSourceUID)

	source, err := client.GetDataSource(dataSourceUID)
	if err != nil {
		return fmt.Errorf("error getting data source: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(source.Type), "tempo") {
		return fmt.Errorf("data source %q must be a Tempo data source, got %q", dataSourceUID, source.Type)
	}

	request := map[string]any{
		"queries": []any{
			map[string]any{
				"refId":      "A",
				"datasource": map[string]string{"uid": dataSourceUID},
				"queryType":  "traceql",
				"query":      strings.TrimSpace(spec.Query),
				"filters":    []any{},
				"limit":      20,
				"spss":       3,
				"tableType":  "traces",
			},
		},
	}

	if spec.TimeFrom != nil && strings.TrimSpace(*spec.TimeFrom) != "" {
		timeFrom, resolveErr := resolveQueryTimeValue(*spec.TimeFrom, spec.Timezone)
		request["from"] = timeFrom
		err = resolveErr
		if err != nil {
			return fmt.Errorf("invalid timeFrom value %q: %w", strings.TrimSpace(*spec.TimeFrom), err)
		}
	}

	if spec.TimeTo != nil && strings.TrimSpace(*spec.TimeTo) != "" {
		timeTo, resolveErr := resolveQueryTimeValue(*spec.TimeTo, spec.Timezone)
		request["to"] = timeTo
		err = resolveErr
		if err != nil {
			return fmt.Errorf("invalid timeTo value %q: %w", strings.TrimSpace(*spec.TimeTo), err)
		}
	}

	fromValue, _ := request["from"].(string)
	toValue, _ := request["to"].(string)
	if fromValue == "" || toValue == "" {
		from, to := defaultTimeRange()
		if fromValue == "" {
			request["from"] = from
		}
		if toValue == "" {
			request["to"] = to
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, status, err := client.execRequest(http.MethodPost, "/api/ds/query", bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("error querying traces: %v", err)
	}

	if status < 200 || status >= 300 {
		return fmt.Errorf("grafana trace query failed with status %d: %s", status, string(responseBody))
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.traces.result",
		[]any{response},
	)
}

func (q *QueryTraces) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (q *QueryTraces) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QueryTraces) Actions() []core.Action {
	return []core.Action{}
}

func (q *QueryTraces) HandleAction(_ core.ActionContext) error {
	return nil
}

func (q *QueryTraces) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (q *QueryTraces) Cleanup(_ core.SetupContext) error {
	return nil
}

func decodeQueryTracesSpec(config any) (QueryTracesSpec, error) {
	spec := QueryTracesSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return QueryTracesSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return QueryTracesSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateQueryTracesSpec(spec QueryTracesSpec) error {
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
