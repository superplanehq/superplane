package guardrails

import (
	"crypto/sha256"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

// Scanner runs the rule engine against a fully-resolved prompt string and
// produces a PromptScanResult ready to be persisted.
type Scanner struct {
	engine *Engine
}

// NewScanner creates a Scanner backed by the provided Engine.
func NewScanner(engine *Engine) *Scanner {
	return &Scanner{engine: engine}
}

// Scan evaluates content and returns a populated PromptScanResult.
func (s *Scanner) Scan(content string, ctx ScanContext, policy *models.PromptGuardrailPolicy) *models.PromptScanResult {
	findings := s.engine.Scan(content, ctx)
	riskScore := Score(findings)

	action, execState := evaluatePolicy(riskScore, findings, policy)

	hash := contentHash(content)

	modelFindings := make([]models.GuardrailFinding, 0, len(findings))
	for _, f := range findings {
		modelFindings = append(modelFindings, models.GuardrailFinding{
			RuleID:      f.RuleID,
			Severity:    string(f.Severity),
			Confidence:  f.Confidence,
			Category:    string(f.Category),
			Evidence:    f.Evidence,
			MatchOffset: f.MatchOffset,
			MatchLen:    f.MatchLen,
			Redacted:    f.Redacted,
			Match:       f.Match,
		})
	}

	provider := ctx.Provider
	componentType := ctx.ComponentType

	return &models.PromptScanResult{
		OrgID:             ctx.OrgID,
		WorkflowID:        ctx.WorkflowID,
		ExecutionID:       ctx.ExecutionID,
		NodeID:            ctx.NodeID,
		ScanPhase:         models.GuardrailScanPhasePostInterpolation,
		ScanDirection:     models.GuardrailScanDirectionInput,
		Provider:          &provider,
		ComponentType:     &componentType,
		RiskScore:         riskScore,
		EnforcementAction: action,
		ExecutionState:    execState,
		ContentHash:       &hash,
		Findings:          datatypes.NewJSONType(modelFindings),
	}
}

// contentHash returns the SHA-256 hex digest of the content.
func contentHash(content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

// evaluatePolicy maps risk score + findings to enforcement action and execution state.
// In audit_only and warn_only modes it never blocks.
func evaluatePolicy(riskScore int, findings []Finding, policy *models.PromptGuardrailPolicy) (action, state string) {
	hasCritical := false
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			hasCritical = true
			break
		}
	}

	mode := EnforcementMode(policy.EnforcementMode)

	switch mode {
	case EnforcementHardBlock:
		if hasCritical || riskScore >= policy.HardBlockScoreThreshold {
			return models.GuardrailEnforcementHardBlock, models.GuardrailExecutionStateBlocked
		}
		if riskScore >= policy.SoftBlockScoreThreshold {
			return models.GuardrailEnforcementSoftBlock, models.GuardrailExecutionStatePendingOverride
		}
		return string(EnforcementWarnOnly), models.GuardrailExecutionStateWarned

	case EnforcementSoftBlock:
		if hasCritical || riskScore >= policy.SoftBlockScoreThreshold {
			return models.GuardrailEnforcementSoftBlock, models.GuardrailExecutionStatePendingOverride
		}
		return string(EnforcementWarnOnly), models.GuardrailExecutionStateWarned

	case EnforcementWarnOnly:
		if len(findings) > 0 {
			return string(EnforcementWarnOnly), models.GuardrailExecutionStateWarned
		}
		return string(EnforcementAuditOnly), models.GuardrailExecutionStateAllowed

	default: // audit_only
		return string(EnforcementAuditOnly), models.GuardrailExecutionStateAllowed
	}
}
