package factories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type bodySectionPublicationOutcome struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func skippedBodySection(reason string) bodySectionPublicationOutcome {
	return bodySectionPublicationOutcome{Status: "skipped", Reason: reason}
}

type sectionPublication struct {
	run           *models.CanvasRun
	nodeID        string
	configuration factorycomponent.UpdatePullRequestBodySectionConfiguration
	stillActive   func(*gorm.DB) (bool, error)
}

// publishBodySection reads the latest description on every attempt. Provider
// access never holds a database transaction or a checked-out connection.
func publishBodySection(ctx context.Context, db *gorm.DB, deps IntakeDependencies, publication sectionPublication) (bodySectionPublicationOutcome, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	db = db.WithContext(ctx)
	cfg := publication.configuration
	if err := vcs.ValidateBodySectionKey(cfg.SectionKey); err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if !cfg.IsEnabled() {
		return skippedBodySection("Publication is disabled for this run."), nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, publication.run.WorkflowID)
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if canvas.FactoryID == nil {
		return skippedBodySection("Canvas is not owned by a factory."), nil
	}
	workspace, err := models.FindFactory(db, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	id, err := uuid.Parse(cfg.PullRequestID)
	if err != nil {
		return bodySectionPublicationOutcome{}, fmt.Errorf("invalid pullRequestId: %w", err)
	}
	pr, err := workspace.FindPullRequest(db, models.FactoryPullRequestLookup{ID: id})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return skippedBodySection("Pull request was not found in this factory."), nil
	}
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	eligible := func() (bool, error) { return publication.eligible(db, pr) }
	active, err := eligible()
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if !active {
		return skippedBodySection("Publication is disabled, cancelled, or superseded."), nil
	}
	if workspace.OnboardingConfigValue().EffectiveVCSProvider() != pr.Provider {
		return skippedBodySection("Pull request provider is unavailable."), nil
	}
	client, ok := openFactoryVCS(db, deps, workspace).(vcs.DescriptionClient)
	if !ok {
		return skippedBodySection("Pull request provider is unavailable."), nil
	}
	ref := vcs.PullRequestRef{Provider: pr.Provider, Repository: pr.Repository, Number: pr.Number}
	latest, err := client.ReadDescription(ctx, ref)
	if err != nil {
		return bodySectionPublicationError(err)
	}
	if latest.State != models.FactoryPullRequestStateOpen || (cfg.ExpectedHeadSHA != "" && latest.HeadSHA != cfg.ExpectedHeadSHA) {
		return skippedBodySection("Pull request is closed or its expected commit changed."), nil
	}
	description, err := vcs.UpdateBodySection(latest.Description, cfg.SectionKey, cfg.Body)
	if err != nil {
		return skippedBodySection("Description contains invalid section markers."), nil
	}
	if description == latest.Description {
		return skippedBodySection("Section is already current."), nil
	}
	// Settings, cancellation, and newer PR activity can change during the read.
	// Recheck immediately before the write. An already-started write may finish.
	active, err = eligible()
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if !active {
		return skippedBodySection("Publication was disabled, cancelled, or superseded before writing."), nil
	}
	if err := client.UpdateDescription(ctx, ref, description); err != nil {
		return bodySectionPublicationError(err)
	}
	return bodySectionPublicationOutcome{Status: "published"}, nil
}

func (p sectionPublication) eligible(db *gorm.DB, pr *models.FactoryPullRequest) (bool, error) {
	if p.stillActive != nil {
		active, err := p.stillActive(db)
		if err != nil || !active {
			return false, err
		}
	}
	version, err := models.FindCanvasVersionInTransaction(db, p.run.WorkflowID, p.run.VersionID)
	if err != nil {
		return false, err
	}
	if !publicationNodeEnabled(version, p.nodeID, p.configuration.SectionKey) {
		return false, nil
	}
	canvas, err := models.FindCanvasInTransaction(db, pr.OrganizationID, p.run.WorkflowID)
	if err != nil {
		return false, err
	}
	live, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	if err != nil {
		return false, err
	}
	if !publicationNodeEnabled(live, p.nodeID, p.configuration.SectionKey) {
		return false, nil
	}
	activity, err := models.FindPullRequestActivityByRunID(db, p.run.ID)
	if errors.Is(err, models.ErrFactoryPullRequestActivityNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if activity.PullRequestID != pr.ID {
		return true, nil
	}
	runs, err := pr.ListActivityRuns(db)
	if err != nil {
		return false, err
	}
	for _, run := range runs {
		saved, err := models.FindCanvasVersionInTransaction(db, run.WorkflowID, run.VersionID)
		if err != nil {
			return false, err
		}
		for _, node := range saved.Nodes {
			// A newer applicable run supersedes this one even when disabled or failed
			// before its publishing node started. Unrelated section keys do not.
			if node.ComponentName() == factorycomponent.UpdatePullRequestBodySectionComponentName && node.Configuration["sectionKey"] == p.configuration.SectionKey {
				return run.ID == p.run.ID, nil
			}
		}
	}
	return false, nil
}

func publicationNodeEnabled(version *models.CanvasVersion, nodeID, sectionKey string) bool {
	for _, node := range version.Nodes {
		if node.ID != nodeID || node.ComponentName() != factorycomponent.UpdatePullRequestBodySectionComponentName || node.Configuration["sectionKey"] != sectionKey {
			continue
		}
		return node.Configuration["enabled"] != false
	}
	return false
}

func bodySectionPublicationError(err error) (bodySectionPublicationOutcome, error) {
	if retryableDescriptionError(err) {
		return bodySectionPublicationOutcome{}, fmt.Errorf("publish pull request body section: %w", err)
	}
	return skippedBodySection("Host rejected publication: " + err.Error()), nil
}
