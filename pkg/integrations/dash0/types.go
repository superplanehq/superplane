package dash0

import (
	"encoding/json"
	"fmt"
)

// HTTPRequestStatusError describes a non-2xx HTTP response returned by Dash0.
type HTTPRequestStatusError struct {
	Operation  string
	StatusCode int
	Body       string
}

// Error formats the HTTP status error with operation context and response body.
func (e *HTTPRequestStatusError) Error() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf("%s: request returned status %d: %s", e.Operation, e.StatusCode, e.Body)
}

// PrometheusResponse represents a standard Dash0 Prometheus API response.
type PrometheusResponse struct {
	Status string                 `json:"status"`
	Data   PrometheusResponseData `json:"data"`
}

// PrometheusResponseData holds query result metadata and values.
type PrometheusResponseData struct {
	ResultType string                  `json:"resultType"`
	Result     []PrometheusQueryResult `json:"result"`
}

// PrometheusQueryResult contains one metric series returned by PromQL.
type PrometheusQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value,omitempty"`
	Values [][]any           `json:"values,omitempty"`
}

// CheckRule is a normalized check rule descriptor for resource listings.
type CheckRule struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Origin string `json:"origin,omitempty"`
}

// SyntheticCheck is a normalized synthetic check descriptor for resource listings.
type SyntheticCheck struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Origin string `json:"origin,omitempty"`
}

// OnAlertEventConfiguration stores trigger settings for alert event filtering.
type OnAlertEventConfiguration struct {
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

// SendLogEventConfiguration stores the batch payload sent to OTLP logs ingestion.
type SendLogEventConfiguration struct {
	ServiceName string               `json:"serviceName" mapstructure:"serviceName"`
	Records     []SendLogEventRecord `json:"records" mapstructure:"records"`
}

// SendLogEventRecord represents one workflow log record in component configuration.
type SendLogEventRecord struct {
	Message    string         `json:"message" mapstructure:"message"`
	Severity   string         `json:"severity,omitempty" mapstructure:"severity"`
	Timestamp  string         `json:"timestamp,omitempty" mapstructure:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty" mapstructure:"attributes"`
}

// SendLogEventOutput is emitted after a successful log ingestion request.
type SendLogEventOutput struct {
	ServiceName string `json:"serviceName"`
	SentCount   int    `json:"sentCount"`
}

// OTLPLogsRequest is the OTLP HTTP JSON request root for log ingestion.
type OTLPLogsRequest struct {
	ResourceLogs []OTLPResourceLogs `json:"resourceLogs"`
}

// OTLPResourceLogs groups log records by resource attributes.
type OTLPResourceLogs struct {
	Resource  OTLPResource    `json:"resource"`
	ScopeLogs []OTLPScopeLogs `json:"scopeLogs"`
}

// OTLPResource identifies the emitting service and resource metadata.
type OTLPResource struct {
	Attributes []OTLPKeyValue `json:"attributes,omitempty"`
}

// OTLPScopeLogs groups records by instrumentation scope.
type OTLPScopeLogs struct {
	Scope      OTLPScope       `json:"scope"`
	LogRecords []OTLPLogRecord `json:"logRecords"`
}

// OTLPScope identifies the instrumentation scope for emitted records.
type OTLPScope struct {
	Name string `json:"name,omitempty"`
}

// OTLPLogRecord represents one OTLP log record entry.
type OTLPLogRecord struct {
	TimeUnixNano   string         `json:"timeUnixNano,omitempty"`
	SeverityNumber int            `json:"severityNumber,omitempty"`
	SeverityText   string         `json:"severityText,omitempty"`
	Body           OTLPAnyValue   `json:"body"`
	Attributes     []OTLPKeyValue `json:"attributes,omitempty"`
}

// OTLPKeyValue is a key/value pair used across OTLP resources and log attributes.
type OTLPKeyValue struct {
	Key   string       `json:"key"`
	Value OTLPAnyValue `json:"value"`
}

// OTLPAnyValue models the OTLP "AnyValue" union structure.
type OTLPAnyValue struct {
	StringValue *string     `json:"stringValue,omitempty"`
	BoolValue   *bool       `json:"boolValue,omitempty"`
	IntValue    *string     `json:"intValue,omitempty"`
	DoubleValue *float64    `json:"doubleValue,omitempty"`
	KvlistValue *OTLPKVList `json:"kvlistValue,omitempty"`
	ArrayValue  *OTLPArray  `json:"arrayValue,omitempty"`
	BytesValue  *string     `json:"bytesValue,omitempty"`
}

// OTLPKVList stores nested key/value objects inside OTLPAnyValue.
type OTLPKVList struct {
	Values []OTLPKeyValue `json:"values"`
}

// OTLPArray stores nested array values inside OTLPAnyValue.
type OTLPArray struct {
	Values []OTLPAnyValue `json:"values"`
}

// GetCheckDetailsConfiguration stores action settings for check detail retrieval.
type GetCheckDetailsConfiguration struct {
	CheckID        string `json:"checkId" mapstructure:"checkId"`
	IncludeHistory bool   `json:"includeHistory" mapstructure:"includeHistory"`
}

// UpsertSyntheticCheckConfiguration stores synthetic check upsert input.
type UpsertSyntheticCheckConfiguration struct {
	OriginOrID string `json:"originOrId" mapstructure:"originOrId"`
	Spec       string `json:"spec" mapstructure:"spec"`
}

// UpsertCheckRuleConfiguration stores check rule upsert input.
type UpsertCheckRuleConfiguration struct {
	OriginOrID string `json:"originOrId" mapstructure:"originOrId"`
	Spec       string `json:"spec" mapstructure:"spec"`
}

// AlertEventPayload is the normalized event emitted by the On Alert Event trigger.
type AlertEventPayload struct {
	EventType   string         `json:"eventType"`
	CheckID     string         `json:"checkId"`
	CheckName   string         `json:"checkName,omitempty"`
	Severity    string         `json:"severity,omitempty"`
	Labels      map[string]any `json:"labels,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Timestamp   string         `json:"timestamp"`
	Event       map[string]any `json:"event"`
}

// AlertWebhookPayload captures raw Dash0 webhook bodies before normalization.
type AlertWebhookPayload struct {
	Data map[string]any
}

// UnmarshalJSON preserves the webhook payload as a generic JSON map.
func (p *AlertWebhookPayload) UnmarshalJSON(data []byte) error {
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	p.Data = decoded
	return nil
}
