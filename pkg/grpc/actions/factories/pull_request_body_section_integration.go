package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
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
	outcome, err := prbody.Publish(ctx, client, config, eligible)
	if err != nil {
		return bodySectionPublicationError(err)
	}
	return outcome, nil
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
