package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type MergeConfidencePublicationOutcome struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func skippedMergeConfidence(reason string) MergeConfidencePublicationOutcome {
	return MergeConfidencePublicationOutcome{Status: "skipped", Reason: reason}
}

// PublishMergeConfidence uses saved run configuration and immutable reports.
// Call with a database handle: provider calls run outside database transactions.
func PublishMergeConfidence(ctx context.Context, db *gorm.DB, deps IntakeDependencies, canvasID, runID uuid.UUID, baseURL string) error {
	_, err := PublishMergeConfidenceWithOutcome(ctx, db, deps, canvasID, runID, baseURL)
	return err
}

func PublishMergeConfidenceWithOutcome(ctx context.Context, db *gorm.DB, deps IntakeDependencies, canvasID, runID uuid.UUID, baseURL string) (outcome MergeConfidencePublicationOutcome, err error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	db = db.WithContext(ctx)
	logger := log.WithFields(log.Fields{"component": "publishMergeConfidence", "run_id": runID})
	run, err := models.FindUnscopedCanvasRun(db, runID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return skippedMergeConfidence("Source run was not found."), nil
	}
	if err != nil {
		return outcome, err
	}
	if run.WorkflowID != canvasID || run.State != models.CanvasRunStateFinished {
		return skippedMergeConfidence("Source run is not finished in this canvas."), nil
	}
	activity, err := models.FindPullRequestActivityByRunID(db, run.ID)
	if errors.Is(err, models.ErrFactoryPullRequestActivityNotFound) {
		return skippedMergeConfidence("Source run has no pull request activity."), nil
	}
	if err != nil {
		return outcome, err
	}
	claim := &models.MergeConfidencePublicationClaim{PullRequestID: activity.PullRequestID, OwnerID: uuid.New(), LeaseUntil: time.Now().Add(5 * time.Minute)}
	for {
		acquired, err := claim.Acquire(db)
		if err != nil {
			return outcome, err
		}
		if acquired {
			break
		}
		select {
		case <-ctx.Done():
			return outcome, fmt.Errorf("wait for publication claim: %w", ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	defer func() {
		// A request cancellation must not prevent releasing our claim.
		releaseContext, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelRelease()
		if releaseErr := claim.Release(db.WithContext(releaseContext)); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
	}()

	var pr *models.FactoryPullRequest
	var workspace *models.Factory
	var assessment *mergeconfidence.Assessment
	err = db.Transaction(func(tx *gorm.DB) error {
		pr = &models.FactoryPullRequest{ID: activity.PullRequestID}
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		var err error
		assessment, err = newestMergeConfidenceAssessment(tx, pr, run.ID)
		if err != nil {
			return err
		}
		if assessment == nil {
			outcome = skippedMergeConfidence("Assessment is stale or has no supported checks.")
			return nil
		}
		workspace, err = models.FindFactory(tx, pr.OrganizationID, pr.FactoryID)
		if err != nil {
			return err
		}
		canvas, err := models.FindCanvasInTransaction(tx, pr.OrganizationID, canvasID)
		if err != nil {
			return err
		}
		if canvas.FactoryID == nil || *canvas.FactoryID != workspace.ID {
			assessment = nil
			outcome = skippedMergeConfidence("Assessment belongs to a different factory.")
			return nil
		}
		order, err := workspace.FindWorkOrder(tx, pr.WorkOrderID)
		if err != nil {
			return err
		}
		assessment.Checks, err = order.ChecksForRun(tx, run.ID)
		if err != nil {
			return err
		}
		event, err := run.RootEvent(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			assessment = nil
			outcome = skippedMergeConfidence("Assessment has no source event.")
			return nil
		}
		if err != nil {
			return err
		}
		assessment.ReviewedCommit = mergeConfidenceReviewedCommit(event)
		if assessment.ReviewedCommit == "" {
			assessment = nil
			outcome = skippedMergeConfidence("Assessment has no reviewed commit.")
			return nil
		}
		assessment.Passed = run.Result == models.CanvasRunResultPassed
		assessment.TaskURL = strings.TrimRight(baseURL, "/") + order.URLPath(workspace.RouteSegment())
		return nil
	})
	if err != nil || assessment == nil {
		return outcome, err
	}
	provider := openFactoryVCS(db, deps, workspace)
	client, ok := provider.(vcs.DescriptionClient)
	if !ok || workspace.OnboardingConfigValue().EffectiveVCSProvider() != pr.Provider {
		return skippedMergeConfidence("Pull request provider is unavailable."), nil
	}
	return publishMergeConfidenceDescription(ctx, client, pr, assessment, logger, func() (bool, error) {
		// New assessments can attach while the provider reads the description.
		// Recheck immediately before writing, without holding a connection at I/O.
		current, err := claim.IsCurrent(db)
		if err != nil || !current {
			return false, err
		}
		latest, err := newestMergeConfidenceAssessment(db, pr, run.ID)
		return latest != nil, err
	})
}

func newestMergeConfidenceAssessment(tx *gorm.DB, pr *models.FactoryPullRequest, runID uuid.UUID) (*mergeconfidence.Assessment, error) {
	runs, err := pr.ListActivityRuns(tx)
	if err != nil {
		return nil, err
	}
	for _, run := range runs {
		version, err := models.FindCanvasVersionInTransaction(tx, run.WorkflowID, run.VersionID)
		if err != nil {
			return nil, err
		}
		if !isMergeConfidenceAssessmentRun(run, version) {
			continue
		}
		// A newer run supersedes this one even if its saved settings are
		// invalid or its agent never started.
		if run.ID != runID {
			return nil, nil
		}
		return mergeConfidenceAssessmentFromVersion(version)
	}
	return nil, nil
}

func isMergeConfidenceAssessmentRun(run models.CanvasRun, version *models.CanvasVersion) bool {
	if models.FactoryAppTemplateID(version.Nodes) != "risk-score" {
		return false
	}
	for _, node := range version.Nodes {
		if node.ID == run.NodeID {
			// Completion branches share the template marker but are not assessments.
			return node.Ref.Trigger != nil && (node.Ref.Trigger.Name == "github.onPullRequest" || node.Ref.Trigger.Name == "bitbucket.onPullRequest")
		}
	}
	return false
}

func mergeConfidenceAssessmentFromVersion(version *models.CanvasVersion) (*mergeconfidence.Assessment, error) {
	var assessment *mergeconfidence.Assessment
	seen := map[string]bool{}
	for _, node := range version.Nodes {
		encoded, err := json.Marshal(node.Configuration)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(node.ComponentName(), "runner") {
			continue
		}
		var config struct {
			Environment []struct {
				Name string `json:"name"`
			} `json:"environment"`
			Steps json.RawMessage `json:"steps"`
		}
		if err := json.Unmarshal(encoded, &config); err != nil {
			return nil, err
		}
		scoped := false
		for _, env := range config.Environment {
			if env.Name == runner.EnvSuperplaneMergeConfidenceOrderID {
				scoped = true
				break
			}
		}
		if !scoped {
			continue
		}
		if assessment == nil {
			assessment = &mergeconfidence.Assessment{Labels: map[string]string{}}
		}
		var steps []runner.AgentStep
		if len(config.Steps) > 0 {
			if err := json.Unmarshal(config.Steps, &steps); err != nil {
				return nil, err
			}
		}
		enabled, labels := runner.MergeConfidenceChecksFromSteps(steps)
		for _, id := range enabled {
			if seen[id] {
				continue
			}
			seen[id] = true
			assessment.Enabled = append(assessment.Enabled, id)
			assessment.Labels[id] = labels[id]
		}
	}
	return assessment, nil
}

func mergeConfidenceReviewedCommit(event *models.CanvasEvent) string {
	// Bitbucket trigger events also carry this normalized pull_request shape.
	payload, ok := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return ""
	}
	pr, _ := payload["pull_request"].(map[string]any)
	head, _ := pr["head"].(map[string]any)
	sha, _ := head["sha"].(string)
	return strings.TrimSpace(sha)
}

func publishMergeConfidenceDescription(ctx context.Context, client vcs.DescriptionClient, pr *models.FactoryPullRequest, a *mergeconfidence.Assessment, logger *log.Entry, eligible func() (bool, error)) (MergeConfidencePublicationOutcome, error) {
	ref := vcs.PullRequestRef{Provider: pr.Provider, Repository: pr.Repository, Number: pr.Number}
	section := mergeconfidence.Render(*a)
	latest, err := client.ReadDescription(ctx, ref)
	if err != nil {
		return descriptionPublicationError(logger, err)
	}
	if latest.State != models.FactoryPullRequestStateOpen || latest.HeadSHA != a.ReviewedCommit {
		return skippedMergeConfidence("Pull request is closed or its reviewed commit changed."), nil
	}
	description, err := mergeconfidence.UpdateDescription(latest.Description, section)
	if err != nil {
		logger.WithError(err).Warn("Skipping merge confidence publication")
		return skippedMergeConfidence("Description contains invalid report markers."), nil
	}
	if description == latest.Description {
		return skippedMergeConfidence("Report is already current."), nil
	}
	current, err := eligible()
	if err != nil {
		return MergeConfidencePublicationOutcome{}, err
	}
	if !current {
		return skippedMergeConfidence("Assessment was superseded before publication."), nil
	}
	if err := client.UpdateDescription(ctx, ref, description); err != nil {
		return descriptionPublicationError(logger, err)
	}
	return MergeConfidencePublicationOutcome{Status: "published"}, nil
}

func descriptionPublicationError(logger *log.Entry, err error) (MergeConfidencePublicationOutcome, error) {
	if retryableDescriptionError(err) {
		return MergeConfidencePublicationOutcome{}, fmt.Errorf("publish merge confidence description: %w", err)
	}
	logger.WithError(err).Warn("Skipping merge confidence publication after a permanent host failure")
	return skippedMergeConfidence("Host rejected publication: " + err.Error()), nil
}
