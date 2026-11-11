package models

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RecordHumanReviewDecision accepts only the stored completion emitted by the
// configured agent execution in the run that preparation authorized.
func RecordHumanReviewDecision(tx *gorm.DB, canvas *Canvas, execution *CanvasNodeExecution, event *CanvasEvent) (*FactoryHumanReviewDecision, error) {
	if canvas == nil || execution == nil || event == nil {
		return nil, nil
	}
	canvas, err := FindCanvasWithoutOrgScopeInTransaction(tx, canvas.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil || canvas.FactoryID == nil {
		return nil, err
	}
	// Cancellation locks the run before its executions. Use the same order and
	// reload both records so cancellation cannot race a stale caller snapshot.
	run, err := LockCanvasRun(tx, canvas.ID, execution.RunID)
	if err != nil || run == nil {
		return nil, err
	}
	if run.CancelledAt != nil || run.State == CanvasRunStateCancelling || run.Result == CanvasRunResultCancelled {
		return nil, nil
	}
	execution, err = LockNodeExecution(tx, canvas.ID, execution.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if execution.RunID != run.ID || execution.State != CanvasNodeExecutionStateFinished || execution.Result == CanvasNodeExecutionResultCancelled || execution.CancelledAt != nil {
		return nil, nil
	}
	event, err = FindCanvasEventForCanvas(tx, canvas.ID, event.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if event.ExecutionID == nil || *event.ExecutionID != execution.ID || event.RunID != run.ID || event.NodeID != execution.NodeID || (event.Channel != "passed" && event.Channel != "failed") {
		return nil, nil
	}
	decision, err := FindFactoryHumanReviewDecisionByRun(tx, run.ID)
	if err != nil || decision == nil {
		return nil, err
	}
	factory, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	pr, err := factory.FindPullRequest(tx, FactoryPullRequestLookup{ID: decision.PullRequestID})
	if err != nil {
		return nil, err
	}
	if err := pr.LockForUpdate(tx); err != nil {
		return nil, err
	}
	current, err := pr.CurrentHumanReview(tx)
	if err != nil {
		return nil, err
	}
	if current == nil || current.ID != decision.ID || current.State != FactoryHumanReviewStateWaitingDecision || pr.CurrentRevisionID == nil || *pr.CurrentRevisionID != current.RevisionID || pr.State != FactoryPullRequestStateOpen {
		return nil, nil
	}
	live, err := FindCanvasWithoutOrgScopeInTransaction(tx, canvas.ID)
	if err != nil {
		return nil, err
	}
	if live.FactoryID == nil || *live.FactoryID != current.FactoryID || live.OrganizationID != current.OrganizationID || live.LiveVersionID == nil || live.LiveVersionID.String() != current.ConfigurationVersion || *live.LiveVersionID != run.VersionID {
		return nil, nil
	}
	version, err := FindCanvasVersionInTransaction(tx, canvas.ID, run.VersionID)
	if err != nil {
		return nil, err
	}
	settings, recognized := HumanReviewSettingsFromNodes(version.Nodes)
	if !recognized || !settings.Enabled || settings.AgentNodeID != execution.NodeID {
		return nil, nil
	}
	source, assessment, err := pr.LatestMergeConfidenceAssessment(tx, current.AssessedCommit)
	if err != nil {
		return nil, err
	}
	if source == nil || current.SourceRunID == nil || source.ID != *current.SourceRunID || source.State != CanvasRunStateFinished || source.Result != CanvasRunResultPassed {
		return nil, nil
	}
	sourceCanvas, err := FindCanvasWithoutOrgScopeInTransaction(tx, source.WorkflowID)
	if err != nil {
		return nil, err
	}
	if sourceCanvas.OrganizationID != canvas.OrganizationID || sourceCanvas.FactoryID == nil || *sourceCanvas.FactoryID != *canvas.FactoryID {
		return nil, nil
	}
	encodedAssessment, err := json.Marshal(assessment)
	if err != nil {
		return nil, err
	}
	var report mergeconfidence.ReportData
	if json.Unmarshal(encodedAssessment, &report) != nil || report.PullRequestID != pr.ID.String() || !humanreview.ValidAssessment(report, current.AssessedCommit) {
		return nil, nil
	}
	envelope, _ := objectMap(event.Data.Data())
	payloadType, _ := envelope["type"].(string)
	component := strings.TrimSuffix(payloadType, ".finished")
	if !IsAgentRunnerComponent(component) || component == payloadType {
		return nil, nil
	}
	matched := false
	for _, node := range version.Nodes {
		if node.ID == execution.NodeID && node.ComponentName() == component {
			matched = true
			break
		}
	}
	if !matched || (execution.FrozenComponentName() != "" && execution.FrozenComponentName() != component) {
		return nil, nil
	}
	var completions int64
	if err := tx.Model(&CanvasEvent{}).Where("execution_id = ? AND channel IN ?", execution.ID, []string{"passed", "failed"}).Count(&completions).Error; err != nil {
		return nil, err
	}
	current.State = FactoryHumanReviewStateFailed
	current.BlockingReason = "The Request review agent failed or returned an invalid decision. Rerun Request review."
	if completions == 1 && event.Channel == "passed" && execution.Result == CanvasNodeExecutionResultPassed {
		payload, _ := objectMap(envelope["data"])
		result, _ := objectMap(payload["result"])
		encoded, marshalErr := json.Marshal(result["humanReview"])
		var output humanreview.Decision
		if marshalErr == nil && json.Unmarshal(encoded, &output) == nil && humanreview.ValidateDecision(output, current.AssessedCommit) == nil {
			var raw map[string]any
			if err := json.Unmarshal(encoded, &raw); err != nil {
				return nil, err
			}
			current.Result = datatypes.JSONMap(raw)
			current.State = FactoryHumanReviewStateDecided
			current.ReviewRequired = *output.ReviewRequired
			current.Rationale = output.Rationale
			current.SelectedReviewers = datatypes.NewJSONSlice([]FactoryHumanReviewReviewer{})
			for _, reviewer := range output.SelectedReviewers {
				current.SelectedReviewers = append(current.SelectedReviewers, FactoryHumanReviewReviewer{ID: reviewer.ID, Login: reviewer.Login, Reason: reviewer.Reason})
			}
			current.BlockingReason = ""
		}
	}
	if current.State == FactoryHumanReviewStateFailed {
		if err := run.AddError(tx, current.BlockingReason, config.MaxPayloadSize()); err != nil {
			return nil, err
		}
	}
	return current, current.Save(tx)
}

func HumanReviewCompletionEvents(tx *gorm.DB, runID uuid.UUID, nodeID string) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := tx.Where("run_id = ? AND node_id = ? AND execution_id IS NOT NULL AND channel IN ?", runID, nodeID, []string{"passed", "failed"}).Order("created_at ASC, id ASC").Find(&events).Error
	return events, err
}
