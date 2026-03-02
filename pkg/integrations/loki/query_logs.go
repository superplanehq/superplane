package loki

import (
	"encoding/json"
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
	Query string `json:"query"`
	Start string `json:"start"`
	End   string `json:"end"`
	Limit string `json:"limit"`
}

type QueryLogsNodeMetadata struct {
	Query string `json:"query"`
}

func (c *QueryLogs) Name() string {
	return "loki.queryLogs"
}

func (c *QueryLogs) Label() string {
	return "Query Logs"
}

func (c *QueryLogs) Description() string {
	return "Query logs from Loki using LogQL"
}

func (c *QueryLogs) Documentation() string {
	return `The Query Logs component executes a LogQL query against Loki (` + "`GET /loki/api/v1/query_range`" + `).

## Configuration

- **Query**: Required LogQL expression (supports expressions). Example: ` + "`{job=\"superplane\"}`" + `
- **Start**: Optional start timestamp in RFC3339 or Unix nanosecond format (supports expressions)
- **End**: Optional end timestamp in RFC3339 or Unix nanosecond format (supports expressions)
- **Limit**: Optional maximum number of entries to return (default: 100)

## Output

Emits one ` + "`loki.queryLogs`" + ` payload with the result type and query results.`
}

func (c *QueryLogs) Icon() string {
	return "loki"
}

func (c *QueryLogs) Color() string {
	return "gray"
}

func (c *QueryLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "Query",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: `{job="superplane"}`,
			Description: "LogQL expression to evaluate",
		},
		{
			Name:        "start",
			Label:       "Start",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "2026-01-01T00:00:00Z",
			Description: "Start timestamp (RFC3339 or Unix nanoseconds)",
		},
		{
			Name:        "end",
			Label:       "End",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "2026-01-02T00:00:00Z",
			Description: "End timestamp (RFC3339 or Unix nanoseconds)",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "100",
			Description: "Maximum number of entries to return",
		},
	}
}

func (c *QueryLogs) Setup(ctx core.SetupContext) error {
	spec := QueryLogsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec = sanitizeQueryLogsSpec(spec)

	if spec.Query == "" {
		return fmt.Errorf("query is required")
	}

	return nil
}

func (c *QueryLogs) Execute(ctx core.ExecutionContext) error {
	spec := QueryLogsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec = sanitizeQueryLogsSpec(spec)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Loki client: %w", err)
	}

	data, err := client.QueryRange(spec.Query, spec.Start, spec.End, spec.Limit)
	if err != nil {
		return fmt.Errorf("failed to query logs: %w", err)
	}

	ctx.Metadata.Set(QueryLogsNodeMetadata{Query: spec.Query})

	var result any
	if err := json.Unmarshal(data.Result, &result); err != nil {
		result = json.RawMessage(data.Result)
	}

	payload := map[string]any{
		"resultType": data.ResultType,
		"result":     result,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"loki.queryLogs",
		[]any{payload},
	)
}

func (c *QueryLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *QueryLogs) Actions() []core.Action {
	return []core.Action{}
}

func (c *QueryLogs) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *QueryLogs) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *QueryLogs) ExampleOutput() map[string]any {
	return exampleOutputQueryLogs()
}

func sanitizeQueryLogsSpec(spec QueryLogsSpec) QueryLogsSpec {
	spec.Query = strings.TrimSpace(spec.Query)
	spec.Start = strings.TrimSpace(spec.Start)
	spec.End = strings.TrimSpace(spec.End)
	spec.Limit = strings.TrimSpace(spec.Limit)
	return spec
}
