package prometheus

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// rangeLookbackOptions are the relative windows offered for a range query,
// matching the lookback convention used by the metrics components.
var rangeLookbackOptions = []configuration.FieldOption{
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
	{Label: "Last 14 days", Value: "14d"},
}

var rangeLookbackDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

// rangeLookbackStepSeconds is the query resolution step per window, kept coarse
// enough to stay well within the Prometheus points-per-query limit.
var rangeLookbackStepSeconds = map[string]int{
	"1h":  60,
	"6h":  300,
	"24h": 300,
	"7d":  3600,
	"14d": 3600,
}

type QueryRange struct{}

type QueryRangeSpec struct {
	Query          string `mapstructure:"query"`
	LookbackPeriod string `mapstructure:"lookbackPeriod"`
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
	return `The Query Range component runs a PromQL range query against Google Cloud Managed Service for Prometheus (GMP) over a relative lookback window.

GMP stores Prometheus metrics in Cloud Monitoring (Monarch) and exposes a Prometheus-compatible HTTP frontend. This component calls ` + "`GET /v1/projects/<project>/location/global/prometheus/api/v1/query_range`" + ` and returns a matrix of samples from ` + "`now - lookback`" + ` to ` + "`now`" + `.

## Use Cases

- **Trend analysis**: Pull a metric over a window to summarise or chart it downstream
- **Incident investigation**: Fetch recent samples when responding to an alert
- **Anomaly checks**: Evaluate an expression across time before acting

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions). Example: ` + "`rate(prometheus_http_requests_total[5m])`" + `
- **Lookback Period**: How far back to query (1 hour to 14 days). The component computes the start/end window and a sensible resolution step automatically.

## Output

Emits one ` + "`gcp.prometheus.queryRange`" + ` payload:
- **resultType**: typically ` + "`matrix`" + `
- **result**: the Prometheus result (series with their labels and ` + "`values`" + ` over time)
- **seriesCount**: number of series returned
- **lookbackPeriod**, **start**, **end**, **step**: the resolved query window

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
			Name:        "lookbackPeriod",
			Label:       "Lookback Period",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "1h",
			Description: "How far back to query metrics data",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: rangeLookbackOptions,
				},
			},
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

	duration := rangeLookbackDurations[spec.LookbackPeriod]
	stepSeconds := rangeLookbackStepSeconds[spec.LookbackPeriod]
	end := time.Now().UTC()
	start := end.Add(-duration)
	step := fmt.Sprintf("%ds", stepSeconds)

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	url := rangeQueryURL(
		client.ProjectID(),
		strings.TrimSpace(spec.Query),
		strconv.FormatInt(start.Unix(), 10),
		strconv.FormatInt(end.Unix(), 10),
		step,
	)
	payload, err := runQuery(client, url)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to query managed prometheus", err))
	}

	payload["lookbackPeriod"] = spec.LookbackPeriod
	payload["start"] = start.Format(time.RFC3339)
	payload["end"] = end.Format(time.RFC3339)
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
	if spec.LookbackPeriod == "" {
		return fmt.Errorf("lookbackPeriod is required")
	}
	if _, ok := rangeLookbackDurations[spec.LookbackPeriod]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of 1h, 6h, 24h, 7d, 14d", spec.LookbackPeriod)
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
