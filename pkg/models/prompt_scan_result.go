package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	GuardrailScanPhasePostInterpolation = "post_interpolation"
	GuardrailScanPhasePreAPICall        = "pre_api_call"

	GuardrailScanDirectionInput  = "input"
	GuardrailScanDirectionOutput = "output"

	GuardrailExecutionStateAllowed        = "allowed"
	GuardrailExecutionStateBlocked        = "blocked"
	GuardrailExecutionStatePendingOverride = "pending_override"
	GuardrailExecutionStateWarned         = "warned"
)

type PromptScanResult struct {
	ID          uuid.UUID `gorm:"primaryKey;default:gen_random_uuid()"`
	OrgID       uuid.UUID `gorm:"not null"`
	WorkflowID  uuid.UUID `gorm:"not null"`
	ExecutionID uuid.UUID `gorm:"not null"`
	NodeID      string    `gorm:"not null"`

	ScanPhase     string `gorm:"not null"`
	ScanDirection string `gorm:"not null;default:'input'"`

	Provider      *string
	ComponentType *string

	RiskScore         int    `gorm:"not null;default:0"`
	EnforcementAction string `gorm:"not null"`
	ExecutionState    string `gorm:"not null"`

	Findings datatypes.JSONType[[]GuardrailFinding]

	ContentHash *string

	OverrideApproved      *bool
	OverrideApprovedBy    *uuid.UUID
	OverrideApprovedAt    *time.Time
	OverrideJustification *string
	OverrideExpiresAt     *time.Time

	ClassifierResultID *uuid.UUID

	CreatedAt time.Time
}

type GuardrailFinding struct {
	RuleID      string  `json:"rule_id"`
	Severity    string  `json:"severity"`
	Confidence  float64 `json:"confidence"`
	Category    string  `json:"category"`
	Evidence    string  `json:"evidence"`
	MatchOffset int     `json:"match_offset"`
	MatchLen    int     `json:"match_len"`
	Redacted    bool    `json:"redacted"`
	Match       string  `json:"match"`
}

func (r *PromptScanResult) TableName() string {
	return "prompt_scan_results"
}

func CreatePromptScanResult(result *PromptScanResult) error {
	return CreatePromptScanResultInTransaction(database.Conn(), result)
}

func CreatePromptScanResultInTransaction(tx *gorm.DB, result *PromptScanResult) error {
	return tx.Create(result).Error
}

func FindPromptScanResult(id uuid.UUID) (*PromptScanResult, error) {
	return FindPromptScanResultInTransaction(database.Conn(), id)
}

func FindPromptScanResultInTransaction(tx *gorm.DB, id uuid.UUID) (*PromptScanResult, error) {
	var result PromptScanResult
	err := tx.Where("id = ?", id).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func FindPromptScanResultByExecution(executionID uuid.UUID, phase string) (*PromptScanResult, error) {
	return FindPromptScanResultByExecutionInTransaction(database.Conn(), executionID, phase)
}

func FindPromptScanResultByExecutionInTransaction(tx *gorm.DB, executionID uuid.UUID, phase string) (*PromptScanResult, error) {
	var result PromptScanResult
	err := tx.
		Where("execution_id = ? AND scan_phase = ?", executionID, phase).
		First(&result).
		Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func ListPendingOverrideScanResults(orgID uuid.UUID) ([]PromptScanResult, error) {
	return ListPendingOverrideScanResultsInTransaction(database.Conn(), orgID)
}

func ListPendingOverrideScanResultsInTransaction(tx *gorm.DB, orgID uuid.UUID) ([]PromptScanResult, error) {
	var results []PromptScanResult
	err := tx.
		Where("org_id = ? AND override_approved IS NULL AND enforcement_action = ?", orgID, GuardrailEnforcementSoftBlock).
		Order("created_at DESC").
		Find(&results).
		Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func ApprovePromptScanResultOverride(tx *gorm.DB, resultID, approvedBy uuid.UUID, justification string) error {
	now := time.Now()
	approved := true
	return tx.Model(&PromptScanResult{}).
		Where("id = ?", resultID).
		Updates(map[string]any{
			"override_approved":       &approved,
			"override_approved_by":    approvedBy,
			"override_approved_at":    &now,
			"override_justification":  justification,
		}).Error
}
