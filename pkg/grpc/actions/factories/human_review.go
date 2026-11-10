package factories

import (
	"context"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
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
		reason := "The Request review agent failed or returned no valid decision. Run the automation again."
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

// Run retention clears SourceRunID through the foreign key. An older run for the
// same head can then become the latest assessment. Runs can finish out of
// creation order, so only a run created after the saved decision replaces the
// saved assessment. Only a completed decision keeps its saved assessment. A
// decision without a result must start an agent, and the agent needs a source run.
func retentionClearedHumanReviewSource(decision *models.FactoryHumanReviewDecision, source *models.CanvasRun) bool {
	if decision.SourceRunID != nil || len(decision.Assessment) == 0 || len(decision.Result) == 0 || source == nil || source.CreatedAt == nil {
		return false
	}
	return !source.CreatedAt.After(decision.CreatedAt)
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
