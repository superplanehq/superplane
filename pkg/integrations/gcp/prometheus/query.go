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

// lookbackInstant runs a point-in-time ("instant") query instead of a range
// query. It is the default so the component reads a single current value unless
// a window is chosen.
const lookbackInstant = "instant"

// lookbackOptions are the windows offered by the Query component. "Instant"
// evaluates the expression at "now"; the others run a range query from
// now-lookback to now.
var lookbackOptions = []configuration.FieldOption{
	{Label: "Instant (current value)", Value: lookbackInstant},
	{Label: "Last 5 minutes", Value: "5m"},
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
	{Label: "Last 14 days", Value: "14d"},
}

var lookbackDurations = map[string]time.Duration{
	"5m":  5 * time.Minute,
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

// lookbackStepSeconds is the range-query resolution per window, kept coarse
// enough to stay well within the Prometheus points-per-query limit.
var lookbackStepSeconds = map[string]int{
	"5m":  15,
	"1h":  60,
	"6h":  300,
	"24h": 300,
	"7d":  3600,
	"14d": 3600,
}

type Query struct{}

type QuerySpec struct {
	Query          string `mapstructure:"query"`
	LookbackPeriod string `mapstructure:"lookbackPeriod"`
}

type QueryNodeMetadata struct {
	Query string `json:"query" mapstructure:"query"`
}

func (q *Query) Name() string {
	return "gcp.prometheus.query"
}

func (q *Query) Label() string {
	return "Managed Prometheus • Query"
}

func (q *Query) Description() string {
	return "Run a PromQL query against Google Cloud Managed Service for Prometheus"
}

func (q *Query) Documentation() string {
	return `The Query component runs a PromQL query against Google Cloud Managed Service for Prometheus (GMP).

GMP stores Prometheus metrics in Cloud Monitoring (Monarch) and exposes a Prometheus-compatible HTTP frontend. By default this component runs an **instant** query (the value at "now"); choosing a **Lookback Period** instead runs a **range** query from ` + "`now - lookback`" + ` to ` + "`now`" + `.

## Use Cases

- **Threshold checks**: Evaluate an expression (e.g. ` + "`up`" + ` or ` + "`rate(...)`" + `) and branch on the value
- **Spot readings**: Read the current value of a metric to enrich a workflow
- **Trend analysis**: Pull a metric over a window to summarise or chart it downstream

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions). Example: ` + "`up`" + `
- **Lookback Period**: ` + "`Instant`" + ` (default) evaluates at a single point in time. A window (5 minutes to 14 days) runs a range query; the component derives the start/end and a sensible resolution step automatically.

## Output

Emits one ` + "`gcp.prometheus.query`" + ` payload:
- **resultType**: ` + "`vector`" + ` / ` + "`scalar`" + ` for instant queries, ` + "`matrix`" + ` for range queries
- **result**: the Prometheus result (series with their labels and value(s))
- **seriesCount**: number of series returned
- For a range query, also **lookbackPeriod**, **start**, **end**, **step**: the resolved query window

## Important Notes

- Requires the ` + "`roles/monitoring.viewer`" + ` IAM role on the integration's service account
- An invalid PromQL expression fails the action with the Prometheus error message
- When querying Google Cloud metrics whose type spans multiple resource types, GMP requires a ` + "`monitored_resource`" + ` label matcher (e.g. ` + "`{__name__=\"...\", monitored_resource=\"consumed_api\"}`" + `)`
}

func (q *Query) Icon() string {
	return "chart-line"
}

func (q *Query) Color() string {
	return "blue"
}

func (q *Query) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *Query) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "up",
			Description: "PromQL expression to evaluate",
		},
		{
			Name:        "lookbackPeriod",
			Label:       "Lookback Period",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     lookbackInstant,
			Description: "Evaluate at a single point in time (Instant) or over a relative window (range query)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: lookbackOptions,
				},
			},
		},
	}
}

func (q *Query) Setup(ctx core.SetupContext) error {
	spec := QuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateQuerySpec(spec); err != nil {
		return err
	}
	return ctx.Metadata.Set(QueryNodeMetadata{Query: strings.TrimSpace(spec.Query)})
}

func (q *Query) Execute(ctx core.ExecutionContext) error {
	spec := QuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	if err := validateQuerySpec(spec); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	query := strings.TrimSpace(spec.Query)
	lookback := strings.TrimSpace(spec.LookbackPeriod)

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	// Instant query: a single point-in-time read.
	if isInstantLookback(lookback) {
		payload, err := runQuery(client, instantQueryURL(client.ProjectID(), query))
		if err != nil {
			return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to query managed prometheus", err))
		}
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.prometheus.query", []any{payload})
	}

	// Range query over the lookback window.
	duration := lookbackDurations[lookback]
	step := fmt.Sprintf("%ds", lookbackStepSeconds[lookback])
	end := time.Now().UTC()
	start := end.Add(-duration)

	payload, err := runQuery(client, rangeQueryURL(
		client.ProjectID(),
		query,
		strconv.FormatInt(start.Unix(), 10),
		strconv.FormatInt(end.Unix(), 10),
		step,
	))
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to query managed prometheus", err))
	}
	payload["lookbackPeriod"] = lookback
	payload["start"] = start.Format(time.RFC3339)
	payload["end"] = end.Format(time.RFC3339)
	payload["step"] = step

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.prometheus.query", []any{payload})
}

// isInstantLookback reports whether the lookback selects an instant query. An
// empty value defaults to instant, preserving point-in-time behavior when the
// field is omitted.
func isInstantLookback(lookback string) bool {
	return lookback == "" || lookback == lookbackInstant
}

func validateQuerySpec(spec QuerySpec) error {
	if strings.TrimSpace(spec.Query) == "" {
		return fmt.Errorf("query is required")
	}
	lookback := strings.TrimSpace(spec.LookbackPeriod)
	if isInstantLookback(lookback) {
		return nil
	}
	if _, ok := lookbackDurations[lookback]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of instant, 5m, 1h, 6h, 24h, 7d, 14d", lookback)
	}
	return nil
}

func (q *Query) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (q *Query) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *Query) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (q *Query) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (q *Query) Hooks() []core.Hook {
	return []core.Hook{}
}

func (q *Query) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
