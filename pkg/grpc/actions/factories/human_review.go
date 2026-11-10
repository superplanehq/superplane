package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const humanReviewCheckName = humanreview.CheckName

type humanReviewGitHub interface {
	humanReviewCheckPublisher
	ListReviews(context.Context, string, int) ([]*github.PullRequestReview, error)
	GetRepositoryPermissionLevel(context.Context, string, string) (*github.RepositoryPermissionLevel, *github.Response, error)
	AddPullRequestReviewers(context.Context, string, int, github.ReviewersRequest) (*github.PullRequest, *github.Response, error)
	HumanReviewTeamMembers(context.Context, string, string) ([]*github.User, error)
	HumanReviewChangedFiles(context.Context, string, int) ([]*github.CommitFile, error)
	HumanReviewOwnershipFile(context.Context, string, string, string) (string, error)
	ListCommits(context.Context, string, *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
}

type humanReviewAutomation struct {
	canvas   *models.Canvas
	version  *models.CanvasVersion
	settings models.HumanReviewSettings
}

// reconcileFactoryHumanReview reads current provider state on every attempt.
// A per-PR advisory lock serializes requests, webhook refreshes, and retries.
// Pending publication stays in the decision row across worker restarts.
func reconcileFactoryHumanReview(ctx context.Context, deps IntakeDependencies, pullRequest models.FactoryPullRequest) error {
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	db := database.DB(ctx)
	factory, err := models.FindFactory(db, pullRequest.OrganizationID, pullRequest.FactoryID)
	if err != nil {
		return err
	}
	automation, err := findHumanReviewAutomation(db, factory)
	if err != nil || automation == nil {
		return err
	}

	api, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return err
	}
	client, ok := api.(humanReviewGitHub)
	if !ok {
		return errors.New("GitHub connection does not support Human review")
	}
	if err := ensureFactoryMergeabilityWebhookForRepository(ctx, db, deps, factory, pullRequest.Repository); err != nil {
		return err
	}
	return pullRequest.WithHumanReviewPublicationLock(db, func(tx *gorm.DB) error {
		if err := pullRequest.LockForUpdate(tx); err != nil {
			return err
		}
		// Reload policy after the lock. A disabled or replaced policy invalidates
		// previous agent output before any new request or check publication.
		current, err := findHumanReviewAutomation(tx, factory)
		if err != nil || current == nil {
			return err
		}
		requestID := pullRequest.HumanReviewRequestID
		if err := reconcileHumanReview(ctx, tx, deps, factory, &pullRequest, current, client); err != nil {
			return err
		}
		return pullRequest.CompleteHumanReviewReconciliation(tx, requestID)
	})
}

func findHumanReviewAutomation(db *gorm.DB, factory *models.Factory) (*humanReviewAutomation, error) {
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return nil, err
	}
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.LiveVersionID == nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		if err != nil {
			return nil, err
		}
		settings, ok := models.HumanReviewSettingsFromNodes(version.Nodes)
		if ok {
			return &humanReviewAutomation{canvas: canvas, version: version, settings: settings}, nil
		}
	}
	return nil, nil
}

