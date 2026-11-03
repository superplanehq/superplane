package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// PublishMergeConfidence handles a canvas run notification. It uses saved run
// configuration and immutable check reports. Host failures never change the run.
func PublishMergeConfidence(ctx context.Context, db *gorm.DB, deps IntakeDependencies, canvasID, runID uuid.UUID, baseURL string) error {
	logger := log.WithFields(log.Fields{"consumer": "merge_confidence_publication", "run_id": runID})
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		run, err := models.FindUnscopedCanvasRun(tx, runID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if run.WorkflowID != canvasID || run.State != models.CanvasRunStateFinished {
			return nil
		}
		activity, err := models.FindPullRequestActivityByRunID(tx, run.ID)
		if errors.Is(err, models.ErrFactoryPullRequestActivityNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		// The same row lock is taken when linking new PR activities. Hold it through
		// the external update, so concurrent consumers cannot publish out of order.
		pr := &models.FactoryPullRequest{ID: activity.PullRequestID}
		if err := pr.LockForUpdate(tx); err != nil {
			return err
		}
		assessment, err := newestMergeConfidenceAssessment(tx, pr, run.ID)
		if err != nil {
			return err
		}
		if assessment == nil {
			return nil
		}
		workspace, err := models.FindFactory(tx, pr.OrganizationID, pr.FactoryID)
		if err != nil {
			return err
		}
		canvas, err := models.FindCanvasInTransaction(tx, pr.OrganizationID, canvasID)
		if err != nil {
			return err
		}
		if canvas.FactoryID == nil || *canvas.FactoryID != workspace.ID {
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
			logger.Warn("Skipping merge confidence publication without a root event")
			return nil
		}
		if err != nil {
			return err
		}
		assessment.ReviewedCommit = mergeConfidenceReviewedCommit(event)
		if assessment.ReviewedCommit == "" {
			logger.Warn("Skipping merge confidence publication without a reviewed commit")
			return nil
		}
		assessment.Passed = run.Result == models.CanvasRunResultPassed
		assessment.TaskURL = strings.TrimRight(baseURL, "/") + order.URLPath(workspace.RouteSegment())
		provider := openFactoryVCS(tx, deps, workspace)
		client, ok := provider.(vcs.DescriptionClient)
		if !ok || workspace.OnboardingConfigValue().EffectiveVCSProvider() != pr.Provider {
			return nil
		}
		// Token refresh, description read, and update can exceed the normal
		// 30-second idle limit. Keep the row lock for the bounded host requests.
		if err := tx.Exec("SET LOCAL idle_in_transaction_session_timeout = '3min'").Error; err != nil {
			return err
		}
		return publishMergeConfidenceDescription(ctx, client, pr, assessment, logger)
	})
	return err
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
		if models.FactoryAppTemplateID(version.Nodes) != "risk-score" {
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

func publishMergeConfidenceDescription(ctx context.Context, client vcs.DescriptionClient, pr *models.FactoryPullRequest, a *mergeconfidence.Assessment, logger *log.Entry) error {
	ref := vcs.PullRequestRef{Provider: pr.Provider, Repository: pr.Repository, Number: pr.Number}
	section := mergeconfidence.Render(*a)
	// Fetch raw Markdown immediately before every write, including retries.
	latest, err := client.ReadDescription(ctx, ref)
	if err != nil {
		return descriptionPublicationError(logger, err)
	}
	if latest.State != models.FactoryPullRequestStateOpen || latest.HeadSHA != a.ReviewedCommit {
		return nil
	}
	description, err := mergeconfidence.UpdateDescription(latest.Description, section)
	if err != nil {
		logger.WithError(err).WithField("pull_request_id", pr.ID).Warn("Skipping merge confidence publication")
		return nil
	}
	if description == latest.Description {
		return nil
	}
	if err := client.UpdateDescription(ctx, ref, description); err != nil {
		return descriptionPublicationError(logger, err)
	}
	return nil
}

func descriptionPublicationError(logger *log.Entry, err error) error {
	if retryableDescriptionError(err) {
		return fmt.Errorf("publish merge confidence description: %w", err)
	}
	logger.WithError(err).Warn("Skipping merge confidence publication after a permanent host failure")
	return nil
}
