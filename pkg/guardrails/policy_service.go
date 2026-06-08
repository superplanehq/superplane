package guardrails

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// UpsertOrgPolicyRequest holds the fields an admin can configure for an org.
type UpsertOrgPolicyRequest struct {
	OrgID                   uuid.UUID
	CallerID                uuid.UUID
	EnforcementMode         string
	SoftBlockScoreThreshold int
	HardBlockScoreThreshold int
	ClassifierEnabled       bool
	ClassifierSamplingRate  float64
	ClassifierSensitivity   string
	SoftBlockTimeoutSeconds int
}

// GetOrgPolicy returns the resolved org-level guardrail policy. Falls back to
// the platform default (audit_only) if no policy is recorded for this org.
func GetOrgPolicy(orgID uuid.UUID) (*models.PromptGuardrailPolicy, error) {
	policy, err := models.FindPromptGuardrailPolicy(orgID)
	if err == nil {
		return policy, nil
	}
	// No policy set — return the platform default without persisting it.
	return &models.PromptGuardrailPolicy{
		OrgID:                   orgID,
		EnforcementMode:         models.GuardrailEnforcementAuditOnly,
		SoftBlockScoreThreshold: 70,
		HardBlockScoreThreshold: 90,
		SoftBlockTimeoutSeconds: 86400,
		ClassifierSensitivity:   models.GuardrailClassifierSensitivityBalanced,
		ClassifierSamplingRate:  1.0,
	}, nil
}

// UpsertOrgPolicy creates or updates the org-level guardrail policy.
func UpsertOrgPolicy(req UpsertOrgPolicyRequest) error {
	if err := validatePolicyRequest(req); err != nil {
		return err
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		existing, err := models.FindPromptGuardrailPolicyInTransaction(tx, req.OrgID)
		if err == nil && existing != nil {
			return tx.Model(existing).Updates(map[string]any{
				"enforcement_mode":           req.EnforcementMode,
				"soft_block_score_threshold": req.SoftBlockScoreThreshold,
				"hard_block_score_threshold": req.HardBlockScoreThreshold,
				"classifier_enabled":         req.ClassifierEnabled,
				"classifier_sampling_rate":   req.ClassifierSamplingRate,
				"classifier_sensitivity":     req.ClassifierSensitivity,
				"soft_block_timeout_seconds": req.SoftBlockTimeoutSeconds,
				"updated_by":                 req.CallerID,
			}).Error
		}

		policy := &models.PromptGuardrailPolicy{
			OrgID:                   req.OrgID,
			EnforcementMode:         req.EnforcementMode,
			SoftBlockScoreThreshold: req.SoftBlockScoreThreshold,
			HardBlockScoreThreshold: req.HardBlockScoreThreshold,
			ClassifierEnabled:       req.ClassifierEnabled,
			ClassifierSamplingRate:  req.ClassifierSamplingRate,
			ClassifierSensitivity:   req.ClassifierSensitivity,
			SoftBlockTimeoutSeconds: req.SoftBlockTimeoutSeconds,
			CreatedBy:               req.CallerID,
		}
		return tx.Create(policy).Error
	})
}

// ListPendingOverrides returns scan results awaiting admin approval for this org.
func ListPendingOverrides(orgID uuid.UUID) ([]models.PromptScanResult, error) {
	return models.ListPendingOverrideScanResults(orgID)
}

// ApproveOverride grants admin approval for a soft-blocked execution.
// The GuardrailGuardianWorker will detect the approval on its next poll and
// resume the execution automatically.
func ApproveOverride(scanResultID, approvedBy uuid.UUID, justification string) error {
	if justification == "" {
		return fmt.Errorf("justification is required for override approval")
	}
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.ApprovePromptScanResultOverride(tx, scanResultID, approvedBy, justification)
	})
}

func validatePolicyRequest(req UpsertOrgPolicyRequest) error {
	validModes := map[string]bool{
		models.GuardrailEnforcementAuditOnly: true,
		models.GuardrailEnforcementWarnOnly:  true,
		models.GuardrailEnforcementSoftBlock: true,
		models.GuardrailEnforcementHardBlock: true,
	}
	if !validModes[req.EnforcementMode] {
		return fmt.Errorf("invalid enforcement_mode %q", req.EnforcementMode)
	}
	if req.SoftBlockScoreThreshold < 0 || req.SoftBlockScoreThreshold > 100 {
		return fmt.Errorf("soft_block_score_threshold must be 0–100")
	}
	if req.HardBlockScoreThreshold < 0 || req.HardBlockScoreThreshold > 100 {
		return fmt.Errorf("hard_block_score_threshold must be 0–100")
	}
	if req.HardBlockScoreThreshold < req.SoftBlockScoreThreshold {
		return fmt.Errorf("hard_block_score_threshold must be >= soft_block_score_threshold")
	}
	if req.ClassifierSamplingRate < 0 || req.ClassifierSamplingRate > 1.0 {
		return fmt.Errorf("classifier_sampling_rate must be 0.0–1.0")
	}
	validSensitivities := map[string]bool{
		models.GuardrailClassifierSensitivityStrict:   true,
		models.GuardrailClassifierSensitivityBalanced: true,
		models.GuardrailClassifierSensitivityLenient:  true,
	}
	if req.ClassifierSensitivity != "" && !validSensitivities[req.ClassifierSensitivity] {
		return fmt.Errorf("invalid classifier_sensitivity %q", req.ClassifierSensitivity)
	}
	return nil
}
