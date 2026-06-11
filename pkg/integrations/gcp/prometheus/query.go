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

type Query struct{}

type QuerySpec struct {
	Query string `mapstructure:"query"`
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
	return "Run an instant PromQL query against Google Cloud Managed Service for Prometheus"
}

func (q *Query) Documentation() string {
	return `The Query component runs an instant PromQL query against Google Cloud Managed Service for Prometheus (GMP).

GMP stores Prometheus metrics in Cloud Monitoring (Monarch) and exposes a Prometheus-compatible HTTP frontend. This component calls ` + "`GET /v1/projects/<project>/location/global/prometheus/api/v1/query`" + ` and returns the result at a single point in time.

## Use Cases

- **Threshold checks**: Evaluate an expression (e.g. ` + "`up`" + ` or ` + "`rate(...)`" + `) and branch on the value
- **Spot readings**: Read the current value of a metric to enrich a workflow
- **Chaining**: Feed a metric value into a downstream notification or decision node

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions). Example: ` + "`up`" + `

The expression is evaluated at execution time ("now").

## Output

Emits one ` + "`gcp.prometheus.query`" + ` payload:
- **resultType**: ` + "`vector`" + `, ` + "`scalar`" + `, etc.
- **result**: the Prometheus result (series with their labels and value)
- **seriesCount**: number of series returned

## Important Notes

- Requires the ` + "`roles/monitoring.viewer`" + ` IAM role on the integration's service account
- An invalid PromQL expression fails the action with the Prometheus error message`
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
	}
}

func (q *Query) Setup(ctx core.SetupContext) error {
	spec := QuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Query) == "" {
		return fmt.Errorf("query is required")
	}
	return ctx.Metadata.Set(QueryNodeMetadata{Query: strings.TrimSpace(spec.Query)})
}

func (q *Query) Execute(ctx core.ExecutionContext) error {
	spec := QuerySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	query := strings.TrimSpace(spec.Query)
	if query == "" {
		return ctx.ExecutionState.Fail("error", "query is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	payload, err := runQuery(client, instantQueryURL(client.ProjectID(), query))
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to query managed prometheus", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.prometheus.query",
		[]any{payload},
	)
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
