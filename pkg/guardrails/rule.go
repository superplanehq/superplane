package guardrails

// Rule is the interface that every detection rule must implement.
type Rule interface {
	ID() string
	Priority() int
	AppliesToProvider(provider string) bool
	AppliesToFieldType(fieldType string) bool
	AppliesToSystemField() bool
	Evaluate(content string, ctx ScanContext) ([]Finding, error)
	DefaultSeverity() Severity
}
