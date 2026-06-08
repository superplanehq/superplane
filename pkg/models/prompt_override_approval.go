package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	OverrideScopeExecution    = "execution"
	OverrideScopeWorkflowNode = "workflow_node"
	OverrideScopeRuleAllowlist = "rule_allowlist"
)

type PromptOverrideApproval struct {
	ID            uuid.UUID `gorm:"primaryKey;default:gen_random_uuid()"`
	ScanResultID  uuid.UUID `gorm:"not null"`
	OrgID         uuid.UUID `gorm:"not null"`

	ApprovedBy uuid.UUID `gorm:"not null"`
	ApprovedAt time.Time `gorm:"not null"`

	Scope         string `gorm:"not null;default:'execution'"`
	Justification string `gorm:"not null"`
	ExpiresAt     *time.Time

	ReviewerIP  *string
	MFAVerified bool `gorm:"not null;default:false"`
}

func (a *PromptOverrideApproval) TableName() string {
	return "prompt_override_approvals"
}

func CreatePromptOverrideApproval(approval *PromptOverrideApproval) error {
	return CreatePromptOverrideApprovalInTransaction(database.Conn(), approval)
}

func CreatePromptOverrideApprovalInTransaction(tx *gorm.DB, approval *PromptOverrideApproval) error {
	return tx.Create(approval).Error
}

func ListOverrideApprovals(orgID uuid.UUID) ([]PromptOverrideApproval, error) {
	var approvals []PromptOverrideApproval
	err := database.Conn().
		Where("org_id = ?", orgID).
		Order("approved_at DESC").
		Find(&approvals).
		Error
	if err != nil {
		return nil, err
	}

	return approvals, nil
}
