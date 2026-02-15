package prometheus

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryRange struct{}

type QueryRangeConfiguration struct {
	Query string `json:"query" mapstructure:"query"`
	Start string `json:"start" mapstructure:"start"`
	End   string `json:"end" mapstructure:"end"`
	Step  string `json:"step" mapstructure:"step"`
}

func (c *QueryRange) Name() string {
	return "prometheus.queryRange"
}

func (c *QueryRange) Label() string {
	return "Query Range"
}

func (c *QueryRange) Description() string {
	return "Execute a range query against Prometheus"
}

func (c *QueryRange) Documentation() string {
	return `The Query Range component executes a range PromQL query against the Prometheus API (` + "`/api/v1/query_range`" + `).

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions)
- **Start**: Required start timestamp in RFC3339 format or relative duration (supports expressions)
- **End**: Required end timestamp in RFC3339 format or relative duration (supports expressions)
- **Step**: Required query resolution step width as a duration (e.g., ` + "`15s`" + `, ` + "`1m`" + `, ` + "`5m`" + `)

## Output

Emits one ` + "`prometheus.queryResult`" + ` payload with the range query result data including resultType and result array.`
}

func (c *QueryRange) Icon() string {
	return "prometheus"
}

func (c *QueryRange) Color() string {
	return "gray"
}

func (c *QueryRange) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryRange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "PromQL Query",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "rate(http_requests_total[5m])",
			Description: "PromQL expression to evaluate",
		},
		{
			Name:        "start",
			Label:       "Start",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-01T00:00:00Z",
			Description: "Start timestamp (RFC3339) or relative duration (e.g., 1h ago)",
		},
		{
			Name:        "end",
			Label:       "End",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-01T01:00:00Z",
			Description: "End timestamp (RFC3339) or relative duration (e.g., now)",
		},
		{
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "15s",
			Placeholder: "15s",
			Description: "Query resolution step width (e.g., 15s, 1m, 5m)",
		},
	}
}

func (c *QueryRange) Setup(ctx core.SetupContext) error {
	config := QueryRangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeQueryRangeConfiguration(config)

	if config.Query == "" {
		return fmt.Errorf("query is required")
	}

	if config.Start == "" {
		return fmt.Errorf("start is required")
	}

	if config.End == "" {
		return fmt.Errorf("end is required")
	}

	if config.Step == "" {
		return fmt.Errorf("step is required")
	}

	if _, err := time.ParseDuration(config.Step); err != nil {
		return fmt.Errorf("invalid step %q: must be a valid duration (e.g., 15s, 1m, 5m)", config.Step)
	}

	return nil
}

func (c *QueryRange) Execute(ctx core.ExecutionContext) error {
	config := QueryRangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeQueryRangeConfiguration(config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	data, err := client.QueryRange(config.Query, config.Start, config.End, config.Step)
	if err != nil {
		return fmt.Errorf("failed to execute range query: %w", err)
	}

	payload := map[string]any{
		"query":      config.Query,
		"start":      config.Start,
		"end":        config.End,
		"step":       config.Step,
		"resultType": data["resultType"],
		"result":     data["result"],
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PrometheusQueryPayloadType,
		[]any{payload},
	)
}

func (c *QueryRange) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryRange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *QueryRange) Actions() []core.Action {
	return []core.Action{}
}

func (c *QueryRange) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *QueryRange) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryRange) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeQueryRangeConfiguration(config QueryRangeConfiguration) QueryRangeConfiguration {
	config.Query = strings.TrimSpace(config.Query)
	config.Start = strings.TrimSpace(config.Start)
	config.End = strings.TrimSpace(config.End)
	config.Step = strings.TrimSpace(config.Step)
	return config
}