func reconcileHumanReview(ctx context.Context, db *gorm.DB, deps IntakeDependencies, factory *models.Factory, pr *models.FactoryPullRequest, automation *humanReviewAutomation, client humanReviewGitHub) error {
	current, _, err := client.GetPullRequest(ctx, pr.Repository, int(pr.Number))
	if err != nil {
		return fmt.Errorf("read Human review pull request: %w", err)
	}
	if current == nil || current.GetState() != "open" || current.GetMerged() {
		return nil
	}
	if err := synchronizeHumanReviewReopen(db, factory, pr, current); err != nil {
		return err
	}
	// Run admission requires a ready PR. Wait without creating a decision so
	// the ready transition can start Human review for the same commit.
	if automation.settings.Enabled && current.GetDraft() {
		return nil
	}
	sha := strings.TrimSpace(current.GetHead().GetSHA())
	if sha == "" {
		return errors.New("Human review pull request has no head commit")
	}
	var observed *models.FactoryPullRequestObserveResult
	err = db.Transaction(func(tx *gorm.DB) error {
		var err error
		observed, err = pr.ObserveRevision(tx, sha)
		return err
	})
	if err != nil {
		return err
	}
	previous, err := pr.CurrentHumanReview(db)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if !automation.settings.Enabled {
		// The explanatory success also covers newly tracked PRs when a repository
		// already requires the gate but administrators leave enforcement disabled.
		decision := previous
		if previous == nil || previous.State != models.FactoryHumanReviewStateDisabled || previous.RevisionID != observed.Revision.ID || previous.ConfigurationVersion != automation.version.ID.String() {
			decision, err = newHumanReviewDecision(db, pr, models.FactoryHumanReviewDecision{RevisionID: observed.Revision.ID, AssessedCommit: sha, ConfigurationVersion: automation.version.ID.String(), State: models.FactoryHumanReviewStateDisabled, CheckRunID: sameRevisionCheck(previous, sha), CheckURL: humanReviewCheckURL(previous)})
			if err != nil {
				return err
			}
		}
		return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "success", Reason: "Human review enforcement is disabled."})
	}
	source, assessment, err := pr.LatestMergeConfidenceAssessment(db, sha)
	if err != nil {
		return err
	}
	var sourceID *uuid.UUID
	if source != nil {
		sourceID = &source.ID
	}
	decision := previous
	changed := decision == nil || decision.RevisionID != observed.Revision.ID || decision.ConfigurationVersion != automation.version.ID.String() || !equalHumanReviewRun(decision.SourceRunID, sourceID) || decision.State == models.FactoryHumanReviewStateDisabled
	if changed {
		decision, err = newHumanReviewDecision(db, pr, models.FactoryHumanReviewDecision{RevisionID: observed.Revision.ID, AssessedCommit: sha, SourceRunID: sourceID, ConfigurationVersion: automation.version.ID.String(), State: models.FactoryHumanReviewStateWaitingAssessment, Assessment: assessment, CheckRunID: sameRevisionCheck(previous, sha)})
		if err != nil {
			return err
		}
		if err := publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "queued", Reason: "Waiting for merge confidence."}); err != nil {
			return err
		}
	}
	// Completed decisions retain their evidence after run retention. A new
	// revision, policy, or assessment still replaces that authority above.
	retainedDecision := source == nil && !changed && len(decision.Assessment) > 0 && len(decision.Result) > 0
	if retainedDecision {
		assessment = decision.Assessment
	}
	if source == nil && !retainedDecision {
		publicationErr := publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "queued", Reason: "Waiting for merge confidence."})
		requests, err := models.ListPullRequestChecksStartRequests(db, pr.ID)
		if err != nil {
			return errors.Join(publicationErr, err)
		}
		if len(requests) == 0 {
			return errors.Join(publicationErr, pr.RequestVerificationChecks(db))
		}
		return publicationErr
	}
	if source != nil && source.State != models.CanvasRunStateFinished {
		return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "queued", Reason: "Waiting for merge confidence."})
	}
	var report mergeconfidence.ReportData
	if encoded, err := json.Marshal(assessment); err != nil {
		return err
	} else if err := json.Unmarshal(encoded, &report); err != nil {
		return err
	}
	if (source != nil && source.Result != models.CanvasRunResultPassed) || !validHumanReviewAssessment(report, pr.ID, sha) {
		previousState, previousReason := decision.State, decision.BlockingReason
		decision.State = models.FactoryHumanReviewStateFailed
		decision.BlockingReason = "Merge confidence failed or did not produce a complete assessment. Run merge confidence again."
		if err := decision.Save(db); err != nil {
			return err
		}
		if previousState != decision.State || previousReason != decision.BlockingReason {
			notifyHumanReviewDecision(db, pr, decision)
		}
		return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason})
	}
	if len(decision.Assessment) == 0 {
		decision.Assessment = assessment
		if err := decision.Save(db); err != nil {
			return err
		}
	}
	if decision.RunID == nil && len(decision.Result) == 0 {
		if err := startHumanReviewAgent(ctx, db, factory, pr, automation, client, current, decision, report); err != nil {
			decision.State = models.FactoryHumanReviewStateFailed
			decision.BlockingReason = "Human review could not start. " + err.Error()
			if saveErr := decision.Save(db); saveErr != nil {
				return errors.Join(err, saveErr)
			}
			notifyHumanReviewDecision(db, pr, decision)
			return errors.Join(err, publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason}))
		}
	}
	if decision.RunID == nil && len(decision.Result) == 0 {
		return nil
	}
	if len(decision.Result) == 0 {
		run, err := models.FindCanvasRunInTransaction(db, automation.canvas.ID, *decision.RunID)
		if err != nil {
			return err
		}
		if run.State != models.CanvasRunStateFinished {
			return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "in_progress", Reason: "Waiting for the Human review decision."})
		}
		decision, err = failHumanReviewWithoutResult(db, pr, decision)
		if err != nil {
			return err
		}
		if decision.State == models.FactoryHumanReviewStateSuperseded {
			return nil
		}
		if len(decision.Result) == 0 {
			return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason})
		}
	}

	err = evaluateHumanReviewDecision(ctx, db, client, pr, current, automation, decision)
	if err == nil || errors.Is(err, errHumanReviewPublication) {
		return err
	}
	decision.State = models.FactoryHumanReviewStateFailed
	decision.BlockingReason = "Human review could not verify the current review state. " + err.Error()
	if saveErr := decision.Save(db); saveErr != nil {
		return errors.Join(err, saveErr)
	}
	notifyHumanReviewDecision(db, pr, decision)
	return errors.Join(err, publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason}))
}

