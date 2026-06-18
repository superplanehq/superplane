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

type QueryRange struct{}

type QueryRangeConfiguration struct {
	Region                    string `json:"region" mapstructure:"region"`
	WorkspaceID               string `json:"workspace" mapstructure:"workspace"`
	Query                     string `json:"query" mapstructure:"query"`
	Start                     string `json:"start" mapstructure:"start"`
	End                       string `json:"end" mapstructure:"end"`
	Step                      string `json:"step" mapstructure:"step"`
	QueryOptionsConfiguration `mapstructure:",squash"`
}

func (c *QueryRange) Name() string {
	return "aws.prometheus.queryRange"
}

func (c *QueryRange) Label() string {
	return "Prometheus • Query Range"
}

func (c *QueryRange) Description() string {
	return "Execute a PromQL range query against an Amazon Managed Service for Prometheus workspace"
}

func (c *QueryRange) Documentation() string {
	return `The Query Range component executes a range PromQL query against an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Query**: Required PromQL expression to evaluate. Example: ` + "`up`" + `
- **Start**: Required start timestamp in RFC3339 or Unix format. Example: ` + "`2026-01-01T00:00:00Z`" + `
- **End**: Required end timestamp in RFC3339 or Unix format. Example: ` + "`2026-01-02T00:00:00Z`" + `
- **Step**: Required query resolution step. Example: ` + "`15s`" + `
- **Timeout**: Optional query timeout duration
- **Query sample thresholds**: Optional warning and error thresholds for query samples processed

## Output

Emits one ` + "`aws.prometheus.queryRange`" + ` payload with the result type and results.`
}

func (c *QueryRange) Icon() string {
	return "aws"
}

func (c *QueryRange) Color() string {
	return "gray"
}

func (c *QueryRange) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryRange) Configuration() []configuration.Field {
	fields := []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace to query"),
		queryField(),
		{
			Name:        "start",
			Label:       "Start",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-01T00:00:00Z",
			Default:     "2026-01-01T00:00:00Z",
			Description: "Start timestamp (RFC3339 or Unix)",
		},
		{
			Name:        "end",
			Label:       "End",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-01-02T00:00:00Z",
			Default:     "2026-01-02T00:00:00Z",
			Description: "End timestamp (RFC3339 or Unix)",
		},
		{
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "15s",
			Default:     "15s",
			Description: "Query resolution step (e.g. 15s, 1m)",
		},
	}

	return append(fields, queryOptionFields()...)
}

func (c *QueryRange) Setup(ctx core.SetupContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}))
}

func (c *QueryRange) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	response, err := client.QueryRangeMetrics(QueryRangeMetricsInput{
		WorkspaceID:                         config.WorkspaceID,
		Query:                               config.Query,
		Start:                               config.Start,
		End:                                 config.End,
		Step:                                config.Step,
		Timeout:                             config.Timeout,
		MaxSamplesProcessedWarningThreshold: config.MaxSamplesProcessedWarningThreshold,
		MaxSamplesProcessedErrorThreshold:   config.MaxSamplesProcessedErrorThreshold,
	})
	if err != nil {
		return fmt.Errorf("failed to execute PromQL query range: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.queryRange",
		[]any{queryOutput(response)},
	)
}

func (c *QueryRange) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryRange) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *QueryRange) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryRange) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *QueryRange) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *QueryRange) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *QueryRange) decodeConfiguration(rawConfiguration any) (QueryRangeConfiguration, error) {
	config := QueryRangeConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return QueryRangeConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.Query = strings.TrimSpace(config.Query)
	config.Start = strings.TrimSpace(config.Start)
	config.End = strings.TrimSpace(config.End)
	config.Step = strings.TrimSpace(config.Step)
	config.Timeout = strings.TrimSpace(config.Timeout)

	if config.Region == "" {
		return QueryRangeConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return QueryRangeConfiguration{}, fmt.Errorf("workspace is required")
	}
	if config.Query == "" {
		return QueryRangeConfiguration{}, fmt.Errorf("query is required")
	}
	if config.Start == "" {
		return QueryRangeConfiguration{}, fmt.Errorf("start is required")
	}
	if config.End == "" {
		return QueryRangeConfiguration{}, fmt.Errorf("end is required")
	}
	if config.Step == "" {
		return QueryRangeConfiguration{}, fmt.Errorf("step is required")
	}
	if err := validateQueryOptions(config.QueryOptionsConfiguration); err != nil {
		return QueryRangeConfiguration{}, err
	}

	return config, nil
}
