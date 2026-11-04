package factories

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	prClosureTriggerNodeID = "on-pr-closed"
	riskScoreTriggerNodeID = "on-pr-risk"
)

const allPullRequestsDoneField = "whenAllPullRequestsDone"

// A workspace-scope closure app closes the task only after every attached
// pull request is done. Older closure apps close on the first one.
var prClosureCloseNodeIDs = []string{"complete-work-order", "reject-work-order"}

// syncWorkspaceRepositoryAccess grants the hosted GitHub App write token
// access to every workspace repository. Workspaces without a hosted app
// binding use a token that already covers the installation.
func syncWorkspaceRepositoryAccess(tx *gorm.DB, factory *models.Factory) error {
	config := factory.OnboardingConfigValue()
	if config.EffectiveVCSProvider() != models.ProviderGitHub {
		return nil
	}
	integrationID, err := uuid.Parse(strings.TrimSpace(config.VCSIntegrationID))
	if err != nil {
		return nil
	}
	_, err = models.FindVCSProviderIntegrationBinding(tx, integrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return models.SyncVCSProviderBindingRepositories(tx, integrationID, models.ProviderGitHub)
}

// refreshWorkspaceRepositoryAutomations points the pull request webhooks of
// the workspace automations at the current repository list. The list change
// is already saved, so a failure here is logged and does not fail the request.
func refreshWorkspaceRepositoryAutomations(ctx context.Context, deps IntakeDependencies, factory *models.Factory) {
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		return republishWorkspaceRepositoryAutomations(ctx, tx, deps, factory)
	})
	if err != nil {
		log.WithError(err).Warnf("factory %s: failed to refresh workspace repository automations", factory.ID)
	}
}

// republishWorkspaceRepositoryAutomations turns on the workspace scope on
// the pull request closure, merge confidence, and PR feedback nodes, then
// runs their Setup again so each webhook covers the current workspace
// repositories.
func republishWorkspaceRepositoryAutomations(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, factory *models.Factory) error {
	if factory.OnboardingConfigValue().EffectiveVCSProvider() != models.ProviderGitHub {
		return nil
	}
	options := changesets.CanvasPublisherOptions{
		Registry:       deps.Registry,
		OrgID:          factory.OrganizationID,
		Encryptor:      deps.Encryptor,
		AuthService:    deps.AuthService,
		WebhookBaseURL: deps.WebhookBaseURL,
	}
	if options.Validate() != nil {
		return nil
	}

	handlers, err := factory.ListPRFeedbackHandlers(tx)
	if err != nil {
		return err
	}
	handlerCanvasIDs := make(map[uuid.UUID]bool, len(handlers))
	for _, handler := range handlers {
		handlerCanvasIDs[handler.CanvasID] = true
	}

	canvasesForFactory, err := factory.ListCanvases(tx)
	if err != nil {
		return err
	}
	for i := range canvasesForFactory {
		canvas := &canvasesForFactory[i]
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return err
		}
		spec := models.LiveCanvasSpec{Nodes: liveVersion.Nodes, Edges: liveVersion.Edges}
		nodeIDs := workspaceScopeNodeIDs(spec, handlerCanvasIDs[canvas.ID])
		if len(nodeIDs) == 0 {
			continue
		}

		nodes, changed := withWorkspaceScope(slices.Clone(liveVersion.Nodes), nodeIDs)
		if slices.Contains(nodeIDs, prClosureTriggerNodeID) {
			nodes, changed = withAllPullRequestsClose(nodes, changed)
		}
		if changed {
			if err := canvases.PublishGeneratedCanvasNodesWithOwner(
				ctx,
				tx,
				canvas,
				nil,
				"Listen on all workspace repositories",
				nodes,
				slices.Clone(liveVersion.Edges),
				options,
			); err != nil {
				return err
			}
		}
		if err := changesets.RerunNodeSetup(ctx, tx, canvas, nodeIDs, options); err != nil {
			return err
		}
	}

	return ensureFactoryMergeabilityWebhook(ctx, tx, deps, factory)
}

// workspaceScopeNodeIDs lists the GitHub nodes that listen for pull request
// events of the workspace: the closure and merge confidence triggers and the
// PR feedback nodes.
func workspaceScopeNodeIDs(spec models.LiveCanvasSpec, prFeedback bool) []string {
	nodeIDs := []string{}
	if template, ok := resolveFactoryTemplate(spec.Nodes); ok {
		switch template.id {
		case "pr-closure":
			nodeIDs = append(nodeIDs, prClosureTriggerNodeID)
		case "risk-score":
			nodeIDs = append(nodeIDs, riskScoreTriggerNodeID)
		}
	}
	if prFeedback {
		graph := resolvePRFeedbackGraph(spec)
		nodeIDs = append(nodeIDs, graph.triggerNodeIDs()...)
		if graph.isChecks() {
			nodeIDs = append(nodeIDs, graph.WaitChecksNodeID)
		}
	}

	return slices.DeleteFunc(nodeIDs, func(nodeID string) bool {
		node := findIntakeNode(spec.Nodes, nodeID)
		return node == nil || !strings.HasPrefix(node.ComponentName(), "github.")
	})
}

// withWorkspaceScope points the given nodes at every workspace repository.
// A trigger uses the workspace expression. A wait node reads its repository
// from the event that started the run.
func withWorkspaceScope(nodes []models.Node, nodeIDs []string) ([]models.Node, bool) {
	changed := false
	for i := range nodes {
		if !slices.Contains(nodeIDs, nodes[i].ID) {
			continue
		}
		repository := models.WorkspaceRepositoriesExpression
		if nodes[i].ComponentName() == prFeedbackWaitChecksComponent {
			repository = prFeedbackEventRepositoryExpression
		}
		if nodes[i].Configuration["repository"] == repository {
			continue
		}
		configuration := maps.Clone(nodes[i].Configuration)
		if configuration == nil {
			configuration = map[string]any{}
		}
		configuration["repository"] = repository
		nodes[i].Configuration = configuration
		changed = true
	}
	return nodes, changed
}

func withAllPullRequestsClose(nodes []models.Node, changed bool) ([]models.Node, bool) {
	for i := range nodes {
		if !slices.Contains(prClosureCloseNodeIDs, nodes[i].ID) || nodes[i].ComponentName() != "updateWorkOrderStatus" {
			continue
		}
		if enabled, _ := nodes[i].Configuration[allPullRequestsDoneField].(bool); enabled {
			continue
		}
		configuration := maps.Clone(nodes[i].Configuration)
		if configuration == nil {
			configuration = map[string]any{}
		}
		configuration[allPullRequestsDoneField] = true
		nodes[i].Configuration = configuration
		changed = true
	}
	return nodes, changed
}
