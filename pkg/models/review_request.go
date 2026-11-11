package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrReviewRequestSourceInvalid = errors.New("review request source is invalid")
	ErrReviewRequestSuperseded    = errors.New("review request source or attempt is superseded")
)

// ReviewRequestSource contains authoritative records for one immutable report.
type ReviewRequestSource struct {
	App         *Canvas
	Run         *CanvasRun
	PullRequest *FactoryPullRequest
	Message     map[string]any
	Report      map[string]any
}

// ValidateReviewRequestAdmission locks the current trigger after the PR, so a
// pause or source change cannot commit before a newly admitted root event.
func ValidateReviewRequestAdmission(tx *gorm.DB, canvas *Canvas, trigger *CanvasNode, message any) (*ReviewRequestSource, error) {
	source, err := ValidateReviewRequestSource(tx, canvas, trigger, message)
	if err != nil {
		return nil, err
	}
	var current CanvasNode
	if err := tx.Clauses(clause.Locking{Strength: lockingForUpdateNoKey}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, trigger.NodeID).First(&current).Error; err != nil {
		return nil, reviewRequestLookupError(err)
	}
	metadata, _ := current.Metadata.Data()[FactoryAppTemplateMetadataKey].(map[string]any)
	if current.Type != NodeTypeTrigger || current.ComponentName() != "onBroadcast" || metadata["id"] != "review-request" || current.State == CanvasNodeStateError || current.Configuration.Data()["enabled"] == false || current.Configuration.Data()["app"] != source.App.ID.String() {
		return nil, invalidReviewRequestSource("review request admission is paused or unavailable")
	}
	return source, nil
}

// ValidateReviewRequestSource verifies the broadcast against the saved source
// run. The PR lock serializes admission and decisions with new activity links.
func ValidateReviewRequestSource(tx *gorm.DB, canvas *Canvas, node *CanvasNode, message any) (*ReviewRequestSource, error) {
	envelope, ok := objectMap(message)
	if !ok || canvas == nil || canvas.FactoryID == nil || node == nil || node.WorkflowID != canvas.ID {
		return nil, invalidReviewRequestSource("workspace broadcast is required")
	}
	app, _ := objectMap(envelope["app"])
	sourceID, err := uuid.Parse(reviewRequestString(app, "id"))
	if err != nil {
		return nil, invalidReviewRequestSource("source app is required")
	}
	configured, _ := node.Configuration.Data()["app"].(string)
	if configured != sourceID.String() {
		return nil, invalidReviewRequestSource("broadcast does not match the configured source app")
	}
	payload, _ := objectMap(envelope["payload"])
	if payload["type"] != "merge-confidence.completed" {
		return nil, invalidReviewRequestSource("completed merge confidence report is required")
	}
	runID, err := uuid.Parse(reviewRequestString(payload, "reportId"))
	if err != nil {
		return nil, invalidReviewRequestSource("report identity is required")
	}
	sourceCanvas, err := FindCanvasInTransaction(tx, canvas.OrganizationID, sourceID)
	if err != nil {
		return nil, reviewRequestLookupError(err)
	}
	if sourceCanvas.FactoryID == nil || *sourceCanvas.FactoryID != *canvas.FactoryID {
		return nil, invalidReviewRequestSource("source app belongs to another workspace")
	}
	run, err := FindCanvasRunInTransaction(tx, sourceCanvas.ID, runID)
	if err != nil {
		return nil, reviewRequestLookupError(err)
	}
	if run.State == CanvasRunStateCancelling || run.Result == CanvasRunResultFailed || run.Result == CanvasRunResultCancelled || len(run.Errors) > 0 || run.CancelledAt != nil {
		return nil, invalidReviewRequestSource("source run did not succeed")
	}
	saved, err := FindCanvasVersionInTransaction(tx, sourceCanvas.ID, run.VersionID)
	if err != nil {
		return nil, reviewRequestLookupError(err)
	}
	if FactoryAppTemplateID(saved.Nodes) != "risk-score" {
		return nil, invalidReviewRequestSource("source app is not merge confidence")
	}
	live, err := FindLiveCanvasVersionByCanvasInTransaction(tx, sourceCanvas)
	if err != nil {
		return nil, reviewRequestLookupError(err)
	}
	if FactoryAppTemplateID(live.Nodes) != "risk-score" {
		return nil, invalidReviewRequestSource("source app is no longer merge confidence")
	}
	activity, err := FindPullRequestActivityByRunID(tx, run.ID)
	if err != nil {
		if errors.Is(err, ErrFactoryPullRequestActivityNotFound) {
			return nil, invalidReviewRequestSource("source run has no pull request activity")
		}
		return nil, err
	}
	pr := &FactoryPullRequest{ID: activity.PullRequestID}
	if err := pr.LockForUpdate(tx); err != nil {
		return nil, reviewRequestLookupError(err)
	}
	if pr.OrganizationID != canvas.OrganizationID || pr.FactoryID != *canvas.FactoryID || pr.Provider != FactoryPullRequestProviderGitHub || pr.State != FactoryPullRequestStateOpen {
		return nil, invalidReviewRequestSource("source pull request is unavailable")
	}
	if reviewRequestString(payload, "pullRequestId") != pr.ID.String() || reviewRequestString(payload, "repository") != pr.Repository || !reviewRequestJSONEqual(payload["pullNumber"], pr.Number) {
		return nil, invalidReviewRequestSource("source pull request identity does not match")
	}
	root, err := run.RootEvent(tx)
	if err != nil {
		return nil, reviewRequestLookupError(err)
	}
	rootPayload, _ := objectMap(RootEventSourcePayload(root.Data.Data()))
	rootPR, _ := objectMap(rootPayload["pull_request"])
	head, _ := objectMap(rootPR["head"])
	sha := reviewRequestString(payload, "reviewedCommit")
	if sha == "" || sha != reviewRequestString(head, "sha") {
		return nil, invalidReviewRequestSource("assessed commit does not match the source run")
	}
	if pr.CurrentRevisionID != nil {
		revision, err := FindPullRequestRevision(tx, *pr.CurrentRevisionID)
		if err != nil {
			return nil, err
		}
		if revision.PullRequestID != pr.ID || revision.SHA != sha {
			return nil, fmt.Errorf("%w: assessed commit is stale", ErrReviewRequestSuperseded)
		}
	}
	report, _ := objectMap(payload["mergeConfidence"])
	if report["complete"] != true || reviewRequestString(report, "pullRequestId") != pr.ID.String() || reviewRequestString(report, "repository") != pr.Repository || !reviewRequestJSONEqual(report["pullNumber"], pr.Number) || reviewRequestString(report, "reviewedCommit") != sha {
		return nil, invalidReviewRequestSource("complete immutable assessment is required")
	}
	if err := verifyReviewRequestCompletion(tx, run, saved, envelope, payload, report); err != nil {
		return nil, err
	}
	latest, err := latestReviewRequestSourceRun(tx, pr.ID, sourceCanvas.ID)
	if err != nil {
		return nil, err
	}
	if latest.ID != run.ID {
		return nil, ErrReviewRequestSuperseded
	}
	return &ReviewRequestSource{App: sourceCanvas, Run: run, PullRequest: pr, Message: envelope, Report: report}, nil
}

