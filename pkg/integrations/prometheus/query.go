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

const PrometheusQueryPayloadType = "prometheus.queryResult"

type QueryComponent struct{}

type QueryConfiguration struct {
	Query string `json:"query" mapstructure:"query"`
}

func (c *QueryComponent) Name() string {
	return "prometheus.query"
}

func (c *QueryComponent) Label() string {
	return "Query"
}

func (c *QueryComponent) Description() string {
	return "Execute an instant query against Prometheus"
}

func (c *QueryComponent) Documentation() string {
	return `The Query component executes an instant PromQL query against the Prometheus API (` + "`/api/v1/query`" + `).

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions)

## Output

Emits one ` + "`prometheus.queryResult`" + ` payload with the query result data including resultType and result array.`
}

func (c *QueryComponent) Icon() string {
	return "prometheus"
}

func (c *QueryComponent) Color() string {
	return "gray"
}

func (c *QueryComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "PromQL Query",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "up",
			Description: "PromQL expression to evaluate",
		},
	}
}

func (c *QueryComponent) Setup(ctx core.SetupContext) error {
	config := QueryConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeQueryConfiguration(config)

	if config.Query == "" {
		return fmt.Errorf("query is required")
	}

	return nil
}

func (c *QueryComponent) Execute(ctx core.ExecutionContext) error {
	config := QueryConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeQueryConfiguration(config)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	data, err := client.Query(config.Query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	payload := map[string]any{
		"query":      config.Query,
		"resultType": data["resultType"],
		"result":     data["result"],
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PrometheusQueryPayloadType,
		[]any{payload},
	)
}

func (c *QueryComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *QueryComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *QueryComponent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *QueryComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeQueryConfiguration(config QueryConfiguration) QueryConfiguration {
	config.Query = strings.TrimSpace(config.Query)
	return config
}
