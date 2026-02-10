package dash0

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

// SyntheticCheck is a normalized synthetic check descriptor for resource listings.
type SyntheticCheck struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Origin string `json:"origin,omitempty"`
}

// UpsertCheckRuleConfiguration stores check rule upsert input.
type UpsertCheckRuleConfiguration struct {
	OriginOrID string `json:"originOrId" mapstructure:"originOrId"`
	Spec       string `json:"spec" mapstructure:"spec"`
}
