package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendLogEvent struct{}

type SendLogEventSpec struct {
	Body         string            `json:"body" mapstructure:"body"`
	SeverityText *string           `json:"severityText,omitempty" mapstructure:"severityText"`
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

- **Body**: The log message content (plain text or JSON string)
- **Severity**: Log severity level (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
- **Dataset**: Optional dataset name for log organization (defaults to "default")
- **Attributes**: Optional key-value pairs for additional log metadata

## Output

Returns a confirmation that the log was sent:
- **sent**: Boolean indicating success
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
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The log message content (plain text or JSON string)",
			Placeholder: "Deployment completed for service 'api-gateway' version 1.2.3",
		},
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
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "default",
			Description: "Dataset name for log organization",
		},
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Additional key-value metadata for the log record",
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

	// Build OTLP log payload
	timestamp := time.Now().UnixNano()

	// Map severity text to OTLP severity number
	severityNumber := s.getSeverityNumber(spec.SeverityText)
	severityText := "INFO"
	if spec.SeverityText != nil {
		severityText = *spec.SeverityText
	}

	// Build attributes array
	attributes := []map[string]any{}
	if spec.Attributes != nil {
		for key, value := range spec.Attributes {
			attributes = append(attributes, map[string]any{
				"key": key,
				"value": map[string]any{
					"stringValue": value,
				},
			})
		}
	}

	// Build the OTLP ExportLogsServiceRequest payload
	otlpPayload := map[string]any{
		"resourceLogs": []map[string]any{
			{
				"scopeLogs": []map[string]any{
					{
						"logRecords": []map[string]any{
							{
								"timeUnixNano":   strconv.FormatInt(timestamp, 10),
								"severityNumber": severityNumber,
								"severityText":   severityText,
								"body": map[string]any{
									"stringValue": spec.Body,
								},
								"attributes": attributes,
							},
						},
					},
				},
			},
		},
	}

	dataset := "default"
	if spec.Dataset != nil && *spec.Dataset != "" {
		dataset = *spec.Dataset
	}

	result, err := client.SendLogRecord(dataset, otlpPayload)
	if err != nil {
		return fmt.Errorf("failed to send log event: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.log.sent",
		[]any{result},
	)
}

// getSeverityNumber maps OTLP severity text to severity number according to OTLP spec
func (s *SendLogEvent) getSeverityNumber(severityText *string) int {
	if severityText == nil {
		return 9 // INFO
	}

	switch strings.ToUpper(*severityText) {
	case "TRACE":
		return 1
	case "DEBUG":
		return 5
	case "INFO":
		return 9
	case "WARN":
		return 13
	case "ERROR":
		return 17
	case "FATAL":
		return 21
	default:
		return 9 // INFO
	}
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

func (s *SendLogEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *SendLogEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
