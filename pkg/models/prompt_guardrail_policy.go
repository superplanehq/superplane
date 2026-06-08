package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	GuardrailEnforcementAuditOnly  = "audit_only"
	GuardrailEnforcementWarnOnly   = "warn_only"
	GuardrailEnforcementSoftBlock  = "soft_block"
	GuardrailEnforcementHardBlock  = "hard_block"

	GuardrailClassifierSensitivityStrict   = "strict"
	GuardrailClassifierSensitivityBalanced = "balanced"
	GuardrailClassifierSensitivityLenient  = "lenient"
)

type PromptGuardrailPolicy struct {
	ID             uuid.UUID  `gorm:"primaryKey;default:gen_random_uuid()"`
	OrgID          uuid.UUID  `gorm:"not null"`
	WorkflowID     *uuid.UUID
	NodeID         *string
	ComponentType  *string

	EnforcementMode string `gorm:"not null;default:'audit_only'"`

	RuleOverrides datatypes.JSONType[map[string]any]

	SoftBlockScoreThreshold int `gorm:"not null;default:70"`
	HardBlockScoreThreshold int `gorm:"not null;default:90"`

	ClassifierEnabled              bool    `gorm:"not null;default:false"`
	ClassifierRequiredForRelease   bool    `gorm:"not null;default:false"`
	ClassifierSamplingRate         float64 `gorm:"not null;default:1.0"`
	ClassifierSensitivity          string  `gorm:"not null;default:'balanced'"`

	SoftBlockTimeoutSeconds int `gorm:"not null;default:86400"`

	ProviderPolicies datatypes.JSONType[map[string]any]

	CreatedBy uuid.UUID  `gorm:"not null"`
	UpdatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *PromptGuardrailPolicy) TableName() string {
	return "prompt_guardrail_policies"
}

func FindPromptGuardrailPolicy(orgID uuid.UUID) (*PromptGuardrailPolicy, error) {
	return FindPromptGuardrailPolicyInTransaction(database.Conn(), orgID)
}

func FindPromptGuardrailPolicyInTransaction(tx *gorm.DB, orgID uuid.UUID) (*PromptGuardrailPolicy, error) {
	var policy PromptGuardrailPolicy
	err := tx.
		Where("org_id = ? AND workflow_id IS NULL AND node_id IS NULL AND component_type IS NULL", orgID).
		First(&policy).
		Error
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

func FindPromptGuardrailPolicyForWorkflow(orgID, workflowID uuid.UUID) (*PromptGuardrailPolicy, error) {
	return FindPromptGuardrailPolicyForWorkflowInTransaction(database.Conn(), orgID, workflowID)
}

func FindPromptGuardrailPolicyForWorkflowInTransaction(tx *gorm.DB, orgID, workflowID uuid.UUID) (*PromptGuardrailPolicy, error) {
	var policy PromptGuardrailPolicy
	err := tx.
		Where("org_id = ? AND workflow_id = ? AND node_id IS NULL AND component_type IS NULL", orgID, workflowID).
		First(&policy).
		Error
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

func ResolvePromptGuardrailPolicy(tx *gorm.DB, orgID, workflowID uuid.UUID, nodeID, componentType string) (*PromptGuardrailPolicy, error) {
	candidates := []struct {
		workflowID    *uuid.UUID
		nodeID        *string
		componentType *string
	}{
		{&workflowID, &nodeID, &componentType},
		{&workflowID, &nodeID, nil},
		{&workflowID, nil, &componentType},
		{&workflowID, nil, nil},
		{nil, nil, nil},
	}

	for _, c := range candidates {
		query := tx.Where("org_id = ?", orgID)
		if c.workflowID != nil {
			query = query.Where("workflow_id = ?", *c.workflowID)
		} else {
			query = query.Where("workflow_id IS NULL")
		}
		if c.nodeID != nil {
			query = query.Where("node_id = ?", *c.nodeID)
		} else {
			query = query.Where("node_id IS NULL")
		}
		if c.componentType != nil {
			query = query.Where("component_type = ?", *c.componentType)
		} else {
			query = query.Where("component_type IS NULL")
		}

		var policy PromptGuardrailPolicy
		err := query.First(&policy).Error
		if err == nil {
			return &policy, nil
		}
	}

	return defaultPolicy(orgID), nil
}

func defaultPolicy(orgID uuid.UUID) *PromptGuardrailPolicy {
	return &PromptGuardrailPolicy{
		OrgID:                   orgID,
		EnforcementMode:         GuardrailEnforcementAuditOnly,
		SoftBlockScoreThreshold: 70,
		HardBlockScoreThreshold: 90,
		SoftBlockTimeoutSeconds: 86400,
		ClassifierSensitivity:   GuardrailClassifierSensitivityBalanced,
		ClassifierSamplingRate:  1.0,
	}
}

func UpsertDefaultGuardrailPolicyInTransaction(tx *gorm.DB, orgID, createdBy uuid.UUID) error {
	existing, err := FindPromptGuardrailPolicyInTransaction(tx, orgID)
	if err == nil && existing != nil {
		return nil
	}

	policy := defaultPolicy(orgID)
	policy.CreatedBy = createdBy
	return tx.Create(policy).Error
}

// CreateWorkflowGuardrailPolicy persists a new workflow-level policy.
func CreateWorkflowGuardrailPolicy(orgID, workflowID, createdBy uuid.UUID, policy PromptGuardrailPolicy) error {
	return database.Conn().Create(&policy).Error
}

// UpdateWorkflowGuardrailPolicy applies a partial update to an existing policy by ID.
func UpdateWorkflowGuardrailPolicy(policyID, updatedBy uuid.UUID, updates map[string]any) error {
	updates["updated_by"] = updatedBy
	return database.Conn().Model(&PromptGuardrailPolicy{}).
		Where("id = ?", policyID).
		Updates(updates).Error
}

// UpsertWarnOnlyPolicyInTransaction sets the org-level policy to warn_only.
// Creates the policy if it doesn't exist; updates the enforcement_mode if it does.
// Use this to promote internal or staging orgs to the warn_only tier.
func UpsertWarnOnlyPolicyInTransaction(tx *gorm.DB, orgID, createdBy uuid.UUID) error {
	existing, err := FindPromptGuardrailPolicyInTransaction(tx, orgID)
	if err == nil && existing != nil {
		return tx.Model(existing).Update("enforcement_mode", GuardrailEnforcementWarnOnly).Error
	}

	policy := defaultPolicy(orgID)
	policy.EnforcementMode = GuardrailEnforcementWarnOnly
	policy.CreatedBy = createdBy
	return tx.Create(policy).Error
}
