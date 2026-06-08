package guardrails

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// GuardrailBlockedError is returned by ScanConfiguration when enforcement mode
// dictates that execution must not proceed.
type GuardrailBlockedError struct {
	ScanResultID uuid.UUID
	Severity     Severity
	Enforcement  EnforcementMode
}

func (e *GuardrailBlockedError) Error() string {
	return fmt.Sprintf("guardrail block: scan_result=%s severity=%s enforcement=%s",
		e.ScanResultID, e.Severity, e.Enforcement)
}

// ScanOutcome carries the result of a non-blocking guardrail scan.
// Callers should inspect Warned to decide whether to annotate the execution.
type ScanOutcome struct {
	State         string    // models.GuardrailExecutionState* constant
	ScanResultID  uuid.UUID
	RiskScore     int
	FindingsCount int
	Warned        bool
}

// Interceptor scans resolved prompt fields and persists scan results.
// In audit_only mode it always returns (outcome, nil). In warn_only mode it
// returns (outcome, nil) with outcome.Warned=true when findings exist.
// Hard/soft blocks return (nil, *GuardrailBlockedError).
type Interceptor struct {
	scanner *Scanner
}

// NewInterceptor returns an Interceptor backed by the default rule set.
func NewInterceptor() *Interceptor {
	return &Interceptor{scanner: NewScanner(NewEngine(DefaultRules()))}
}

// ScanConfiguration scans all prompt fields in a resolved configuration map.
// It uses the field schema to identify which fields to scan, then persists
// a single PromptScanResult aggregating all findings.
//
// Returns (outcome, nil) for allowed/warned executions.
// Returns (nil, *GuardrailBlockedError) when execution must stop.
func (i *Interceptor) ScanConfiguration(
	tx *gorm.DB,
	resolvedConfig map[string]any,
	fields []configuration.Field,
	ctx ScanContext,
) (*ScanOutcome, error) {
	// Collect prompt-field contents.
	var promptContent string
	var promptFields []configuration.Field
	for _, field := range fields {
		if !field.PromptField && !field.SystemPromptField {
			continue
		}
		val, ok := resolvedConfig[field.Name]
		if !ok {
			continue
		}
		strVal, ok := val.(string)
		if !ok || strVal == "" {
			continue
		}
		promptContent += strVal + "\n"
		promptFields = append(promptFields, field)
	}

	if len(promptFields) == 0 {
		return &ScanOutcome{State: models.GuardrailExecutionStateAllowed}, nil
	}

	// Idempotency: if a scan result already exists for this execution, reuse it.
	existing, err := models.FindPromptScanResultByExecutionInTransaction(tx, ctx.ExecutionID, models.GuardrailScanPhasePostInterpolation)
	if err == nil && existing != nil {
		return enforceExisting(existing)
	}

	policy, err := models.ResolvePromptGuardrailPolicy(tx, ctx.OrgID, ctx.WorkflowID, ctx.NodeID, ctx.ComponentType)
	if err != nil {
		log.Warnf("guardrail: failed to resolve policy for org=%s: %v", ctx.OrgID, err)
		policy = defaultAuditPolicy(ctx.OrgID)
	}

	// Scan the combined prompt content. A single scan result covers all prompt
	// fields in the execution so that the risk score is computed holistically.
	result := i.scanner.Scan(promptContent, ctx, policy)

	if persistErr := models.CreatePromptScanResultInTransaction(tx, result); persistErr != nil {
		// Non-fatal: a scan persistence failure must not block execution.
		log.Errorf("guardrail: failed to persist scan result for execution=%s: %v", ctx.ExecutionID, persistErr)
		return &ScanOutcome{State: models.GuardrailExecutionStateAllowed}, nil
	}

	scheduleClassifierJob(tx, result, policy)
	logScanResult(ctx, result)

	return enforceResult(result)
}

