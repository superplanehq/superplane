package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errBodySectionIntegrationUnavailable = errors.New("publication integration is unavailable")

// permanentPublicationError marks an ordered publication that retries cannot fix.
type permanentPublicationError struct{ cause error }

func (e permanentPublicationError) Error() string { return e.cause.Error() }
func (e permanentPublicationError) Unwrap() error { return e.cause }

// newPublicationConnection builds the provider access of an integration node.
// Tests replace it to substitute provider responses.
var newPublicationConnection = func(db *gorm.DB, deps IntakeDependencies, node *models.CanvasNode, integration *models.Integration) (core.IntegrationContext, core.HTTPContext) {
	return contexts.NewIntegrationContext(db, node, integration, deps.Encryptor, deps.Registry, nil), deps.Registry.HTTPContextInTransaction(db)
}

func isPublicationComponent(component string) bool {
	return prbody.IsComponent(component) || component == pulls.PullRequestCommentComponent
}

func publicationHook(component string) string {
	if component == pulls.PullRequestCommentComponent {
		return pulls.PullRequestCommentHook
	}
	return factorycomponent.UpdatePullRequestBodySectionHook
}

func (a *bodySectionActionAttempt) isComment() bool {
	return a.component == pulls.PullRequestCommentComponent
}

// Ordered publications apply source and attempt ordering at the write. They
// fail the step when a write cannot complete, so Runs shows the destination.
func (a *bodySectionActionAttempt) ordered() bool {
	return a.isComment() || (a.component != factorycomponent.UpdatePullRequestBodySectionComponentName && a.integrationConfiguration.Ordered)
}

func (a *bodySectionActionAttempt) destination() string {
	if a.isComment() {
		return "pull request comment"
	}
	return "pull request description"
}

func (a *bodySectionActionAttempt) enabled() bool {
	if a.isComment() {
		return a.commentConfiguration.IsEnabled()
	}
	return a.integrationConfiguration.IsEnabled()
}

