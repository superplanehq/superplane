package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var errHumanReviewPublication = errors.New("human review publication failed")

type humanReviewCheckPublisher interface {
	CreateHumanReviewCheck(context.Context, string, github.CreateCheckRunOptions) (*github.CheckRun, *github.Response, error)
	UpdateHumanReviewCheck(context.Context, string, int64, github.UpdateCheckRunOptions) (*github.CheckRun, *github.Response, error)
	ListCheckRunsForRef(context.Context, string, string, *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
}

type humanReviewPublication struct{ Status, Conclusion, Reason string }

func publishHumanReviewCheck(ctx context.Context, db *gorm.DB, client humanReviewCheckPublisher, pr *models.FactoryPullRequest, decision *models.FactoryHumanReviewDecision, publication humanReviewPublication) error {
	// Callers hold the PR lock. The current revision and superseded flag are
	// checked again immediately before the network mutation.
	current, err := pr.CurrentHumanReview(db)
	if err != nil {
		return err
	}
	if current == nil || current.ID != decision.ID || pr.CurrentRevisionID == nil || *pr.CurrentRevisionID != decision.RevisionID {
		return nil
	}
	decision.PublicationPending = true
	decision.BlockingReason = publication.Reason
	if publication.Conclusion == "success" {
		decision.BlockingReason = ""
	}
	if err := decision.RecordPublication(db); err != nil {
		return err
	}
	title := "Human review"
	summary := publication.Reason
	if decision.Rationale != "" {
		summary += "\n\n" + decision.Rationale
	}
	if assessment, ok := decision.Assessment["taskUrl"].(string); ok && assessment != "" {
		summary += "\n\n[Merge confidence assessment](" + assessment + ")"
	}
	for _, reviewer := range decision.SelectedReviewers {
		summary += "\n\n@" + reviewer.Login + ": " + reviewer.Reason
	}
	output := &github.CheckRunOutput{Title: &title, Summary: &summary}
	{
		// Recover a create accepted by GitHub just before a process restart. The
		// durable decision ID makes the remote publication discoverable.
		options := &github.ListCheckRunsOptions{CheckName: github.Ptr(humanReviewCheckName), ListOptions: github.ListOptions{PerPage: 100}}
		for {
			checks, response, err := client.ListCheckRunsForRef(ctx, pr.Repository, decision.AssessedCommit, options)
			if err != nil {
				return recordHumanReviewPublicationError(db, decision, err)
			}
			if checks != nil {
				for _, check := range checks.CheckRuns {
					if check.GetExternalID() == decision.ID.String() || (decision.CheckRunID != 0 && decision.CheckRunID == check.GetID()) {
						decision.CheckRunID = check.GetID()
						decision.CheckURL = check.GetHTMLURL()
						if check.GetExternalID() == decision.ID.String() && check.GetStatus() == publication.Status && check.GetConclusion() == publication.Conclusion && check.GetOutput().GetSummary() == summary {
							decision.PublicationPending = false
							decision.PublicationError = ""
							decision.RetryAt = nil
							return decision.RecordPublication(db)
						}
						break
					}
				}
			}
			if response == nil || response.NextPage == 0 {
				break
			}
			options.Page = response.NextPage
		}
	}
	active, err := humanReviewPublicationCurrent(db, pr, decision)
	if err != nil || !active {
		return err
	}
	var check *github.CheckRun
	if decision.CheckRunID == 0 {
		options := github.CreateCheckRunOptions{Name: humanReviewCheckName, HeadSHA: decision.AssessedCommit, ExternalID: github.Ptr(decision.ID.String()), Status: &publication.Status, Output: output}
		if publication.Conclusion != "" {
			options.Conclusion = &publication.Conclusion
		}
		check, _, err = client.CreateHumanReviewCheck(ctx, pr.Repository, options)
	} else {
		options := github.UpdateCheckRunOptions{Name: humanReviewCheckName, ExternalID: github.Ptr(decision.ID.String()), Status: &publication.Status, Output: output}
		if publication.Conclusion != "" {
			options.Conclusion = &publication.Conclusion
		}
		check, _, err = client.UpdateHumanReviewCheck(ctx, pr.Repository, decision.CheckRunID, options)
	}
	if err != nil {
		return recordHumanReviewPublicationError(db, decision, err)
	}
	if check == nil || check.GetID() == 0 {
		return recordHumanReviewPublicationError(db, decision, fmt.Errorf("GitHub did not return a check ID"))
	}
	decision.CheckRunID = check.GetID()
	if check.GetHTMLURL() != "" {
		decision.CheckURL = check.GetHTMLURL()
	}
	decision.PublicationPending = false
	decision.PublicationError = ""
	decision.RetryAt = nil
	return decision.RecordPublication(db)
}

func recordHumanReviewPublicationError(db *gorm.DB, decision *models.FactoryHumanReviewDecision, cause error) error {
	retryAt := time.Now().Add(time.Minute)
	decision.PublicationPending = true
	decision.PublicationError = strings.TrimSpace(cause.Error())
	decision.RetryAt = &retryAt
	if err := decision.RecordPublication(db); err != nil {
		return fmt.Errorf("record Human review publication failure: %w", err)
	}
	log.WithError(cause).WithField("decision", decision.ID).Warn("Human review publication failed")
	return fmt.Errorf("%w: %w", errHumanReviewPublication, cause)
}

func humanReviewPublicationCurrent(db *gorm.DB, pr *models.FactoryPullRequest, decision *models.FactoryHumanReviewDecision) (bool, error) {
	if err := pr.LockForUpdate(db); err != nil {
		return false, err
	}
	current, err := pr.CurrentHumanReview(db)
	if err != nil {
		return false, err
	}
	if current == nil || current.ID != decision.ID || pr.CurrentRevisionID == nil || *pr.CurrentRevisionID != decision.RevisionID || (pr.State != models.FactoryPullRequestStateOpen && pr.State != models.FactoryPullRequestStateDraft) {
		return false, nil
	}
	if decision.ConfigurationVersion == "" {
		return true, nil
	}
	factory, err := models.FindFactory(db, pr.OrganizationID, pr.FactoryID)
	if err != nil {
		return false, err
	}
	automation, err := findHumanReviewAutomation(db, factory)
	if err != nil {
		return false, err
	}
	if automation == nil || automation.version.ID.String() != decision.ConfigurationVersion {
		return false, nil
	}
	if decision.State == models.FactoryHumanReviewStateDisabled {
		return !automation.settings.Enabled, nil
	}
	if !automation.settings.Enabled {
		return false, nil
	}
	source, _, err := pr.LatestMergeConfidenceAssessment(db, decision.AssessedCommit)
	if err != nil {
		return false, err
	}
	if source == nil {
		return decision.SourceRunID == nil, nil
	}
	return decision.SourceRunID != nil && *decision.SourceRunID == source.ID, nil
}
