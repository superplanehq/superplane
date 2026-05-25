package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
)

const guardrailGuardianPollInterval = 30 * time.Second

// GuardrailGuardianWorker polls executions stuck in guardrail_blocked state and
// either resumes them (admin approved the override) or times them out (policy
// SoftBlockTimeoutSeconds elapsed with no decision).
type GuardrailGuardianWorker struct {
	logger *log.Entry
}

func NewGuardrailGuardianWorker() *GuardrailGuardianWorker {
	return &GuardrailGuardianWorker{
		logger: log.WithFields(log.Fields{"worker": "GuardrailGuardianWorker"}),
	}
}

func (w *GuardrailGuardianWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(guardrailGuardianPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(); err != nil {
				w.logger.Errorf("tick error: %v", err)
			}
		}
	}
}

func (w *GuardrailGuardianWorker) tick() error {
	executions, err := models.ListGuardrailBlockedExecutions()
	if err != nil {
		return err
	}

	for _, execution := range executions {
		if err := w.processExecution(execution); err != nil {
			w.logger.Errorf("error processing guardrail_blocked execution %s: %v", execution.ID, err)
		}
	}

	return nil
}

func (w *GuardrailGuardianWorker) processExecution(execution models.CanvasNodeExecution) error {
	var toResume *models.CanvasNodeExecution

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockGuardrailBlockedExecutionInTransaction(tx, execution.ID)
		if err != nil {
			// Already being processed by another worker or state has changed.
			return nil
		}

		if locked.GuardrailScanID == nil {
			w.logger.Warnf("guardrail_blocked execution %s has no scan_id; skipping", locked.ID)
			return nil
		}

		scanResult, err := models.FindPromptScanResultInTransaction(tx, *locked.GuardrailScanID)
		if err != nil {
			return err
		}

		// Admin approved the override: resume execution.
		if scanResult.OverrideApproved != nil && *scanResult.OverrideApproved {
			if err := locked.ResetGuardrailBlockInTransaction(tx); err != nil {
				return err
			}
			toResume = locked
			w.logger.WithFields(log.Fields{
				"execution_id":   locked.ID,
				"scan_result_id": scanResult.ID,
				"approved_by":    scanResult.OverrideApprovedBy,
			}).Info("guardrail: soft-block override approved — execution resumed")
			return nil
		}

		// No decision yet — check if the override window has expired.
		if locked.GuardrailBlockedAt == nil {
			return nil
		}

		policy, err := models.ResolvePromptGuardrailPolicy(
			tx,
			scanResult.OrgID,
			scanResult.WorkflowID,
			scanResult.NodeID,
			guardrailComponentType(scanResult),
		)
		if err != nil {
			w.logger.Warnf("guardrail: failed to resolve policy for execution %s: %v", locked.ID, err)
			return nil
		}

		timeout := time.Duration(policy.SoftBlockTimeoutSeconds) * time.Second
		if time.Since(*locked.GuardrailBlockedAt) < timeout {
			return nil
		}

		w.logger.WithFields(log.Fields{
			"execution_id":    locked.ID,
			"blocked_at":      locked.GuardrailBlockedAt,
			"timeout_seconds": policy.SoftBlockTimeoutSeconds,
		}).Warn("guardrail: soft-block override timed out — failing execution")

		return locked.FailInTransaction(tx, models.CanvasNodeExecutionResultReasonGuardrailTimeout, "guardrail soft-block override window expired")
	})

	if err != nil {
		return err
	}

	// Publish resume message outside the transaction so the NodeExecutor
	// can pick it up immediately without waiting for a poll cycle.
	if toResume != nil {
		messages.NewCanvasExecutionMessage(
			toResume.WorkflowID.String(),
			toResume.ID.String(),
			toResume.NodeID,
		).Publish()
	}

	return nil
}

func guardrailComponentType(r *models.PromptScanResult) string {
	if r.ComponentType != nil {
		return *r.ComponentType
	}
	return ""
}
