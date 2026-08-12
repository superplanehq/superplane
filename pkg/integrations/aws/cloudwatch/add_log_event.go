package cloudwatch

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const AddLogEventPayloadType = "aws.cloudwatch.logEvent"

type AddLogEvent struct{}

type AddLogEventConfiguration struct {
	Region    string `json:"region" mapstructure:"region"`
	LogGroup  string `json:"logGroup" mapstructure:"logGroup"`
	LogStream string `json:"logStream" mapstructure:"logStream"`
	Message   string `json:"message" mapstructure:"message"`
	Timestamp string `json:"timestamp" mapstructure:"timestamp"`
}

func (c *AddLogEvent) Name() string {
	return "aws.cloudwatch.addLogEvent"
}

func (c *AddLogEvent) Label() string {
	return "CloudWatch • Add Log Event"
}

func (c *AddLogEvent) Description() string {
	return "Write a log event to a CloudWatch Logs log stream"
}

func (c *AddLogEvent) Documentation() string {
	return `The Add Log Event component writes a single log event to a CloudWatch Logs log stream, creating the stream first if it doesn't exist yet.

## Use Cases

- **Workflow auditing**: Record workflow decisions or approvals in a searchable log group
- **External event logging**: Forward events from other systems into CloudWatch Logs for correlation with application logs
- **Custom instrumentation**: Emit structured log lines that downstream Logs Insights queries can pick up

## How It Works

1. Creates the log stream if it doesn't already exist
2. Writes the message as a log event, using the given timestamp or the current time

## Configuration

- **Region**: AWS region of the log group
- **Log Group**: Target log group
- **Log Stream**: Name of the log stream to write to; created automatically if missing
- **Message**: The log event message
- **Timestamp**: Optional RFC3339 timestamp; defaults to now. Must be within the last 14 days and no more than 2 hours in the future

## Output

- ` + "`logGroup`" + `, ` + "`logStream`" + `, ` + "`message`" + `, ` + "`timestamp`" + ``
}

func (c *AddLogEvent) Icon() string {
	return "aws"
}

func (c *AddLogEvent) Color() string {
	return "gray"
}

func (c *AddLogEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddLogEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "logGroup",
			Label:       "Log Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Log group to write to",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "cloudwatch.logGroup",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "logStream",
			Label:       "Log Stream",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Log stream to write to; created automatically if it doesn't exist",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "logGroup", Values: []string{"*"}},
			},
		},
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The log event message",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "logStream", Values: []string{"*"}},
			},
		},
		{
			Name:        "timestamp",
			Label:       "Timestamp",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Defaults to now (RFC3339)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "logStream", Values: []string{"*"}},
			},
		},
	}
}

func (c *AddLogEvent) Setup(ctx core.SetupContext) error {
	config := AddLogEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	_, err := validateAddLogEventConfiguration(config)
	return err
}

func (c *AddLogEvent) Execute(ctx core.ExecutionContext) error {
	config := AddLogEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	timestamp, err := validateAddLogEventConfiguration(config)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	if err := client.CreateLogStream(config.LogGroup, config.LogStream); err != nil && !common.IsAlreadyExistsErr(err) {
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	err = client.PutLogEvents(config.LogGroup, config.LogStream, []LogEvent{
		{Message: config.Message, Timestamp: timestamp},
	})
	if err != nil {
		return fmt.Errorf("failed to put log event: %w", err)
	}

	output := map[string]any{
		"logGroup":  config.LogGroup,
		"logStream": config.LogStream,
		"message":   config.Message,
		"timestamp": timestamp.Format(time.RFC3339),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, AddLogEventPayloadType, []any{output})
}

func (c *AddLogEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddLogEvent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddLogEvent) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *AddLogEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddLogEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddLogEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}

// validateAddLogEventConfiguration validates the configuration and resolves the log event timestamp.
func validateAddLogEventConfiguration(config AddLogEventConfiguration) (time.Time, error) {
	if strings.TrimSpace(config.Region) == "" {
		return time.Time{}, fmt.Errorf("region is required")
	}

	if strings.TrimSpace(config.LogGroup) == "" {
		return time.Time{}, fmt.Errorf("log group is required")
	}

	if strings.TrimSpace(config.LogStream) == "" {
		return time.Time{}, fmt.Errorf("log stream is required")
	}

	if strings.TrimSpace(config.Message) == "" {
		return time.Time{}, fmt.Errorf("message is required")
	}

	timestampStr := strings.TrimSpace(config.Timestamp)
	if timestampStr == "" {
		return time.Now().UTC(), nil
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: must be RFC3339", config.Timestamp)
	}

	return timestamp.UTC(), nil
}
