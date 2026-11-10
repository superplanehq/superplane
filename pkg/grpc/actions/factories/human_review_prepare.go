package factories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type humanReviewPreparation struct {
	Decision    *models.FactoryHumanReviewDecision
	Payload     map[string]any
	AgentNeeded bool
}

// Preparation owns admission. Notifications never grant decision authority.
func prepareHumanReview(ctx context.Context, db *gorm.DB, factory *models.Factory, pr *models.FactoryPullRequest, automation *humanReviewAutomation, client humanReviewGitHub, run *models.CanvasRun, root *models.CanvasEvent) (*humanReviewPreparation, error) {
	if run == nil || root == nil || run.WorkflowID != automation.canvas.ID || root.WorkflowID != automation.canvas.ID || pr.FactoryID != factory.ID || pr.OrganizationID != factory.OrganizationID {
		return nil, errors.New("Request review context is outside the workspace")
	}
	current, _, err := client.GetPullRequest(ctx, pr.Repository, int(pr.Number))
	if err != nil {
		return nil, err
	}
	prepared := &humanReviewPreparation{}
	if current == nil {
		return nil, errors.New("GitHub did not return the Request review pull request")
	}
	if current.GetState() != "open" || current.GetMerged() {
		if _, err := applyGitHubPullRequestClosed(db, factory, pr, current.GetMerged(), nil, nil, nil); err != nil {
			return nil, err
		}
		return prepared, pr.CompleteHumanReviewReconciliation(db, pr.HumanReviewRequestID)
	}
	if err := synchronizeHumanReviewReopen(db, factory, pr, current); err != nil {
		return nil, err
	}
	if automation.settings.Enabled && current.GetDraft() {
		return prepared, nil
	}
	sha := strings.TrimSpace(current.GetHead().GetSHA())
	if sha == "" {
		return nil, errors.New("Request review pull request has no head commit")
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		fresh, err := models.LockCanvasRun(tx, automation.canvas.ID, run.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return errors.New("Request review run is busy or no longer available")
		}
		run = fresh
		if run.CancelledAt != nil || run.State == models.CanvasRunStateCancelling || run.Result == models.CanvasRunResultCancelled {
			return nil
		}
		if err := pr.RequireHumanReviewPublicationIdle(tx); err != nil {
			return err
		}
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		live, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, automation.canvas.ID)
		if err != nil {
			return err
		}
		if live.LiveVersionID == nil || *live.LiveVersionID != run.VersionID || run.VersionID != automation.version.ID {
			return nil
		}
		observed, err := pr.ObserveRevision(tx, sha)
		if err != nil {
			return err
		}
		previous, err := pr.CurrentHumanReview(tx)
		if err != nil {
			return err
		}
		if !automation.settings.Enabled {
			prepared.Decision = previous
			if previous == nil || previous.State != models.FactoryHumanReviewStateDisabled || previous.RevisionID != observed.Revision.ID || previous.ConfigurationVersion != automation.version.ID.String() {
				prepared.Decision, err = pr.NewHumanReviewDecision(tx, models.FactoryHumanReviewDecision{RevisionID: observed.Revision.ID, AssessedCommit: sha, ConfigurationVersion: automation.version.ID.String(), State: models.FactoryHumanReviewStateDisabled, CheckRunID: sameRevisionCheck(previous, sha), CheckURL: humanReviewCheckURL(previous)})
			}
			return err
		}
		source, assessment, err := pr.LatestMergeConfidenceAssessment(tx, sha)
		if err != nil {
			return err
		}
		// Delayed or foreign broadcasts cannot replace the authoritative assessment.
		matches, err := humanReviewNotificationMatches(tx, factory, root, source, pr.ID, sha)
		if err != nil || !matches {
			return err
		}
		samePolicy := previous != nil && previous.RevisionID == observed.Revision.ID && previous.ConfigurationVersion == automation.version.ID.String() && previous.State != models.FactoryHumanReviewStateDisabled
		retained := samePolicy && (retentionClearedHumanReviewSource(previous, source) || (source == nil && previous.SourceRunID == nil && len(previous.Result) > 0 && len(previous.Assessment) > 0))
		var sourceID *uuid.UUID
		if source != nil {
			sourceID = &source.ID
		}
		decision := previous
		changed := !samePolicy || (!retained && !equalHumanReviewRun(previous.SourceRunID, sourceID))
		explicit := humanReviewExplicitRerun(root, run)
		if changed || (explicit && !retained && ((previous.RunID != nil && *previous.RunID != run.ID) || previous.State == models.FactoryHumanReviewStateFailed)) {
			decision, err = pr.NewHumanReviewDecision(tx, models.FactoryHumanReviewDecision{RevisionID: observed.Revision.ID, AssessedCommit: sha, SourceRunID: sourceID, ConfigurationVersion: automation.version.ID.String(), State: models.FactoryHumanReviewStateWaitingAssessment, Assessment: assessment, CheckRunID: sameRevisionCheck(previous, sha), CheckURL: humanReviewCheckURL(previous)})
			if err != nil {
				return err
			}
		}
		prepared.Decision = decision
		if retained {
			return nil
		}
		if source == nil {
			requests, err := models.ListPullRequestChecksStartRequests(tx, pr.ID)
			if err != nil {
				return err
			}
			if len(requests) == 0 {
				return pr.RequestVerificationChecks(tx)
			}
			return nil
		}
		if source.State != models.CanvasRunStateFinished {
			return nil
		}
		var report mergeconfidence.ReportData
		encoded, err := json.Marshal(assessment)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(encoded, &report); err != nil {
			return err
		}
		if source.Result != models.CanvasRunResultPassed || !validHumanReviewAssessment(report, pr.ID, sha) {
			decision.State = models.FactoryHumanReviewStateFailed
			decision.BlockingReason = "Merge confidence failed or did not produce a complete assessment. Run merge confidence again."
			return decision.Save(tx)
		}
		// A failed or interrupted agent needs an explicit rerun, never scheduled retry.
		if len(decision.Result) > 0 || (decision.RunID != nil && *decision.RunID != run.ID) || decision.State == models.FactoryHumanReviewStateFailed {
			return nil
		}
		if run.State == models.CanvasRunStateCancelling || run.Result == models.CanvasRunResultCancelled || run.CancelledAt != nil {
			return nil
		}
		if err := pr.AdmitHumanReviewRun(tx, automation.canvas, run); err != nil {
			if !models.IsPermanentSubscriptionAdmissionError(err) {
				return err
			}
			decision.State = models.FactoryHumanReviewStateFailed
			decision.BlockingReason = err.Error()
			return decision.Save(tx)
		}
		decision.Assessment = assessment
		decision.RunID = &run.ID
		decision.State = models.FactoryHumanReviewStateWaitingDecision
		if err := decision.Save(tx); err != nil {
			return err
		}
		prepared.AgentNeeded = true
		return nil
	})
	if err != nil || !prepared.AgentNeeded {
		return prepared, err
	}
	contextData, err := gatherHumanReviewContext(ctx, client, pr, current, automation.settings)
	if err != nil {
		decision := prepared.Decision
		cause := err
		saveErr := db.Transaction(func(tx *gorm.DB) error {
			if err := pr.LockForUpdate(tx); err != nil {
				return err
			}
			current, err := pr.CurrentHumanReview(tx)
			if err != nil || current == nil || current.ID != decision.ID || current.RunID == nil || *current.RunID != run.ID {
				return err
			}
			current.State = models.FactoryHumanReviewStateWaitingDecision
			current.BlockingReason = "Request review could not start. " + cause.Error()
			current.RunID = nil
			prepared.Decision = current
			return current.Save(tx)
		})
		prepared.AgentNeeded = false
		return prepared, saveErr
	}
	order, err := factory.FindWorkOrder(db, pr.WorkOrderID)
	if err != nil {
		return nil, err
	}
	payload, err := pullRequestChecksEventData(pr, order, current)
	if err != nil {
		return nil, err
	}
	decision := prepared.Decision
	contextData["decisionId"] = decision.ID.String()
	contextData["pullRequestId"] = pr.ID.String()
	contextData["assessedCommit"] = decision.AssessedCommit
	contextData["sourceRunId"] = decision.SourceRunID.String()
	contextData["assessment"] = decision.Assessment
	payload["humanReview"] = contextData
	prepared.Payload = payload
	return prepared, nil
}

