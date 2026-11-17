package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

// RunMergeConfidenceRequest selects an existing task in one workspace.
type RunMergeConfidenceRequest struct {
	FactoryID string
	Task      string
}

type MergeConfidenceRun struct {
	PullRequestURL string `json:"pull_request_url"`
	AutomationID   string `json:"automation_id"`
	RunID          string `json:"run_id"`
	EventID        string `json:"event_id"`
	Status         string `json:"status"`
}

type MergeConfidenceSkipped struct {
	PullRequestURL string `json:"pull_request_url"`
	Reason         string `json:"reason"`
}

type RunMergeConfidenceResult struct {
	Order   *models.FactoryWorkOrder
	Runs    []MergeConfidenceRun
	Skipped []MergeConfidenceSkipped
}

type mergeConfidenceCandidate struct {
	pullRequest   models.FactoryPullRequest
	automationIDs []uuid.UUID
	payload       map[string]any
}

// RunMergeConfidence starts assessments asynchronously using the saved automations.
// All provider snapshots are read before any new event is saved.
func RunMergeConfidence(ctx context.Context, deps IntakeDependencies, organizationID string, req RunMergeConfidenceRequest) (*RunMergeConfidenceResult, error) {
	result, err := runMergeConfidence(ctx, deps, organizationID, req)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to run merge confidence")
	}
	return result, nil
}

