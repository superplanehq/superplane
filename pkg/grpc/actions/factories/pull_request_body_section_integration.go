package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/exprruntime"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errBodySectionIntegrationUnavailable = errors.New("publication integration is unavailable")

func (a *bodySectionActionAttempt) publish(ctx context.Context, db *gorm.DB, deps IntakeDependencies) (bodySectionPublicationOutcome, error) {
	if a.component == factorycomponent.UpdatePullRequestBodySectionComponentName {
		return publishBodySection(ctx, db, deps, sectionPublication{run: a.source, nodeID: a.execution.NodeID, configuration: a.configuration, stillActive: a.stillActive})
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	db = db.WithContext(ctx)
	config := a.integrationConfiguration
	if !config.IsEnabled() {
		return prbody.Skipped("Publication is disabled for this run."), nil
	}
	eligible := func() (bool, error) { return a.integrationPublicationEligible(db) }
	active, err := eligible()
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if !active {
		return prbody.Skipped("Publication is disabled, cancelled, or superseded."), nil
	}
	client, err := a.integrationClient(db, deps)
	if err != nil {
		return bodySectionPublicationError(err)
	}
	return a.publishIntegrationBodySection(ctx, db, client)
}

func (a *bodySectionActionAttempt) publishIntegrationBodySection(ctx context.Context, db *gorm.DB, client prbody.Client) (bodySectionPublicationOutcome, error) {
	config := a.integrationConfiguration
	eligible := func() (bool, error) { return a.integrationPublicationEligible(db) }
	active, err := eligible()
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if !config.IsEnabled() || !active {
		return prbody.Skipped("Publication is disabled, cancelled, or superseded."), nil
	}
	if githubClient, ok := client.(*pulls.BodySectionClient); ok && config.SectionKey == "merge-confidence" {
		config.Body = a.enrichReviewedCommit(ctx, db, githubClient, config)
	}
	outcome, err := prbody.Publish(ctx, client, config, eligible)
	if err != nil {
		return bodySectionPublicationError(err)
	}
	return outcome, nil
}

func (a *bodySectionActionAttempt) enrichReviewedCommit(ctx context.Context, db *gorm.DB, client *pulls.BodySectionClient, config prbody.Configuration) string {
	version, err := models.FindCanvasVersionInTransaction(db, a.source.WorkflowID, a.source.VersionID)
	if err != nil || models.FactoryAppTemplateID(version.Nodes) != "risk-score" {
		return config.Body
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, a.source.WorkflowID)
	if err != nil || canvas.FactoryID == nil {
		return config.Body
	}
	root, err := a.source.RootEvent(db)
	if err != nil {
		return config.Body
	}
	sha := mergeconfidence.ReviewedCommit(root)
	if sha == "" {
		return config.Body
	}
	line := "Reviewed commit: " + sha
	lines := strings.Split(config.Body, "\n")
	index := -1
	for i, candidate := range lines {
		if candidate == line {
			index = i
			break
		}
	}
	if index < 0 {
		return config.Body
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	commit, err := client.GetCommit(lookupCtx, config.Repository, sha)
	if err != nil || commit == nil || commit.GetSHA() != sha {
		return config.Body
	}
	title, _, _ := strings.Cut(commit.GetCommit().GetMessage(), "\n")
	title = strings.TrimSpace(title)
	if title == "" {
		return config.Body
	}
	commitURL := client.CommitURL(config.Repository, sha)
	lines[index] = fmt.Sprintf("Reviewed commit: [%s](%s)", exprruntime.MarkdownText(title), commitURL)
	return strings.Join(lines, "\n")
}

func (a *bodySectionActionAttempt) integrationClient(db *gorm.DB, deps IntakeDependencies) (prbody.Client, error) {
	version, err := models.FindCanvasVersionInTransaction(db, a.source.WorkflowID, a.source.VersionID)
	if err != nil {
		return nil, err
	}
	var integrationID string
	for _, node := range version.Nodes {
		if node.ID == a.execution.NodeID && node.IntegrationID != nil {
			integrationID = *node.IntegrationID
			break
		}
	}
	if integrationID == "" {
		return nil, errBodySectionIntegrationUnavailable
	}
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, errBodySectionIntegrationUnavailable
	}
	integration, err := models.FindUnscopedIntegrationInTransaction(db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errBodySectionIntegrationUnavailable
	}
	if err != nil {
		return nil, err
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, a.source.WorkflowID)
	if err != nil {
		return nil, err
	}
	provider, _, _ := strings.Cut(a.component, ".")
	if integration.OrganizationID != canvas.OrganizationID || integration.AppName != provider || integration.DeletedAt.Valid {
		return nil, errBodySectionIntegrationUnavailable
	}
	integrationContext := contexts.NewIntegrationContext(db, a.node, integration, deps.Encryptor, deps.Registry, nil)
	http := deps.Registry.HTTPContextInTransaction(db)
	switch provider {
	case models.ProviderGitHub:
		return pulls.NewBodySectionClient(integrationContext, http)
	case models.ProviderBitbucket:
		return bitbucket.NewBodySectionClient(integrationContext, http, a.integrationConfiguration.Repository)
	default:
		return nil, fmt.Errorf("unsupported publication provider: %s", provider)
	}
}

// Integration actions work in ordinary canvases. Factory apps additionally fence
// publication against disabled settings and newer activity on the same PR.
func (a *bodySectionActionAttempt) integrationPublicationEligible(db *gorm.DB) (bool, error) {
	active, err := a.stillActive(db)
	if err != nil || !active {
		return false, err
	}
	config := a.integrationConfiguration
	publication := sectionPublication{run: a.source, nodeID: a.execution.NodeID, configuration: factorycomponent.UpdatePullRequestBodySectionConfiguration{SectionKey: config.SectionKey}}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, a.source.WorkflowID)
	if err != nil {
		return false, err
	}
	if canvas.FactoryID == nil {
		return true, nil
	}
	enabled, err := publication.nodesEnabled(db)
	if err != nil || !enabled {
		return false, err
	}
	workspace, err := models.FindFactory(db, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return false, err
	}
	number, err := config.Number()
	if err != nil {
		return false, err
	}
	provider, _, _ := strings.Cut(a.component, ".")
	pr, err := workspace.FindPullRequest(db, models.FactoryPullRequestLookup{Provider: provider, Repository: config.Repository, Number: number})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return publication.eligible(db, pr)
}
