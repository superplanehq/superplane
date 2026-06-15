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

type QueryConfiguration struct {
	Region                    string `json:"region" mapstructure:"region"`
	WorkspaceID               string `json:"workspace" mapstructure:"workspace"`
	Query                     string `json:"query" mapstructure:"query"`
	Time                      string `json:"time" mapstructure:"time"`
	QueryOptionsConfiguration `mapstructure:",squash"`
}

func (c *Query) Name() string {
	return "aws.prometheus.query"
}

func (c *Query) Label() string {
	return "Prometheus • Query"
}

func (c *Query) Description() string {
	return "Execute a PromQL instant query against an Amazon Managed Service for Prometheus workspace"
}

func (c *Query) Documentation() string {
	return `The Query component executes an instant PromQL query against an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Query**: Required PromQL expression to evaluate. Example: ` + "`up`" + `
- **Time**: Optional evaluation timestamp in RFC3339 or Unix format
- **Timeout**: Optional query timeout duration
- **Query sample thresholds**: Optional warning and error thresholds for query samples processed

## Output

Emits one ` + "`aws.prometheus.query`" + ` payload with the result type and results.`
}

func (c *Query) Icon() string {
	return "aws"
}

func (c *Query) Color() string {
	return "gray"
}

func (c *Query) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Query) Configuration() []configuration.Field {
	fields := []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace to query"),
		queryField(),
		{
			Name:        "time",
			Label:       "Time",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional evaluation timestamp, as RFC3339 or Unix timestamp",
		},
	}

	return append(fields, queryOptionFields()...)
}

func (c *Query) Setup(ctx core.SetupContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}))
}

func (c *Query) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	response, err := client.QueryMetrics(QueryMetricsInput{
		WorkspaceID:                         config.WorkspaceID,
		Query:                               config.Query,
		Time:                                config.Time,
		Timeout:                             config.Timeout,
		MaxSamplesProcessedWarningThreshold: config.MaxSamplesProcessedWarningThreshold,
		MaxSamplesProcessedErrorThreshold:   config.MaxSamplesProcessedErrorThreshold,
	})
	if err != nil {
		return fmt.Errorf("failed to execute PromQL query: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.query",
		[]any{queryOutput(response)},
	)
}

func (c *Query) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Query) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Query) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Query) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *Query) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Query) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *Query) decodeConfiguration(rawConfiguration any) (QueryConfiguration, error) {
	config := QueryConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return QueryConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.Query = strings.TrimSpace(config.Query)
	config.Time = strings.TrimSpace(config.Time)
	config.Timeout = strings.TrimSpace(config.Timeout)

	if config.Region == "" {
		return QueryConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return QueryConfiguration{}, fmt.Errorf("workspace is required")
	}
	if config.Query == "" {
		return QueryConfiguration{}, fmt.Errorf("query is required")
	}
	if err := validateQueryOptions(config.QueryOptionsConfiguration); err != nil {
		return QueryConfiguration{}, err
	}

	return config, nil
}