func failHumanReviewWithoutResult(db *gorm.DB, pr *models.FactoryPullRequest, decision *models.FactoryHumanReviewDecision) (*models.FactoryHumanReviewDecision, error) {
	changed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		// Runner completion takes the same PR lock. Reload after observing the
		// finished run so a result saved between the two reads cannot be erased.
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		current, err := models.FindFactoryHumanReviewDecision(tx, decision.ID)
		if err != nil {
			return err
		}
		decision = current
		if len(decision.Result) > 0 || decision.State == models.FactoryHumanReviewStateSuperseded {
			return nil
		}
		reason := "The Human review agent failed or returned no valid decision. Run the automation again."
		changed = decision.State != models.FactoryHumanReviewStateFailed || decision.BlockingReason != reason
		decision.State = models.FactoryHumanReviewStateFailed
		decision.BlockingReason = reason
		return decision.Save(tx)
	})
	if err == nil && changed {
		notifyHumanReviewDecision(db, pr, decision)
	}
	return decision, err
}

func equalHumanReviewRun(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func humanReviewCheckURL(previous *models.FactoryHumanReviewDecision) string {
	if previous == nil {
		return ""
	}
	return previous.CheckURL
}

func sameRevisionCheck(previous *models.FactoryHumanReviewDecision, sha string) int64 {
	if previous == nil || previous.AssessedCommit != sha {
		return 0
	}
	return previous.CheckRunID
}

func validHumanReviewAssessment(report mergeconfidence.ReportData, prID uuid.UUID, sha string) bool {
	return report.PullRequestID == prID.String() && humanreview.ValidAssessment(report, sha)
}

func startHumanReviewAgent(ctx context.Context, db *gorm.DB, factory *models.Factory, pr *models.FactoryPullRequest, automation *humanReviewAutomation, client humanReviewGitHub, current *github.PullRequest, decision *models.FactoryHumanReviewDecision, report mergeconfidence.ReportData) error {
	contextData, err := gatherHumanReviewContext(ctx, client, pr, current, automation.settings)
	if err != nil {
		return err
	}
	order, err := factory.FindWorkOrder(db, pr.WorkOrderID)
	if err != nil {
		return err
	}
	payload, err := pullRequestChecksEventData(pr, order, current)
	if err != nil {
		return err
	}
	contextData["decisionId"] = decision.ID.String()
	contextData["pullRequestId"] = pr.ID.String()
	contextData["assessedCommit"] = decision.AssessedCommit
	contextData["sourceRunId"] = decision.SourceRunID.String()
	contextData["assessment"] = report
	payload["humanReview"] = contextData
	var event *models.CanvasEvent
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		active, err := humanReviewPublicationCurrent(tx, pr, decision)
		if err != nil {
			return err
		}
		if !active {
			return nil
		}
		trigger, err := models.FindCanvasNode(tx, automation.canvas.ID, models.HumanReviewTriggerNodeID)
		if err != nil {
			return err
		}
		var created []models.CanvasEvent
		events := contexts.NewEventContext(tx, trigger, nil, func(events []models.CanvasEvent) { created = append(created, events...) })
		if err := events.EmitDeduplicated("factory.humanReview", payload, decision.ID.String()); err != nil {
			return err
		}
		if len(created) == 0 {
			return errors.New("Human review did not create an agent run")
		}
		event = &created[0]
		decision.RunID = &event.RunID
		decision.State = models.FactoryHumanReviewStateWaitingDecision
		if _, err := pr.CreateActivity(tx, models.FactoryPullRequestActivityParams{RunID: event.RunID, RevisionSHA: decision.AssessedCommit, Title: "Human review", Access: models.FactoryPullRequestAccessConcurrent}); err != nil {
			return err
		}
		return decision.Save(tx)
	})
	if err != nil {
		return err
	}
	if event == nil {
		return nil
	}

	if err := messages.PublishCanvasEventCreatedMessage(event); err != nil {
		// The event is durable and the event router also recovers pending rows.
		log.WithError(err).Warnf("Human review: failed to publish event %s", event.ID)
	}
	return nil
}

