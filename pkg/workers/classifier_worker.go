package workers

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/guardrails"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	classifierWorkerPollInterval = 10 * time.Second
	classifierWorkerBatchSize    = 20
	classifierWorkerTimeout      = 30 * time.Second
)

// ClassifierWorker drains the prompt_classifier_results queue.
// It calls the configured Classifier for each pending job, then records the
// outcome. In Phase 4 the default classifier is a no-op (marks jobs "skipped").
type ClassifierWorker struct {
	classifier guardrails.Classifier
	logger     *log.Entry
}

func NewClassifierWorker(classifier guardrails.Classifier) *ClassifierWorker {
	if classifier == nil {
		classifier = guardrails.NewNoOpClassifier()
	}
	return &ClassifierWorker{
		classifier: classifier,
		logger:     log.WithFields(log.Fields{"worker": "ClassifierWorker"}),
	}
}

// NewClassifierWorkerFromEnv reads ANTHROPIC_CLASSIFIER_API_KEY (falling back to
// ANTHROPIC_API_KEY) and creates a real AnthropicClassifier when a key is present.
// Falls back to NoOpClassifier when no key is configured.
func NewClassifierWorkerFromEnv() *ClassifierWorker {
	apiKey := os.Getenv("ANTHROPIC_CLASSIFIER_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if apiKey == "" {
		log.Info("ClassifierWorker: no API key configured, using no-op classifier")
		return NewClassifierWorker(guardrails.NewNoOpClassifier())
	}

	classifier, err := guardrails.NewAnthropicClassifier(guardrails.AnthropicClassifierConfig{
		APIKey: apiKey,
		Model:  os.Getenv("ANTHROPIC_CLASSIFIER_MODEL"),
	})
	if err != nil {
		log.Warnf("ClassifierWorker: failed to create Anthropic classifier (%v), using no-op", err)
		return NewClassifierWorker(guardrails.NewNoOpClassifier())
	}

	log.Infof("ClassifierWorker: using Anthropic classifier (model=%s)", classifier.Model())
	return NewClassifierWorker(classifier)
}

func (w *ClassifierWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(classifierWorkerPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				w.logger.Errorf("tick error: %v", err)
			}
		}
	}
}

func (w *ClassifierWorker) tick(ctx context.Context) error {
	jobs, err := models.FindPendingClassifierJobs(classifierWorkerBatchSize)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if err := w.processJob(ctx, job); err != nil {
			w.logger.Errorf("error processing classifier job %s: %v", job.ID, err)
		}
	}

	return nil
}

func (w *ClassifierWorker) processJob(ctx context.Context, job models.PromptClassifierResult) error {
	if err := w.markRunning(job.ID); err != nil {
		// Another worker may have picked this up.
		return nil
	}

	classifyCtx, cancel := context.WithTimeout(ctx, classifierWorkerTimeout)
	defer cancel()

	scanResult, err := models.FindPromptScanResultInTransaction(database.Conn(), job.ScanResultID)
	if err != nil {
		return w.markFailed(job.ID, "scan_not_found", err.Error())
	}

	req := guardrails.ClassificationRequest{
		Findings: toGuardrailFindings(scanResult.Findings.Data()),
		ScanContext: guardrails.ScanContext{
			OrgID:      scanResult.OrgID,
			WorkflowID: scanResult.WorkflowID,
			NodeID:     scanResult.NodeID,
		},
		ContentHash: stringOrEmpty(scanResult.ContentHash),
	}

	startedAt := time.Now()
	result, classErr := w.classifier.Classify(classifyCtx, req)
	latencyMs := int(time.Since(startedAt).Milliseconds())

	if classErr != nil {
		return w.markFailed(job.ID, "classifier_error", classErr.Error())
	}

	if result == nil {
		// nil result = intentionally skipped (e.g., no-op classifier).
		return w.markSkipped(job.ID)
	}

	return w.markCompleted(job, result, latencyMs)
}

func (w *ClassifierWorker) markRunning(jobID interface{ String() string }) error {
	now := time.Now()
	return database.Conn().Model(&models.PromptClassifierResult{}).
		Where("id = ? AND status = ?", jobID, models.ClassifierStatusPending).
		Updates(map[string]any{
			"status":     models.ClassifierStatusRunning,
			"started_at": &now,
		}).Error
}

func (w *ClassifierWorker) markCompleted(job models.PromptClassifierResult, result *guardrails.ClassificationResult, latencyMs int) error {
	now := time.Now()
	model := result.Model
	tokenCount := result.TokenCount
	rawResponse := result.RawResponse
	riskScore := result.RiskScore

	findings := toModelFindings(result.Findings)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if updateErr := tx.Model(&models.PromptClassifierResult{}).
			Where("id = ?", job.ID).
			Updates(map[string]any{
				"status":            models.ClassifierStatusCompleted,
				"completed_at":      &now,
				"classifier_model":  &model,
				"risk_score":        &riskScore,
				"findings":          findings,
				"raw_response":      &rawResponse,
				"token_count":       &tokenCount,
				"latency_ms":        &latencyMs,
			}).Error; updateErr != nil {
			return updateErr
		}

		// Link the classifier result back to the scan result.
		return tx.Model(&models.PromptScanResult{}).
			Where("id = ?", job.ScanResultID).
			Update("classifier_result_id", job.ID).Error
	})

	if err != nil {
		w.logger.Errorf("classifier: failed to record completion for job %s: %v", job.ID, err)
		return err
	}

	w.logger.WithFields(log.Fields{
		"job_id":        job.ID,
		"scan_id":       job.ScanResultID,
		"model":         model,
		"risk_score":    riskScore,
		"latency_ms":    latencyMs,
	}).Info("classifier: job completed")

	return nil
}

func (w *ClassifierWorker) markSkipped(jobID interface{ String() string }) error {
	now := time.Now()
	return database.Conn().Model(&models.PromptClassifierResult{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       models.ClassifierStatusSkipped,
			"completed_at": &now,
		}).Error
}

func (w *ClassifierWorker) markFailed(jobID interface{ String() string }, code, message string) error {
	now := time.Now()
	return database.Conn().Model(&models.PromptClassifierResult{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":        models.ClassifierStatusFailed,
			"completed_at":  &now,
			"error_code":    &code,
			"error_message": &message,
		}).Error
}

func toGuardrailFindings(mf []models.GuardrailFinding) []guardrails.Finding {
	out := make([]guardrails.Finding, len(mf))
	for i, f := range mf {
		out[i] = guardrails.Finding{
			RuleID:      f.RuleID,
			Severity:    guardrails.Severity(f.Severity),
			Confidence:  f.Confidence,
			Category:    guardrails.Category(f.Category),
			Evidence:    f.Evidence,
			MatchOffset: f.MatchOffset,
			MatchLen:    f.MatchLen,
			Redacted:    f.Redacted,
			Match:       f.Match,
		}
	}
	return out
}

func toModelFindings(findings []guardrails.Finding) []models.GuardrailFinding {
	out := make([]models.GuardrailFinding, len(findings))
	for i, f := range findings {
		out[i] = models.GuardrailFinding{
			RuleID:      f.RuleID,
			Severity:    string(f.Severity),
			Confidence:  f.Confidence,
			Category:    string(f.Category),
			Evidence:    f.Evidence,
			MatchOffset: f.MatchOffset,
			MatchLen:    f.MatchLen,
			Redacted:    f.Redacted,
			Match:       f.Match,
		}
	}
	return out
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