// CheckReviewRequestAttempt must run in the transaction that accepts a decision
// or requests reviewers. Keep the transaction open until the mutation finishes.
func CheckReviewRequestAttempt(tx *gorm.DB, canvas *Canvas, run *CanvasRun) (*ReviewRequestSource, error) {
	if canvas == nil || run == nil || run.WorkflowID != canvas.ID || run.State == CanvasRunStateCancelling || run.State == CanvasRunStateFinished || run.CancelledAt != nil {
		return nil, invalidReviewRequestSource("selection run is inactive")
	}
	root, err := run.RootEvent(tx)
	if err != nil {
		return nil, err
	}
	trigger, err := FindCanvasNode(tx, canvas.ID, root.NodeID)
	if err != nil {
		return nil, err
	}
	source, err := ValidateReviewRequestSource(tx, canvas, trigger, RootEventSourcePayload(root.Data.Data()))
	if err != nil {
		return nil, err
	}
	var latest CanvasEvent
	err = tx.Where("workflow_id = ? AND node_id = ? AND execution_id IS NULL", canvas.ID, root.NodeID).
		Where("data #>> '{data,app,id}' = ? AND data #>> '{data,payload,reportId}' = ?", source.App.ID.String(), source.Run.ID.String()).
		Order("created_at DESC, id DESC").First(&latest).Error
	if err != nil {
		return nil, err
	}
	if latest.ID != root.ID {
		return nil, ErrReviewRequestSuperseded
	}
	return source, nil
}

// ListReviewRequestDecisionEvents returns the output events of the reviewer
// selection node in the run, newest first.
func ListReviewRequestDecisionEvents(tx *gorm.DB, run *CanvasRun, selectionNodeID string) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := tx.Where("workflow_id = ? AND run_id = ? AND node_id = ? AND execution_id IS NOT NULL", run.WorkflowID, run.ID, selectionNodeID).
		Order("created_at DESC, id DESC").Find(&events).Error
	return events, err
}