func logScanResult(ctx ScanContext, result *models.PromptScanResult) {
	fields := log.Fields{
		"execution_id":       ctx.ExecutionID,
		"node_id":            ctx.NodeID,
		"risk_score":         result.RiskScore,
		"enforcement_action": result.EnforcementAction,
		"findings":           len(result.Findings.Data()),
	}

	if result.ExecutionState == models.GuardrailExecutionStateWarned {
		log.WithFields(fields).Warn("guardrail: warn_only — prompt findings detected, execution continues")
		for _, f := range result.Findings.Data() {
			log.WithFields(log.Fields{
				"execution_id": ctx.ExecutionID,
				"node_id":      ctx.NodeID,
				"rule_id":      f.RuleID,
				"severity":     f.Severity,
				"category":     f.Category,
				"evidence":     f.Evidence,
			}).Warn("guardrail: finding")
		}
		return
	}

	log.WithFields(fields).Info("guardrail: scan complete")
}

func enforceExisting(existing *models.PromptScanResult) (*ScanOutcome, error) {
	outcome := &ScanOutcome{
		State:         existing.ExecutionState,
		ScanResultID:  existing.ID,
		RiskScore:     existing.RiskScore,
		FindingsCount: len(existing.Findings.Data()),
		Warned:        existing.ExecutionState == models.GuardrailExecutionStateWarned,
	}

	switch existing.EnforcementAction {
	case models.GuardrailEnforcementHardBlock:
		return nil, &GuardrailBlockedError{
			ScanResultID: existing.ID,
			Severity:     SeverityCritical,
			Enforcement:  EnforcementHardBlock,
		}
	case models.GuardrailEnforcementSoftBlock:
		approved := existing.OverrideApproved
		if approved == nil || !*approved {
			return nil, &GuardrailBlockedError{
				ScanResultID: existing.ID,
				Severity:     SeverityHigh,
				Enforcement:  EnforcementSoftBlock,
			}
		}
		return outcome, nil
	default:
		return outcome, nil
	}
}

func enforceResult(result *models.PromptScanResult) (*ScanOutcome, error) {
	outcome := &ScanOutcome{
		State:         result.ExecutionState,
		ScanResultID:  result.ID,
		RiskScore:     result.RiskScore,
		FindingsCount: len(result.Findings.Data()),
		Warned:        result.ExecutionState == models.GuardrailExecutionStateWarned,
	}

	switch result.EnforcementAction {
	case models.GuardrailEnforcementHardBlock:
		return nil, &GuardrailBlockedError{
			ScanResultID: result.ID,
			Severity:     SeverityCritical,
			Enforcement:  EnforcementHardBlock,
		}
	case models.GuardrailEnforcementSoftBlock:
		return nil, &GuardrailBlockedError{
			ScanResultID: result.ID,
			Severity:     SeverityHigh,
			Enforcement:  EnforcementSoftBlock,
		}
	default:
		return outcome, nil
	}
}

// scheduleClassifierJob enqueues an async LLM classification job for the scan
// result when the policy has ClassifierEnabled=true and the sampling dice rolls
// in. Failures are non-fatal — they never block execution.
func scheduleClassifierJob(tx *gorm.DB, result *models.PromptScanResult, policy *models.PromptGuardrailPolicy) {
	if !policy.ClassifierEnabled {
		return
	}
	if !shouldSample(policy.ClassifierSamplingRate) {
		return
	}
	now := time.Now()
	job := &models.PromptClassifierResult{
		ScanResultID: result.ID,
		Status:       models.ClassifierStatusPending,
		SubmittedAt:  now,
	}
	if err := models.CreatePromptClassifierResultInTransaction(tx, job); err != nil {
		log.Warnf("guardrail: failed to schedule classifier job for scan=%s: %v", result.ID, err)
		return
	}
	log.Infof("guardrail: classifier job scheduled for scan=%s", result.ID)
}

// shouldSample returns true with probability rate (0.0–1.0).
func shouldSample(rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1.0 {
		return true
	}
	return rand.Float64() < rate
}

func defaultAuditPolicy(orgID uuid.UUID) *models.PromptGuardrailPolicy {
	return &models.PromptGuardrailPolicy{
		OrgID:                   orgID,
		EnforcementMode:         models.GuardrailEnforcementAuditOnly,
		SoftBlockScoreThreshold: 70,
		HardBlockScoreThreshold: 90,
		SoftBlockTimeoutSeconds: 86400,
		ClassifierSensitivity:   models.GuardrailClassifierSensitivityBalanced,
		ClassifierSamplingRate:  1.0,
	}
}
