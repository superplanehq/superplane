package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendLogEvent struct{}

type SendLogEventSpec struct {
	Body         string            `json:"body" mapstructure:"body"`
	SeverityText *string           `json:"severityText,omitempty" mapstructure:"severityText"`
	EventName    string            `json:"eventName,omitempty" mapstructure:"eventName"`
	ServiceName  string            `json:"serviceName,omitempty" mapstructure:"serviceName"`
	Dataset      *string           `json:"dataset,omitempty" mapstructure:"dataset"`
	Attributes   map[string]string `json:"attributes,omitempty" mapstructure:"attributes"`
}

func (s *SendLogEvent) Name() string {
	return "dash0.sendLogEvent"
}

func (s *SendLogEvent) Label() string {
	return "Send Log Event"
}

func (s *SendLogEvent) Description() string {
	return "Send a log record to Dash0 via OTLP HTTP ingestion for audit trails and observability correlation"
}

func (s *SendLogEvent) Documentation() string {
	return `The Send Log Event component sends log records from workflows to Dash0 via OTLP HTTP ingestion.

## Use Cases

- **Audit trails**: Record workflow events (deployments, approvals, alerts) as log lines
- **Observability correlation**: Tie workflow activity to traces and metrics in Dash0
- **Event tracking**: Create searchable log entries for workflow milestones
- **Debugging**: Send diagnostic information from workflows to Dash0 Logs Explorer

## Configuration

- **Severity**: Log severity level (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
- **Event Name**: Optional name for this log event (e.g. deployment.completed)
- **Service Name**: Optional service identifier (becomes OTLP resource attribute 'service.name')
- **Body**: The log message content (plain text or JSON string)
- **Attributes**: Optional key-value pairs for additional log metadata
- **Dataset**: Optional dataset name for log organization (defaults to "default")

## Output

Returns a confirmation that the log was sent along with the log record details:
- **sent**: Boolean indicating success
- **severityText**: The log severity level
- **body**: The log message content
- **eventName**: The event name (if provided)
- **serviceName**: The service name (if provided)
- **attributes**: Additional metadata (if provided)
- **dataset**: The dataset name
- **timestamp**: When the log was sent

## Notes

- Requires Dash0 API token and base URL configured in application settings
- Logs appear in Dash0 Logs Explorer and can be correlated with traces and metrics
- Use INFO severity for normal workflow events, WARN/ERROR for issues`
}

func (s *SendLogEvent) Icon() string {
	return "file-text"
}

func (s *SendLogEvent) Color() string {
	return "blue"
}

func (s *SendLogEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (s *SendLogEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "severityText",
			Label:    "Severity",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "INFO",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "TRACE", Value: "TRACE"},
						{Label: "DEBUG", Value: "DEBUG"},
						{Label: "INFO", Value: "INFO"},
						{Label: "WARN", Value: "WARN"},
						{Label: "ERROR", Value: "ERROR"},
						{Label: "FATAL", Value: "FATAL"},
					},
				},
			},
			Description: "Log severity level",
		},
		{
			Name:        "eventName",
			Label:       "Event Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "A name for this log event (e.g. deployment.completed, approval.granted)",
			Placeholder: "deployment.completed",
		},
		{
			Name:        "serviceName",
			Label:       "Service Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The name of the service generating this log (OTLP resource attribute 'service.name')",
			Placeholder: "api-gateway",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The log message content (plain text or JSON string)",
			Placeholder: "Deployment completed for service 'api-gateway' version 1.2.3",
		},
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Additional key-value metadata for the log record",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "default",
			Description: "Dataset name for log organization",
		},
	}
}

func (s *SendLogEvent) Setup(ctx core.SetupContext) error {
	spec := SendLogEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Body == "" {
		return errors.New("body is required")
	}

	if len(strings.TrimSpace(spec.Body)) == 0 {
		return errors.New("body cannot be empty")
	}

	return nil
}

func (s *SendLogEvent) Execute(ctx core.ExecutionContext) error {
	spec := SendLogEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	severityText := "INFO"
	if spec.SeverityText != nil {
		severityText = *spec.SeverityText
	}

	record := LogRecord{
		SeverityText: severityText,
		Body:         spec.Body,
		EventName:    spec.EventName,
		ServiceName:  spec.ServiceName,
		Attributes:   spec.Attributes,
	}

	dataset := "default"
	if spec.Dataset != nil && *spec.Dataset != "" {
		dataset = *spec.Dataset
	}

	result, err := client.SendLogRecord(dataset, record)
	if err != nil {
		return fmt.Errorf("failed to send log event: %v", err)
	}

	payload := map[string]any{
		"sent":         result["sent"],
		"severityText": record.SeverityText,
		"body":         record.Body,
		"eventName":    record.EventName,
		"serviceName":  record.ServiceName,
		"attributes":   record.Attributes,
		"dataset":      dataset,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.log.sent",
		[]any{payload},
	)
}

func (s *SendLogEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (s *SendLogEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (s *SendLogEvent) Actions() []core.Action {
	return []core.Action{}
}

func (s *SendLogEvent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (s *SendLogEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (s *SendLogEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
