package prometheus

import (
	"fmt"
	"strings"

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

type QueryRangeNodeMetadata struct {
	Query string `json:"query"`
}

func (c *QueryRange) Name() string {
	return "prometheus.queryRange"
}

func (c *QueryRange) Label() string {
	return "Query Range"
}

func (c *QueryRange) Description() string {
	return "Execute a PromQL range query"
}

func (c *QueryRange) Documentation() string {
	return `The Query Range component executes a range PromQL query against Prometheus (` + "`GET /api/v1/query_range`" + `).

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions). Example: ` + "`up`" + `
- **Start**: Required start timestamp in RFC3339 or Unix format (supports expressions). Example: ` + "`2026-01-01T00:00:00Z`" + `
- **End**: Required end timestamp in RFC3339 or Unix format (supports expressions). Example: ` + "`2026-01-02T00:00:00Z`" + `
- **Step**: Required query resolution step (e.g. ` + "`15s`" + `, ` + "`1m`" + `)

## Output

Emits one ` + "`prometheus.queryRange`" + ` payload with the result type and results.`
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
			Label:       "Query",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "up",
			Description: "PromQL expression to evaluate",
		},
		{
			Name:        "start",
			Label:       "Start",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-01T00:00:00Z",
			Description: "Start timestamp (RFC3339 or Unix)",
		},
		{
			Name:        "end",
			Label:       "End",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-02T00:00:00Z",
			Description: "End timestamp (RFC3339 or Unix)",
		},
		{
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "15s",
			Description: "Query resolution step (e.g. 15s, 1m)",
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
		return fmt.Errorf("failed to execute query range: %w", err)
	}

	ctx.Metadata.Set(QueryRangeNodeMetadata{Query: config.Query})

	payload := map[string]any{
		"resultType": data["resultType"],
		"result":     data["result"],
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"prometheus.queryRange",
		[]any{payload},
	)
}

func (c *QueryRange) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryRange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
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
