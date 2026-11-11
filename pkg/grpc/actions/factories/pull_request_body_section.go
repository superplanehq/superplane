package factories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type bodySectionPublicationOutcome = prbody.Outcome

func skippedBodySection(reason string) bodySectionPublicationOutcome {
	return prbody.Skipped(reason)
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
	if err := prbody.ValidateBodySectionKey(cfg.SectionKey); err != nil {
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
	outcome, err := prbody.Publish(ctx, factoryBodySectionClient{client: client, ref: ref}, prbody.Configuration{
		Repository: ref.Repository, PullNumber: strconv.FormatInt(ref.Number, 10), SectionKey: cfg.SectionKey,
		Body: cfg.Body, ExpectedHeadSHA: cfg.ExpectedHeadSHA, Enabled: cfg.Enabled,
	}, eligible)
	if err != nil {
		return bodySectionPublicationError(err)
	}
	return outcome, nil
}

func (p sectionPublication) eligible(db *gorm.DB, pr *models.FactoryPullRequest) (bool, error) {
	if p.stillActive != nil {
		active, err := p.stillActive(db)
		if err != nil || !active {
			return false, err
		}
	}
	enabled, err := p.nodesEnabled(db)
	if err != nil || !enabled {
		return false, err
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
			if prbody.IsComponent(node.ComponentName()) && node.Configuration["sectionKey"] == p.configuration.SectionKey {
				return run.ID == p.run.ID, nil
			}
		}
	}
	return false, nil
}

func (p sectionPublication) nodesEnabled(db *gorm.DB) (bool, error) {
	version, err := models.FindCanvasVersionInTransaction(db, p.run.WorkflowID, p.run.VersionID)
	if err != nil {
		return false, err
	}
	if !publicationNodeEnabled(version, p.nodeID, p.configuration.SectionKey) {
		return false, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, p.run.WorkflowID)
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
	return true, nil
}

func publicationNodeEnabled(version *models.CanvasVersion, nodeID, sectionKey string) bool {
	for _, node := range version.Nodes {
		if node.ID != nodeID || !prbody.IsComponent(node.ComponentName()) || node.Configuration["sectionKey"] != sectionKey {
			continue
		}
		return node.Configuration["enabled"] != false
	}
	return false
}

func bodySectionPublicationError(err error) (bodySectionPublicationOutcome, error) {
	if errors.Is(err, errBodySectionIntegrationUnavailable) {
		return skippedBodySection("Publication integration is unavailable."), nil
	}
	if retryableDescriptionError(err) {
		return bodySectionPublicationOutcome{}, fmt.Errorf("publish pull request body section: %w", err)
	}
	return skippedBodySection("Host rejected publication: " + err.Error()), nil
}

// The legacy action resolves its credentials through the workspace connection.
type factoryBodySectionClient struct {
	client vcs.DescriptionClient
	ref    vcs.PullRequestRef
}

func (c factoryBodySectionClient) Read(ctx context.Context, _ string, _ int64) (prbody.Description, error) {
	latest, err := c.client.ReadDescription(ctx, c.ref)
	if err != nil {
		return prbody.Description{}, err
	}
	return prbody.Description{Body: latest.Description, Open: latest.State == models.FactoryPullRequestStateOpen, HeadSHA: latest.HeadSHA}, nil
}
func (c factoryBodySectionClient) Write(ctx context.Context, _ string, _ int64, body string) error {
	return c.client.UpdateDescription(ctx, c.ref, body)
}