func humanReviewExplicitRerun(root *models.CanvasEvent, run *models.CanvasRun) bool {
	envelope, _ := root.Data.Data().(map[string]any)
	if _, ok := envelope["replayedFromEventId"].(string); ok {
		return true
	}
	// Child runs include scheduled recovery. Only direct invocation can retry a decision.
	return root.NodeID == models.HumanReviewTriggerNodeID && envelope["type"] == "app.invocation" && run.ParentRunID == nil
}

func humanReviewNotificationMatches(db *gorm.DB, factory *models.Factory, root *models.CanvasEvent, source *models.CanvasRun, prID uuid.UUID, sha string) (bool, error) {
	if source != nil {
		canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, source.WorkflowID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if canvas.OrganizationID != factory.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
			return false, nil
		}
	}
	envelope, _ := root.Data.Data().(map[string]any)
	if envelope["type"] != "app.broadcast" {
		return true, nil
	}
	data, _ := envelope["data"].(map[string]any)
	app, _ := data["app"].(map[string]any)
	run, _ := data["run"].(map[string]any)
	payload, _ := data["payload"].(map[string]any)
	if source == nil || app["id"] != source.WorkflowID.String() || run["id"] != source.ID.String() || payload["pullRequestId"] != prID.String() || payload["assessedCommit"] != sha {
		return false, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, source.WorkflowID)
	if err != nil {
		return false, err
	}
	return canvas.OrganizationID == factory.OrganizationID && canvas.FactoryID != nil && *canvas.FactoryID == factory.ID, nil
}