func (a *bodySectionActionAttempt) publish(ctx context.Context, db *gorm.DB, deps IntakeDependencies) (bodySectionPublicationOutcome, error) {
	if a.component == factorycomponent.UpdatePullRequestBodySectionComponentName {
		return publishBodySection(ctx, db, deps, sectionPublication{run: a.source, nodeID: a.execution.NodeID, configuration: a.configuration, stillActive: a.stillActive})
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	db = db.WithContext(ctx)
	if !a.enabled() {
		return prbody.Skipped("Publication is disabled for this run."), nil
	}
	if a.ordered() && a.state.Publication == nil {
		return bodySectionPublicationOutcome{}, permanentPublicationError{fmt.Errorf("the publication was not accepted when the step started")}
	}
	eligible := func() (bool, error) { return a.integrationPublicationEligible(db) }
	active, err := eligible()
	if err != nil {
		return bodySectionPublicationOutcome{}, err
	}
	if !active {
		return prbody.Skipped("Publication is disabled, cancelled, or superseded."), nil
	}
	integration, http, err := a.integrationConnection(db, deps)
	if err != nil {
		return a.publicationError(err)
	}
	if a.isComment() {
		return a.publishComment(ctx, db, integration, http, eligible)
	}
	client, err := a.bodySectionClient(integration, http)
	if err != nil {
		return a.publicationError(err)
	}
	publications := a.publications(db)
	if a.ordered() {
		client = prbody.OrderedClient(client, publications, *a.state.Publication)
	}
	outcome, err := prbody.PublishSerialized(ctx, client, publications, a.integrationConfiguration, eligible)
	if err != nil {
		return a.publicationError(err)
	}
	return outcome, nil
}

func (a *bodySectionActionAttempt) publishComment(ctx context.Context, db *gorm.DB, integration core.IntegrationContext, http core.HTTPContext, eligible func() (bool, error)) (bodySectionPublicationOutcome, error) {
	client, err := pulls.NewCommentClient(integration, http)
	if err != nil {
		return a.publicationError(err)
	}
	boundary := pulls.NewOrderedCommentBoundary(a.publications(db), *a.state.Publication)
	outcome, err := pulls.PublishComment(ctx, client, a.commentConfiguration, a.commentConfiguration.Marker(a.execution.WorkflowID), eligible, boundary)
	if outcome.CommentID > 0 {
		a.state.CommentID = outcome.CommentID
	}
	if err != nil {
		return a.publicationError(err)
	}
	return outcome.Outcome, nil
}

func (a *bodySectionActionAttempt) publications(db *gorm.DB) *contexts.PublicationContext {
	return contexts.NewPublicationContext(db, a.organizationID, a.node, a.execution)
}

// publicationError keeps unordered behavior: host rejections are skipped.
func (a *bodySectionActionAttempt) publicationError(err error) (bodySectionPublicationOutcome, error) {
	if errors.Is(err, core.ErrPublicationSuperseded) {
		return prbody.Skipped(prbody.SupersededReason), nil
	}
	if !a.ordered() {
		return bodySectionPublicationError(err)
	}
	if errors.Is(err, errBodySectionIntegrationUnavailable) || !retryableDescriptionError(err) {
		return bodySectionPublicationOutcome{}, permanentPublicationError{err}
	}
	return bodySectionPublicationOutcome{}, fmt.Errorf("publish %s: %w", a.destination(), err)
}

func (a *bodySectionActionAttempt) integrationConnection(db *gorm.DB, deps IntakeDependencies) (core.IntegrationContext, core.HTTPContext, error) {
	version, err := models.FindCanvasVersionInTransaction(db, a.source.WorkflowID, a.source.VersionID)
	if err != nil {
		return nil, nil, err
	}
	var integrationID string
	for _, node := range version.Nodes {
		if node.ID == a.execution.NodeID && node.IntegrationID != nil {
			integrationID = *node.IntegrationID
			break
		}
	}
	if integrationID == "" {
		return nil, nil, errBodySectionIntegrationUnavailable
	}
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, nil, errBodySectionIntegrationUnavailable
	}
	integration, err := models.FindUnscopedIntegrationInTransaction(db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, errBodySectionIntegrationUnavailable
	}
	if err != nil {
		return nil, nil, err
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, a.source.WorkflowID)
	if err != nil {
		return nil, nil, err
	}
	provider, _, _ := strings.Cut(a.component, ".")
	if integration.OrganizationID != canvas.OrganizationID || integration.AppName != provider || integration.DeletedAt.Valid {
		return nil, nil, errBodySectionIntegrationUnavailable
	}
	connection, http := newPublicationConnection(db, deps, a.node, integration)
	return connection, http, nil
}

func (a *bodySectionActionAttempt) bodySectionClient(integration core.IntegrationContext, http core.HTTPContext) (prbody.Client, error) {
	provider, _, _ := strings.Cut(a.component, ".")
	switch provider {
	case models.ProviderGitHub:
		return pulls.NewBodySectionClient(integration, http)
	case models.ProviderBitbucket:
		return bitbucket.NewBodySectionClient(integration, http, a.integrationConfiguration.Repository)
	default:
		return nil, fmt.Errorf("unsupported publication provider: %s", provider)
	}
}

// Integration actions work in ordinary canvases. Factory apps additionally fence
// publication against disabled settings and newer activity on the same PR.
// Ordered publications fence saved activation and use persisted source and
// attempt order instead of pull request activity.
func (a *bodySectionActionAttempt) integrationPublicationEligible(db *gorm.DB) (bool, error) {
	active, err := a.stillActive(db)
	if err != nil || !active {
		return false, err
	}
	if a.ordered() {
		return a.orderedPublicationEligible(db)
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

func (a *bodySectionActionAttempt) orderedPublicationEligible(db *gorm.DB) (bool, error) {
	component, key := a.component, a.integrationConfiguration.SectionKey
	keyField := "sectionKey"
	if a.isComment() {
		key, keyField = a.commentConfiguration.CommentKey, "commentKey"
	}
	enabled, err := publicationNodesEnabled(db, a.source, a.execution.NodeID, func(node models.Node) bool {
		return node.ComponentName() == component && node.Configuration[keyField] == key
	})
	if err != nil || !enabled {
		return false, err
	}
	return core.IsCurrentOrderedWrite(a.publications(db), *a.state.Publication)
}
