package prometheus

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryRange struct{}

type QueryRangeSpec struct {
	Query string `mapstructure:"query"`
	Start string `mapstructure:"start"`
	End   string `mapstructure:"end"`
	Step  string `mapstructure:"step"`
}

func (q *QueryRange) Name() string {
	return "gcp.prometheus.queryRange"
}

func (q *QueryRange) Label() string {
	return "Managed Prometheus • Query Range"
}

func (q *QueryRange) Description() string {
	return "Run a PromQL range query against Google Cloud Managed Service for Prometheus"
}

func (q *QueryRange) Documentation() string {
	return `The Query Range component runs a PromQL range query against Google Cloud Managed Service for Prometheus (GMP) over an explicit time range.

GMP stores Prometheus metrics in Cloud Monitoring (Monarch) and exposes a Prometheus-compatible HTTP frontend. This component calls ` + "`GET /v1/projects/<project>/location/global/prometheus/api/v1/query_range`" + ` and returns a matrix of samples between ` + "`start`" + ` and ` + "`end`" + ` at the given ` + "`step`" + ` resolution.

## Use Cases

- **Trend analysis**: Pull a metric over a window to summarise or chart it downstream
- **Incident investigation**: Fetch samples for a specific time range when responding to an alert
- **Anomaly checks**: Evaluate an expression across time before acting

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions). Example: ` + "`rate(prometheus_http_requests_total[5m])`" + `
- **Start**: Required start timestamp in RFC3339 or Unix format (supports expressions). Example: ` + "`2026-01-01T00:00:00Z`" + `
- **End**: Required end timestamp in RFC3339 or Unix format (supports expressions). Example: ` + "`2026-01-02T00:00:00Z`" + `
- **Step**: Required query resolution step (e.g. ` + "`15s`" + `, ` + "`1m`" + `)

## Output

Emits one ` + "`gcp.prometheus.queryRange`" + ` payload:
- **resultType**: typically ` + "`matrix`" + `
- **result**: the Prometheus result (series with their labels and ` + "`values`" + ` over time)
- **seriesCount**: number of series returned
- **start**, **end**, **step**: the query window

## Important Notes

- Requires the ` + "`roles/monitoring.viewer`" + ` IAM role on the integration's service account
- An invalid PromQL expression fails the action with the Prometheus error message`
}

func (q *QueryRange) Icon() string {
	return "chart-line"
}

func (q *QueryRange) Color() string {
	return "blue"
}

func (q *QueryRange) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QueryRange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "rate(prometheus_http_requests_total[5m])",
			Description: "PromQL expression to evaluate",
		},
		{
			Name:        "start",
			Label:       "Start",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-01T00:00:00Z",
			Default:     "2026-01-01T00:00:00Z",
			Description: "Start timestamp (RFC3339 or Unix)",
		},
		{
			Name:        "end",
			Label:       "End",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-02T00:00:00Z",
			Default:     "2026-01-02T00:00:00Z",
			Description: "End timestamp (RFC3339 or Unix)",
		},
		{
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "15s",
			Default:     "15s",
			Description: "Query resolution step (e.g. 15s, 1m)",
		},
	}
}

func (q *QueryRange) Setup(ctx core.SetupContext) error {
	spec := QueryRangeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateRangeSpec(spec); err != nil {
		return err
	}
	return ctx.Metadata.Set(QueryNodeMetadata{Query: strings.TrimSpace(spec.Query)})
}

func (q *QueryRange) Execute(ctx core.ExecutionContext) error {
	spec := QueryRangeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	if err := validateRangeSpec(spec); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	start := strings.TrimSpace(spec.Start)
	end := strings.TrimSpace(spec.End)
	step := strings.TrimSpace(spec.Step)

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	url := rangeQueryURL(client.ProjectID(), strings.TrimSpace(spec.Query), start, end, step)
	payload, err := runQuery(client, url)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to query managed prometheus", err))
	}

	payload["start"] = start
	payload["end"] = end
	payload["step"] = step

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.prometheus.queryRange",
		[]any{payload},
	)
}

func validateRangeSpec(spec QueryRangeSpec) error {
	if strings.TrimSpace(spec.Query) == "" {
		return fmt.Errorf("query is required")
	}
	if strings.TrimSpace(spec.Start) == "" {
		return fmt.Errorf("start is required")
	}
	if strings.TrimSpace(spec.End) == "" {
		return fmt.Errorf("end is required")
	}
	if strings.TrimSpace(spec.Step) == "" {
		return fmt.Errorf("step is required")
	}
	return nil
}

func (q *QueryRange) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (q *QueryRange) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QueryRange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (q *QueryRange) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (q *QueryRange) Hooks() []core.Hook {
	return []core.Hook{}
}

func (q *QueryRange) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