func newHumanReviewDecision(db *gorm.DB, pr *models.FactoryPullRequest, params models.FactoryHumanReviewDecision) (*models.FactoryHumanReviewDecision, error) {
	var decision *models.FactoryHumanReviewDecision
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		var err error
		decision, err = pr.NewHumanReviewDecision(tx, params)
		return err
	})
	if err == nil {
		notifyHumanReviewDecision(db, pr, decision)
	}
	return decision, err
}

// The authenticated Factory webhook path calls this after resolving tracked
// PRs. Reopen webhooks persist a request before scheduling so polling can also
// recover closed PRs after a full queue or restart. Polling retains its
// cursor until the queue accepts the request, so later PRs cannot be starved.
func ScheduleFactoryHumanReview(ctx context.Context, deps IntakeDependencies, pr models.FactoryPullRequest) bool {
	return factoryHumanReviewScheduler.schedule(context.WithoutCancel(ctx), deps, pr)
}

func notifyHumanReviewDecision(db *gorm.DB, pr *models.FactoryPullRequest, decision *models.FactoryHumanReviewDecision) {
	if decision.RunID != nil {
		activity, err := models.FindPullRequestActivityByRunID(db, *decision.RunID)
		if err == nil {
			description := decision.Rationale
			if decision.BlockingReason != "" {
				description += "\n\n" + decision.BlockingReason
			}
			for _, reviewer := range decision.SelectedReviewers {
				description += "\n\n@" + reviewer.Login + ": " + reviewer.Reason
			}
			err = activity.UpdateContent(db, nil, &description)
		}
		if err != nil {
			log.WithError(err).Warn("Human review: failed to update PR activity")
		}
	}
	log.WithFields(log.Fields{"pull_request": pr.ID, "decision": decision.ID, "state": decision.State, "blocking_reason": decision.BlockingReason}).Info("Human review decision changed")
	if err := publishFactoryWorkOrderUpdated(pr.FactoryID.String(), pr.WorkOrderID.String(), factoryevents.EventTypeOrderPullRequestUpdated); err != nil {
		log.WithError(err).Warn("Human review: failed to publish task update")
	}
}

func synchronizeHumanReviewReopen(db *gorm.DB, factory *models.Factory, pr *models.FactoryPullRequest, current *github.PullRequest) error {
	state := models.FactoryPullRequestStateOpen
	if current.GetDraft() {
		state = models.FactoryPullRequestStateDraft
	}
	if pr.State == state {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := models.LockFactoryWorkOrder(tx, pr.WorkOrderID); err != nil {
			return err
		}
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		if err := pr.ReopenForHumanReview(tx, state); err != nil {
			return err
		}
		order, err := factory.FindWorkOrder(tx, pr.WorkOrderID)
		if err != nil {
			return err
		}
		if order.IsOpen() || order.Result != models.FactoryWorkOrderResultPRClosed {
			return nil
		}
		_, err = order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen, SkipSame: true})
		return err
	})
}
