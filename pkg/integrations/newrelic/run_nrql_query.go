package newrelic

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const maxNRQLTimeout = 120 // seconds

type RunNRQLQuery struct{}

type RunNRQLQuerySpec struct {
	Query   string `json:"query" mapstructure:"query"`
	Timeout int    `json:"timeout" mapstructure:"timeout"`
}

func (c *RunNRQLQuery) Name() string {
	return "newrelic.runNRQLQuery"
}

func (c *RunNRQLQuery) Label() string {
	return "Run NRQL Query"
}

func (c *RunNRQLQuery) Description() string {
	return "Run a NRQL query against New Relic data via NerdGraph"
}

func (c *RunNRQLQuery) Icon() string {
	return "chart-bar"
}

func (c *RunNRQLQuery) Color() string {
	return "gray"
}

func (c *RunNRQLQuery) Documentation() string {
	return `The Run NRQL Query component executes a NRQL query against New Relic data via the NerdGraph API.

## Use Cases

- **Health checks**: Query application error rates or response times before deployments
- **Capacity planning**: Check resource utilization metrics
- **Incident investigation**: Query telemetry data during incident workflows

## Configuration

- ` + "`query`" + `: The NRQL query string to execute
- ` + "`timeout`" + `: Optional query timeout in seconds (default: 30)

## Outputs

The component emits query results containing:
- ` + "`query`" + `: The executed NRQL query
- ` + "`results`" + `: Array of result rows returned by the query
`
}

func (c *RunNRQLQuery) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunNRQLQuery) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "NRQL Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The NRQL query to execute",
			Placeholder: "SELECT count(*) FROM Transaction SINCE 1 hour ago",
		},
		{
			Name:     "timeout",
			Label:    "Timeout (seconds)",
			Type:     configuration.FieldTypeNumber,
			Required: false,
			Default:  30,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
					Max: func() *int { max := maxNRQLTimeout; return &max }(),
				},
			},
			Description: "Query timeout in seconds",
		},
	}
}

func (c *RunNRQLQuery) Setup(ctx core.SetupContext) error {
	spec := RunNRQLQuerySpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Query == "" {
		return errors.New("query is required")
	}

	if spec.Timeout > maxNRQLTimeout {
		return fmt.Errorf("timeout cannot exceed %d seconds", maxNRQLTimeout)
	}

	return nil
}

func (c *RunNRQLQuery) Execute(ctx core.ExecutionContext) error {
	spec := RunNRQLQuerySpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	timeout := 30
	if spec.Timeout > 0 {
		timeout = spec.Timeout
	}

	if timeout > maxNRQLTimeout {
		timeout = maxNRQLTimeout
	}

	results, err := client.RunNRQLQuery(context.Background(), spec.Query, timeout)
	if err != nil {
		return fmt.Errorf("failed to execute NRQL query: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"newrelic.nrqlResult",
		[]any{map[string]any{
			"query":   spec.Query,
			"results": results,
		}},
	)
}

func (c *RunNRQLQuery) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunNRQLQuery) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunNRQLQuery) Actions() []core.Action {
	return []core.Action{}
}

func (c *RunNRQLQuery) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RunNRQLQuery) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RunNRQLQuery) Cleanup(ctx core.SetupContext) error {
	return nil
}
