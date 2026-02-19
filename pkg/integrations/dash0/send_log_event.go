package dash0

import (
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

const (
	SendLogEventPayloadType = "dash0.log.event.sent"
	maxLogEventRecords      = 50
)

var logSeverityOptions = []configuration.FieldOption{
	{Label: "Trace", Value: "TRACE"},
	{Label: "Debug", Value: "DEBUG"},
	{Label: "Info", Value: "INFO"},
	{Label: "Warn", Value: "WARN"},
	{Label: "Error", Value: "ERROR"},
	{Label: "Fatal", Value: "FATAL"},
}

var severityNumberByText = map[string]int{
	"TRACE": 1,
	"DEBUG": 5,
	"INFO":  9,
	"WARN":  13,
	"ERROR": 17,
	"FATAL": 21,
}

// SendLogEvent publishes workflow log records to Dash0 OTLP HTTP ingestion.
type SendLogEvent struct{}

// Name returns the stable component identifier.
func (c *SendLogEvent) Name() string {
	return "dash0.sendLogEvent"
}

// Label returns the display name used in the workflow builder.
func (c *SendLogEvent) Label() string {
	return "Send Log Event"
}

// Description returns a short summary of component behavior.
func (c *SendLogEvent) Description() string {
	return "Send one or more workflow log records to Dash0 OTLP HTTP ingestion"
}

// Documentation returns markdown help shown in the component docs panel.
func (c *SendLogEvent) Documentation() string {
	return `The Send Log Event component sends workflow log records to Dash0 using OTLP HTTP ingestion.

## Use Cases

- **Audit trails**: Record workflow milestones in Dash0 logs
- **Change tracking**: Emit deployment, approval, and remediation events
- **Observability correlation**: Correlate workflow activity with traces and metrics

## Configuration

- **Service Name**: Service name attached to emitted records
- **Records**: One or more log records containing:
  - message
  - severity
  - timestamp (optional, RFC3339 or unix)
  - attributes (optional key/value map)

## Output

Emits:
- **serviceName**: Service name used for ingestion
- **sentCount**: Number of records sent in this request`
}

// Icon returns the Lucide icon name for this component.
func (c *SendLogEvent) Icon() string {
	return "file-text"
}

// Color returns the node color used in the UI.
func (c *SendLogEvent) Color() string {
	return "blue"
}

// OutputChannels declares the channel emitted by this action.
func (c *SendLogEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration defines fields required to send OTLP log records.
func (c *SendLogEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serviceName",
			Label:       "Service Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "superplane.workflow",
			Description: "Service name attached to ingested log records",
			Placeholder: "superplane.workflow",
		},
		{
			Name:        "records",
			Label:       "Records",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "List of log records to send (max 50 per execution)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Record",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "message",
								Label:       "Message",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Log message text",
							},
							{
								Name:        "severity",
								Label:       "Severity",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Default:     "INFO",
								Description: "Log severity",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: logSeverityOptions,
									},
								},
							},
							{
								Name:        "timestamp",
								Label:       "Timestamp",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Optional timestamp (RFC3339, unix seconds, unix milliseconds, or unix nanoseconds)",
								Placeholder: "2026-02-09T12:00:00Z",
							},
							{
								Name:        "attributes",
								Label:       "Attributes",
								Type:        configuration.FieldTypeObject,
								Required:    false,
								Togglable:   true,
								Description: "Optional key/value attributes for this record",
							},
						},
					},
				},
			},
		},
	}
}

// Setup validates component configuration during save/setup.
func (c *SendLogEvent) Setup(ctx core.SetupContext) error {
	scope := "dash0.sendLogEvent setup"
	config := SendLogEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	return validateSendLogEventConfiguration(config, scope)
}

// ProcessQueueItem delegates queue processing to default behavior.
func (c *SendLogEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute transforms configured records into OTLP and sends them to Dash0.
func (c *SendLogEvent) Execute(ctx core.ExecutionContext) error {
	scope := "dash0.sendLogEvent execute"
	config := SendLogEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if err := validateSendLogEventConfiguration(config, scope); err != nil {
		return err
	}

	request, serviceName, err := buildOTLPLogRequest(config, scope)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: create client: %w", scope, err)
	}

	response, err := client.SendLogEvents(request)
	if err != nil {
		return fmt.Errorf("%s: send log events: %w", scope, err)
	}

	payload := map[string]any{
		"serviceName": serviceName,
		"sentCount":   len(config.Records),
		"response":    response,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, SendLogEventPayloadType, []any{payload})
}

