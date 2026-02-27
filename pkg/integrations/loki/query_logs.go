package loki

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryLogs struct{}

type QueryLogsSpec struct {
	Query     string `json:"query"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Limit     string `json:"limit"`
	Direction string `json:"direction"`
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

func (c *QueryLogs) Icon() string {
	return "file-text"
}

func (c *QueryLogs) Color() string {
	return "gray"
}

func (c *QueryLogs) Documentation() string {
	return `The Query Logs component queries logs from a Grafana Loki instance using LogQL.

## Use Cases

- **Log analysis**: Query logs to check for errors or specific patterns in a workflow
- **Health checks**: Verify system health by querying for recent error logs
- **Incident investigation**: Retrieve logs related to a specific service or time window

## Outputs

The component emits an event containing:
- ` + "`status`" + `: The query status (e.g., "success")
- ` + "`result_type`" + `: The type of result (e.g., "streams")
- ` + "`result`" + `: Array of stream results, each containing labels and log values
`
}

func (c *QueryLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "LogQL Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The LogQL query to execute",
			Placeholder: `{job="superplane"}`,
		},
		{
			Name:        "start",
			Label:       "Start Time",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Start of the query time range (e.g., 2026-01-01T00:00:00Z, 1h, 30m)",
			Placeholder: "1h",
		},
		{
			Name:        "end",
			Label:       "End Time",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "End of the query time range (e.g., 2026-01-01T01:00:00Z, now)",
			Placeholder: "now",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "100",
			Description: "Maximum number of log entries to return",
		},
		{
			Name:     "direction",
			Label:    "Direction",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "backward",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Backward (newest first)", Value: "backward"},
						{Label: "Forward (oldest first)", Value: "forward"},
					},
				},
			},
		},
	}
}

func (c *QueryLogs) Setup(ctx core.SetupContext) error {
	spec := QueryLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Query == "" {
		return errors.New("query is required")
	}

	return nil
}

func (c *QueryLogs) Execute(ctx core.ExecutionContext) error {
	spec := QueryLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	response, err := client.QueryLogs(spec.Query, spec.Start, spec.End, spec.Limit, spec.Direction)
	if err != nil {
		return fmt.Errorf("failed to query logs: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"loki.queryResult",
		[]any{queryResultToMap(response)},
	)
}

func (c *QueryLogs) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryLogs) Actions() []core.Action {
	return []core.Action{}
}

func (c *QueryLogs) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *QueryLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *QueryLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}

func queryResultToMap(response *QueryResponse) map[string]any {
	results := make([]map[string]any, 0, len(response.Data.Result))
	for _, r := range response.Data.Result {
		results = append(results, map[string]any{
			"stream": r.Stream,
			"values": r.Values,
		})
	}

	return map[string]any{
		"status":      response.Status,
		"result_type": response.Data.ResultType,
		"result":      results,
	}
}