func refreshHumanReviewGate(ctx context.Context, db *gorm.DB, factory *models.Factory, pr *models.FactoryPullRequest, automation *humanReviewAutomation, client humanReviewGitHub) error {
	requestID := pr.HumanReviewRequestID
	err := pr.WithHumanReviewPublicationLease(db, func(db *gorm.DB) error {
		if err := pr.LockForUpdate(db); err != nil {
			return err
		}
		decision, err := pr.CurrentHumanReview(db)
		if err != nil || decision == nil {
			return err
		}
		if decision.RunID != nil && decision.State != models.FactoryHumanReviewStateFailed {
			run, err := models.FindUnscopedCanvasRun(db, *decision.RunID)
			if err != nil {
				return err
			}
			cancelled := run.CancelledAt != nil || run.State == models.CanvasRunStateCancelling || run.Result == models.CanvasRunResultCancelled
			if cancelled {
				err := db.Transaction(func(tx *gorm.DB) error {
					if err := pr.LockForUpdate(tx); err != nil {
						return err
					}
					fresh, err := pr.CurrentHumanReview(tx)
					if err != nil || fresh == nil || fresh.ID != decision.ID {
						return err
					}
					decision = fresh
					decision.State = models.FactoryHumanReviewStateFailed
					decision.BlockingReason = "The Request review run was cancelled. Rerun Request review."
					return decision.Save(tx)
				})
				if err != nil {
					return err
				}
			}
			if !cancelled && decision.State == models.FactoryHumanReviewStateWaitingDecision {
				events, err := models.HumanReviewCompletionEvents(db, run.ID, automation.settings.AgentNodeID)
				if err != nil {
					return err
				}
				for _, event := range events {
					agent, err := models.FindNodeExecutionInTransaction(db, automation.canvas.ID, *event.ExecutionID)
					if err != nil {
						return err
					}
					var recorded *models.FactoryHumanReviewDecision
					err = db.Transaction(func(tx *gorm.DB) error {
						var err error
						recorded, err = models.RecordHumanReviewDecision(tx, automation.canvas, agent, &event)
						return err
					})
					if err != nil {
						return err
					}
					if recorded != nil {
						decision = recorded
						break
					}
				}
			}
		}
		active, err := humanReviewPublicationCurrent(db, pr, decision)
		if err != nil || !active {
			return err
		}
		current, _, err := client.GetPullRequest(ctx, pr.Repository, int(pr.Number))
		if err != nil {
			return err
		}
		if current == nil || current.GetState() != "open" || current.GetMerged() || current.GetHead().GetSHA() != decision.AssessedCommit {
			return nil
		}
		switch decision.State {
		case models.FactoryHumanReviewStateDisabled:
			return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "success", Reason: "Request review enforcement is disabled."})
		case models.FactoryHumanReviewStateWaitingAssessment:
			return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "queued", Reason: "Waiting for merge confidence."})
		case models.FactoryHumanReviewStateWaitingDecision:
			if decision.RunID == nil && decision.BlockingReason != "" {
				return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason})
			}
			if decision.RunID != nil {
				run, err := models.FindUnscopedCanvasRun(db, *decision.RunID)
				if err != nil {
					return err
				}
				if run.State == models.CanvasRunStateFinished || run.State == models.CanvasRunStateCancelling || run.CancelledAt != nil {
					decision, err = failHumanReviewWithoutResult(db, pr, decision)
					if err != nil {
						return err
					}
				}
			}
			if decision.State == models.FactoryHumanReviewStateWaitingDecision {
				return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "in_progress", Reason: "Waiting for the Request review decision."})
			}
		}
		if decision.State == models.FactoryHumanReviewStateFailed {
			return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason})
		}
		if len(decision.Result) == 0 {
			return nil
		}
		return evaluateHumanReviewDecision(ctx, db, client, pr, current, automation, decision)
	})
	if err != nil {
		return err
	}
	return pr.CompleteHumanReviewReconciliation(db, requestID)
}