// Actions returns no manual actions for this component.
func (c *SendLogEvent) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this component has no actions.
func (c *SendLogEvent) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook is unused because this component does not receive webhooks.
func (c *SendLogEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel is a no-op because execution is synchronous and short-lived.
func (c *SendLogEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (c *SendLogEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}

// validateSendLogEventConfiguration enforces required fields and record constraints.
func validateSendLogEventConfiguration(config SendLogEventConfiguration, scope string) error {
	if len(config.Records) == 0 {
		return fmt.Errorf("%s: records is required", scope)
	}

	if len(config.Records) > maxLogEventRecords {
		return fmt.Errorf("%s: records cannot exceed %d", scope, maxLogEventRecords)
	}

	for index, record := range config.Records {
		recordScope := fmt.Sprintf("%s: record[%d]", scope, index)
		if strings.TrimSpace(record.Message) == "" {
			return fmt.Errorf("%s: message is required", recordScope)
		}

		if strings.TrimSpace(record.Timestamp) != "" {
			if _, err := parseRecordTimestamp(record.Timestamp); err != nil {
				return fmt.Errorf("%s: invalid timestamp: %w", recordScope, err)
			}
		}
	}

	return nil
}

// buildOTLPLogRequest converts component configuration into an OTLP logs request.
func buildOTLPLogRequest(config SendLogEventConfiguration, scope string) (OTLPLogsRequest, string, error) {
	serviceName := strings.TrimSpace(config.ServiceName)
	if serviceName == "" {
		serviceName = "superplane.workflow"
	}

	resourceAttributes := []OTLPKeyValue{
		{
			Key:   "service.name",
			Value: otlpStringValue(serviceName),
		},
	}

	logRecords := make([]OTLPLogRecord, 0, len(config.Records))
	for index, record := range config.Records {
		recordScope := fmt.Sprintf("%s: record[%d]", scope, index)
		recordTime, err := parseRecordTimestamp(record.Timestamp)
		if err != nil {
			return OTLPLogsRequest{}, "", fmt.Errorf("%s: parse timestamp: %w", recordScope, err)
		}

		severityText, severityNumber := normalizeSeverity(record.Severity)
		attributes := make([]OTLPKeyValue, 0, len(record.Attributes))
		for key, value := range record.Attributes {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			attributes = append(attributes, OTLPKeyValue{
				Key:   trimmedKey,
				Value: otlpAnyValue(value),
			})
		}

		logRecords = append(logRecords, OTLPLogRecord{
			TimeUnixNano:   strconv.FormatInt(recordTime.UnixNano(), 10),
			SeverityText:   severityText,
			SeverityNumber: severityNumber,
			Body:           otlpStringValue(record.Message),
			Attributes:     attributes,
		})
	}

	request := OTLPLogsRequest{
		ResourceLogs: []OTLPResourceLogs{
			{
				Resource: OTLPResource{
					Attributes: resourceAttributes,
				},
				ScopeLogs: []OTLPScopeLogs{
					{
						Scope: OTLPScope{
							Name: "superplane.workflow",
						},
						LogRecords: logRecords,
					},
				},
			},
		},
	}

	return request, serviceName, nil
}

// parseRecordTimestamp parses RFC3339 or unix-like timestamps into UTC time.
func parseRecordTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "nil") || strings.EqualFold(trimmed, "<nil>") {
		return time.Now().UTC(), nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	intValue, intErr := strconv.ParseInt(trimmed, 10, 64)
	if intErr != nil {
		floatValue, floatErr := strconv.ParseFloat(trimmed, 64)
		if floatErr != nil {
			return time.Time{}, fmt.Errorf("unsupported timestamp format %q", trimmed)
		}
		intValue = int64(floatValue)
	}

	switch {
	case intValue >= 1_000_000_000_000_000_000:
		return time.Unix(0, intValue).UTC(), nil
	case intValue >= 1_000_000_000_000:
		return time.UnixMilli(intValue).UTC(), nil
	default:
		return time.Unix(intValue, 0).UTC(), nil
	}
}

// normalizeSeverity maps input severity values to OTLP text and numeric levels.
func normalizeSeverity(value string) (string, int) {
	trimmed := strings.TrimSpace(strings.ToUpper(value))
	if trimmed == "WARNING" {
		trimmed = "WARN"
	}
	if trimmed == "" {
		trimmed = "INFO"
	}

	severityNumber, ok := severityNumberByText[trimmed]
	if !ok {
		return "INFO", severityNumberByText["INFO"]
	}

	return trimmed, severityNumber
}

// otlpStringValue wraps a string into OTLP AnyValue format.
func otlpStringValue(value string) OTLPAnyValue {
	normalized := value
	return OTLPAnyValue{
		StringValue: &normalized,
	}
}

// otlpAnyValue converts generic Go values into OTLP AnyValue representations.
func otlpAnyValue(value any) OTLPAnyValue {
	switch typed := value.(type) {
	case nil:
		return otlpStringValue("")
	case string:
		return otlpStringValue(typed)
	case bool:
		boolean := typed
		return OTLPAnyValue{BoolValue: &boolean}
	case int:
		integer := strconv.Itoa(typed)
		return OTLPAnyValue{IntValue: &integer}
	case int8:
		integer := strconv.FormatInt(int64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case int16:
		integer := strconv.FormatInt(int64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case int32:
		integer := strconv.FormatInt(int64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case int64:
		integer := strconv.FormatInt(typed, 10)
		return OTLPAnyValue{IntValue: &integer}
	case uint:
		integer := strconv.FormatUint(uint64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case uint8:
		integer := strconv.FormatUint(uint64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case uint16:
		integer := strconv.FormatUint(uint64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case uint32:
		integer := strconv.FormatUint(uint64(typed), 10)
		return OTLPAnyValue{IntValue: &integer}
	case uint64:
		integer := strconv.FormatUint(typed, 10)
		return OTLPAnyValue{IntValue: &integer}
	case float32:
		float := float64(typed)
		return OTLPAnyValue{DoubleValue: &float}
	case float64:
		float := typed
		return OTLPAnyValue{DoubleValue: &float}
	case []any:
		values := make([]OTLPAnyValue, 0, len(typed))
		for _, entry := range typed {
			values = append(values, otlpAnyValue(entry))
		}
		return OTLPAnyValue{
			ArrayValue: &OTLPArray{
				Values: values,
			},
		}
	case map[string]any:
		values := make([]OTLPKeyValue, 0, len(typed))
		for key, entry := range typed {
			if strings.TrimSpace(key) == "" {
				continue
			}
			values = append(values, OTLPKeyValue{
				Key:   key,
				Value: otlpAnyValue(entry),
			})
		}
		return OTLPAnyValue{
			KvlistValue: &OTLPKVList{
				Values: values,
			},
		}
	default:
		return otlpStringValue(fmt.Sprintf("%v", typed))
	}
}
