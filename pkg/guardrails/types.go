package guardrails

import (
	"github.com/google/uuid"
)

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

type Category string

const (
	CategorySecret       Category = "SECRET"
	CategoryInjection    Category = "INJECTION"
	CategoryPII          Category = "PII"
	CategoryToolAbuse    Category = "TOOL_ABUSE"
	CategoryExfiltration Category = "EXFILTRATION"
)

type EnforcementMode string

const (
	EnforcementAuditOnly EnforcementMode = "audit_only"
	EnforcementWarnOnly  EnforcementMode = "warn_only"
	EnforcementSoftBlock EnforcementMode = "soft_block"
	EnforcementHardBlock EnforcementMode = "hard_block"
)

type Finding struct {
	RuleID      string   `json:"rule_id"`
	Severity    Severity `json:"severity"`
	Confidence  float64  `json:"confidence"`
	Category    Category `json:"category"`
	Evidence    string   `json:"evidence"`
	MatchOffset int      `json:"match_offset"`
	MatchLen    int      `json:"match_len"`
	Redacted    bool     `json:"redacted"`
	Match       string   `json:"match"`
}

type ScanContext struct {
	OrgID         uuid.UUID
	WorkflowID    uuid.UUID
	ExecutionID   uuid.UUID
	NodeID        string
	Provider      string
	ComponentType string
	FieldName     string
	IsSystemField bool
}

type ScanResult struct {
	ExecutionID       uuid.UUID
	Phase             string
	Findings          []Finding
	RiskScore         int
	EnforcementAction string
	ExecutionState    string
}
