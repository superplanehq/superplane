package prometheus

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type Query struct{}

type QueryConfiguration struct {
	Query string `json:"query" mapstructure:"query"`
}

type QueryNodeMetadata struct {
	Query string `json:"query"`
}

func (c *Query) Name() string {
	return "prometheus.query"
}

func (c *Query) Label() string {
	return "Query"
}

func (c *Query) Description() string {
	return "Execute a PromQL instant query"
}

func (c *Query) Documentation() string {
	return `The Query component executes an instant PromQL query against Prometheus (` + "`GET /api/v1/query`" + `).

## Configuration

- **Query**: Required PromQL expression to evaluate (supports expressions). Example: ` + "`up`" + `

## Output

Emits one ` + "`prometheus.query`" + ` payload with the result type and results.`
}

func (c *Query) Icon() string {
	return "prometheus"
}

func (c *Query) Color() string {
	return "gray"
}

func (c *Query) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Query) Configuration() []configuration.Field {
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

func (c *Query) Setup(ctx core.SetupContext) error {
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

func (c *Query) Execute(ctx core.ExecutionContext) error {
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

	ctx.Metadata.Set(QueryNodeMetadata{Query: config.Query})

	payload := map[string]any{
		"resultType": data["resultType"],
		"result":     data["result"],
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"prometheus.query",
		[]any{payload},
	)
}

func (c *Query) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Query) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *Query) Actions() []core.Action {
	return []core.Action{}
}

func (c *Query) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *Query) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Query) Cleanup(ctx core.SetupContext) error {
	return nil
}

func sanitizeQueryConfiguration(config QueryConfiguration) QueryConfiguration {
	config.Query = strings.TrimSpace(config.Query)
	return config
}