// MaybeLinkReviewRequestRun gives the selection run the same task authority as
// its validated producer, before ordinary agent admission checks run.
func MaybeLinkReviewRequestRun(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	if canvas == nil || canvas.FactoryID == nil || event == nil || run == nil {
		return nil
	}
	matches, err := CanvasHasFactoryAppTemplate(tx, canvas.ID, "review-request")
	if err != nil || !matches {
		return err
	}
	trigger, err := FindCanvasNode(tx, canvas.ID, event.NodeID)
	if err != nil {
		return err
	}
	source, err := ValidateReviewRequestSource(tx, canvas, trigger, RootEventSourcePayload(event.Data.Data()))
	if err != nil {
		return err
	}
	workspace, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	order, err := workspace.FindWorkOrder(tx, source.PullRequest.WorkOrderID)
	if err != nil {
		return err
	}
	if err := order.RequireExecutionOwner(tx, true); err != nil {
		return recordColumnAppRejection(tx, order, "Review request", err)
	}
	if err := order.ValidateSubscriptionCanvasModels(tx, canvas.ID, ""); err != nil {
		return recordColumnAppRejection(tx, order, "Review request", err)
	}
	_, err = source.PullRequest.CreateActivity(tx, FactoryPullRequestActivityParams{RunID: run.ID, Title: "Review request", Access: FactoryPullRequestAccessConcurrent})
	return err
}

func verifyReviewRequestCompletion(tx *gorm.DB, run *CanvasRun, version *CanvasVersion, envelope, payload, report map[string]any) error {
	var executions []CanvasNodeExecution
	if err := tx.Where("workflow_id = ? AND run_id = ?", run.WorkflowID, run.ID).Find(&executions).Error; err != nil {
		return err
	}
	byID := map[uuid.UUID]CanvasNodeExecution{}
	for _, execution := range executions {
		if execution.Result == CanvasNodeExecutionResultFailed || execution.Result == CanvasNodeExecutionResultCancelled || execution.State == CanvasNodeExecutionStateCancelling {
			return invalidReviewRequestSource("source execution failed or was cancelled")
		}
		byID[execution.ID] = execution
	}
	nodes := map[string]Node{}
	for _, node := range version.Nodes {
		nodes[node.ID] = node
	}
	var events []CanvasEvent
	if err := tx.Where("workflow_id = ? AND run_id = ? AND execution_id IS NOT NULL", run.WorkflowID, run.ID).Find(&events).Error; err != nil {
		return err
	}
	sourceNode, _ := objectMap(envelope["node"])
	assessment, publication, broadcast := false, false, false
	for _, event := range events {
		execution := byID[*event.ExecutionID]
		if execution.Result != CanvasNodeExecutionResultPassed || execution.State != CanvasNodeExecutionStateFinished || execution.NodeID != event.NodeID {
			continue
		}
		node := nodes[event.NodeID]
		component := node.ComponentName()
		if frozen := execution.FrozenComponentName(); frozen != "" && frozen != component {
			continue
		}
		output, _ := objectMap(event.Data.Data())
		data, _ := objectMap(output["data"])
		if IsAgentRunnerComponent(component) && event.Channel == "passed" && output["type"] == component+".finished" && reviewRequestJSONEqual(data["mergeConfidence"], report) {
			assessment = true
		}
		if component == "github.updatePullRequestBodySection" && node.Configuration["sectionKey"] == "merge-confidence" && node.Configuration["enabled"] != false && (data["status"] == "published" || data["status"] == "unchanged") && execution.Configuration.Data()["expectedHeadSha"] == payload["reviewedCommit"] {
			current, err := FindCanvasNode(tx, run.WorkflowID, node.ID)
			if err != nil {
				return reviewRequestLookupError(err)
			}
			publication = current.ComponentName() == component && current.Configuration.Data()["enabled"] != false
		}
		if component == "broadcastMessage" && event.NodeID == reviewRequestString(sourceNode, "id") && output["type"] == "app.broadcast" && reviewRequestJSONEqual(data, payload) {
			broadcast = true
		}
	}
	if !assessment || !publication || !broadcast {
		return invalidReviewRequestSource("saved assessment, publication, and broadcast do not match")
	}
	return nil
}

func latestReviewRequestSourceRun(tx *gorm.DB, pullRequestID, appID uuid.UUID) (*CanvasRun, error) {
	var run CanvasRun
	err := tx.Table("workflow_runs").Select("workflow_runs.*").
		Joins("JOIN factory_pull_request_runs a ON a.run_id = workflow_runs.id").
		Where("a.pull_request_id = ? AND workflow_runs.workflow_id = ?", pullRequestID, appID).
		Order("workflow_runs.created_at DESC, workflow_runs.id DESC").First(&run).Error
	return &run, err
}

func invalidReviewRequestSource(reason string) error {
	return fmt.Errorf("%w: %s", ErrReviewRequestSourceInvalid, reason)
}

func reviewRequestLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return invalidReviewRequestSource("saved source record was not found")
	}
	return err
}

func reviewRequestString(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return strings.TrimSpace(value)
}

func reviewRequestJSONEqual(left, right any) bool {
	a, leftErr := json.Marshal(left)
	b, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(a, b)
}