func runMergeConfidence(ctx context.Context, deps IntakeDependencies, organizationID string, req RunMergeConfidenceRequest) (*RunMergeConfidenceResult, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.FactoryID)
	if err != nil {
		return nil, err
	}
	order, err := findWorkOrder(db, factory, req.Task)
	if err != nil {
		return nil, err
	}
	if err := requireMergeConfidenceTask(db, order); err != nil {
		return nil, err
	}
	candidates, err := loadMergeConfidenceCandidates(ctx, db, deps, factory, order)
	if err != nil {
		return nil, err
	}
	result := &RunMergeConfidenceResult{Order: order, Runs: []MergeConfidenceRun{}, Skipped: []MergeConfidenceSkipped{}}
	var created []models.CanvasEvent
	err = db.Transaction(func(tx *gorm.DB) error {
		// Use the task -> PR lock order shared by PR closure and checks startup.
		if err := order.LockForUpdate(tx); err != nil {
			return err
		}
		if err := requireMergeConfidenceTask(tx, order); err != nil {
			return err
		}
		open, matched := false, false
		for i := range candidates {
			candidate := &candidates[i]
			pr := &candidate.pullRequest
			identity := *pr
			if err := pr.LockForUpdate(tx); err != nil {
				return err
			}
			if pr.WorkOrderID != order.ID || pr.Provider != identity.Provider || pr.Repository != identity.Repository || pr.Number != identity.Number {
				return grpcFailedPrecondition("The tracked pull request changed. Try again.")
			}
			if pr.State != models.FactoryPullRequestStateOpen || (len(candidate.automationIDs) > 0 && candidate.payload == nil) {
				result.Skipped = append(result.Skipped, MergeConfidenceSkipped{pr.URL, "The pull request is no longer open."})
				continue
			}
			open = true
			found := false
			previousRuns := len(result.Runs)
			for _, automationID := range candidate.automationIDs {
				canvas, trigger, err := mergeConfidenceTrigger(tx, factory, automationID, pr)
				if err != nil {
					return err
				}
				if trigger == nil {
					continue
				}
				found, matched = true, true
				if checksStartupExcludesDraft(trigger, pr.Provider, candidate.payload) {
					continue
				}
				if err := validateChecksStartupSnapshot(pr, candidate.payload); err != nil {
					return err
				}
				event, started, err := contexts.CreateFactoryPullRequestEvent(tx, trigger, pr.Provider+".pullRequest", candidate.payload)
				if err != nil {
					return err
				}
				if event == nil {
					return grpcFailedPrecondition("The merge confidence automation is no longer eligible.")
				}
				// Link from the saved payload so task authority exists before routing.
				event, err = models.FindCanvasEventForCanvas(tx, canvas.ID, event.ID)
				if err != nil {
					return err
				}
				run, err := models.FindCanvasRunInTransaction(tx, canvas.ID, event.RunID)
				if err != nil {
					return err
				}
				activity, err := models.FindPullRequestActivityByRunID(tx, run.ID)
				if err != nil && !errors.Is(err, models.ErrFactoryPullRequestActivityNotFound) {
					return err
				}
				if activity != nil && activity.PullRequestID != pr.ID {
					return models.ErrFactoryPullRequestRunAlreadyLinked
				}
				// Reuse keeps existing task authority without admitting another run.
				if activity == nil {
					if err := mergeconfidence.LinkRun(tx, canvas, event, run); err != nil {
						return err
					}
				}
				status := "reused"
				if started {
					status = "started"
					created = append(created, *event)
				}
				result.Runs = append(result.Runs, MergeConfidenceRun{PullRequestURL: pr.URL, AutomationID: canvas.ID.String(), RunID: event.RunID.String(), EventID: event.ID.String(), Status: status})
			}
			if !found {
				result.Skipped = append(result.Skipped, MergeConfidenceSkipped{pr.URL, "No eligible Merge confidence automation matches this pull request."})
				continue
			}
			// A PR can match several automations with different GitHub draft settings.
			if len(result.Runs) == previousRuns {
				result.Skipped = append(result.Skipped, MergeConfidenceSkipped{pr.URL, "The pull request is excluded by draft rules."})
			}
		}
		if !open {
			return grpcFailedPrecondition("The task has no open pull requests.")
		}
		if !matched {
			return grpcFailedPrecondition("The task has no eligible Merge confidence automation.")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range created {
		if err := messages.PublishCanvasEventCreatedMessage(&created[i]); err != nil {
			log.WithError(err).Warnf("merge confidence: failed to publish event %s", created[i].ID)
		}
	}
	return result, nil
}

func requireMergeConfidenceTask(tx *gorm.DB, order *models.FactoryWorkOrder) error {
	if order.IsArchived() {
		return grpcFailedPrecondition("Archived tasks cannot run merge confidence.")
	}
	return requireWorkOrderInVerify(tx, order)
}

func loadMergeConfidenceCandidates(ctx context.Context, db *gorm.DB, deps IntakeDependencies, factory *models.Factory, order *models.FactoryWorkOrder) ([]mergeConfidenceCandidate, error) {
	pullRequests, err := trackedPullRequests(db, order.ID)
	if err != nil {
		return nil, err
	}
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return nil, err
	}
	candidates := []mergeConfidenceCandidate{}
	matched := false
	for _, pr := range pullRequests {
		if pr.State != models.FactoryPullRequestStateOpen {
			continue
		}
		candidate := mergeConfidenceCandidate{pullRequest: pr}
		for _, canvas := range canvases {
			_, trigger, err := mergeConfidenceTrigger(db, factory, canvas.ID, &pr)
			if err != nil {
				return nil, err
			}
			if trigger != nil {
				candidate.automationIDs = append(candidate.automationIDs, canvas.ID)
			}
		}
		if len(candidate.automationIDs) > 0 {
			matched = true
			candidate.payload, err = checksStartupSnapshot(ctx, db, deps, factory, &pr)
			if err != nil {
				return nil, err
			}
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, grpcFailedPrecondition("The task has no open pull requests.")
	}
	if !matched {
		return nil, grpcFailedPrecondition("The task has no eligible Merge confidence automation.")
	}
	return candidates, nil
}

func mergeConfidenceTrigger(tx *gorm.DB, factory *models.Factory, automationID uuid.UUID, pr *models.FactoryPullRequest) (*models.Canvas, *models.CanvasNode, error) {
	canvas, err := models.FindCanvasInTransaction(tx, factory.OrganizationID, automationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID || canvas.LiveVersionID == nil {
		return nil, nil, nil
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if models.FactoryAppTemplateID(version.Nodes) != "risk-score" {
		return nil, nil, nil
	}
	trigger, err := canvas.VerificationTrigger(tx, pr.Provider, pr.Repository)
	return canvas, trigger, err
}
