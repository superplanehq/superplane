package factories

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

const (
	legacyAppRepositoryExpression = "{{ install_params.appRepository }}"
	legacyDefaultBranchExpression = "{{ install_params.defaultBranch }}"
	orderRepositoryExpression     = "{{ order().repository }}"
	orderDefaultBranchExpression  = "{{ order().default_branch }}"
)

// UpdateFactoryRepository changes the repository that factory-managed
// automations use. It snapshots active work orders before publishing the
// generated canvas changes, so an in-progress order stays on its original
// repository and branch.
func UpdateFactoryRepository(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryRepositoryRequest,
) (*pb.UpdateFactoryRepositoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	factoryID, err := parseFactoryID(req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	repository := strings.TrimSpace(req.GetRepository())
	defaultBranch := strings.TrimSpace(req.GetDefaultBranch())
	if repository == "" {
		return nil, factoryErrorToStatus(invalidArgument("repository is required"), "failed to update factory repository")
	}
	if defaultBranch == "" {
		return nil, factoryErrorToStatus(invalidArgument("default branch is required"), "failed to update factory repository")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to update factory repository")
	}

	db := database.DB(ctx)
	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		loaded, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}
		factory = loaded

		previous := factory.OnboardingConfigValue()
		if _, err := factory.OnboardingConfigAfter(models.FactoryOnboardingPatch{AppRepository: &repository}); err != nil {
			return err
		}
		if previous.VCSIntegrationID == "" {
			return invalidArgument("connect GitHub before selecting a repository")
		}
		integration, err := findReadyOnboardingIntegration(tx, orgID, previous.VCSIntegrationID)
		if err != nil {
			return err
		}
		if integration.AppName != "github" {
			return invalidArgument("repository settings currently require a GitHub integration")
		}

		previousBranch, err := currentFactoryDefaultBranch(tx, factory, previous.DefaultBranch)
		if err != nil {
			return err
		}
		if err := factory.SnapshotActiveWorkOrderRepository(tx, previous.AppRepository, previousBranch); err != nil {
			return err
		}

		patch := models.FactoryOnboardingPatch{
			AppRepository:     &repository,
			BacklogRepository: &repository,
			DefaultBranch:     &defaultBranch,
		}
		if err := factory.UpdateOnboarding(tx, patch); err != nil {
			return err
		}

		return reconcileFactoryRepository(
			ctx,
			tx,
			deps,
			factory,
			actorID,
			previous.AppRepository,
			previous.BacklogRepository,
			previousBranch,
			repository,
		)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	return &pb.UpdateFactoryRepositoryResponse{Factory: serialized}, nil
}

// currentFactoryDefaultBranch resolves the branch used before a repository
// switch. Older factories did not persist this setting, so read it from the
// generated implementation canvas instead of assuming a default.
func currentFactoryDefaultBranch(tx *gorm.DB, factory *models.Factory, configuredBranch string) (string, error) {
	if configuredBranch = strings.TrimSpace(configuredBranch); configuredBranch != "" {
		return configuredBranch, nil
	}

	canvasesForFactory, err := factory.ListCanvases(tx)
	if err != nil {
		return "", err
	}
	for i := range canvasesForFactory {
		canvas := &canvasesForFactory[i]
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return "", err
		}
		template, ok := resolveFactoryTemplate(version.Nodes)
		if !ok || template.id != "line-implementation" {
			continue
		}
		if defaultBranch := factoryDefaultBranchFromNodes(version.Nodes); defaultBranch != "" {
			return defaultBranch, nil
		}
	}

	return "", invalidArgument("could not determine the current workspace default branch")
}

func factoryDefaultBranchFromNodes(nodes []models.Node) string {
	defaultBranch := strings.TrimSpace(deriveFactoryInstallParams(nodes)["defaultBranch"])
	if strings.HasPrefix(defaultBranch, "{{") {
		return ""
	}
	return defaultBranch
}

func reconcileFactoryRepository(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	actorID uuid.UUID,
	previousAppRepository, previousBacklogRepository, previousDefaultBranch, repository string,
) error {
	canvasesForFactory, err := factory.ListCanvases(tx)
	if err != nil {
		return err
	}
	intakes, err := factory.ListIntakes(tx)
	if err != nil {
		return err
	}
	handlers, err := factory.ListPRFeedbackHandlers(tx)
	if err != nil {
		return err
	}

	intakeCanvasIDs := make(map[uuid.UUID]struct{}, len(intakes))
	for _, intake := range intakes {
		if intake.Source == models.FactoryIntakeSourceGitHubIssues {
			intakeCanvasIDs[intake.CanvasID] = struct{}{}
		}
	}
	handlerCanvasIDs := make(map[uuid.UUID]struct{}, len(handlers))
	for _, handler := range handlers {
		handlerCanvasIDs[handler.CanvasID] = struct{}{}
	}

	for i := range canvasesForFactory {
		canvas := &canvasesForFactory[i]
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return err
		}
		nodes := slices.Clone(liveVersion.Nodes)
		changed := false

		if template, ok := resolveFactoryTemplate(nodes); ok {
			switch template.id {
			case "line-planning", "line-implementation":
				changed = replaceNodeConfigurationValues(nodes, []configurationReplacement{
					{from: previousAppRepository, to: orderRepositoryExpression},
					{from: previousDefaultBranch, to: orderDefaultBranchExpression},
					{from: legacyAppRepositoryExpression, to: orderRepositoryExpression},
					{from: legacyDefaultBranchExpression, to: orderDefaultBranchExpression},
				}) || changed
			case "pr-closure":
				changed = replaceNodeConfigurationValues(nodes, []configurationReplacement{{from: previousAppRepository, to: repository}}) || changed
			case "issue-intake":
				changed = replaceNodeConfigurationValues(nodes, []configurationReplacement{{from: previousBacklogRepository, to: repository}}) || changed
			}
		}
		if _, ok := intakeCanvasIDs[canvas.ID]; ok {
			changed = replaceTriggerRepository(nodes, "github.onIssue", previousBacklogRepository, repository) || changed
		}
		if _, ok := handlerCanvasIDs[canvas.ID]; ok {
			changed = replaceGitHubTriggerRepository(nodes, previousAppRepository, repository) || changed
		}
		if !changed {
			continue
		}

		if err := canvases.PublishGeneratedCanvasNodes(
			ctx,
			tx,
			canvas,
			actorID,
			"Update workspace repository",
			nodes,
			slices.Clone(liveVersion.Edges),
			changesets.CanvasPublisherOptions{
				Registry:       deps.Registry,
				OrgID:          canvas.OrganizationID,
				Encryptor:      deps.Encryptor,
				AuthService:    deps.AuthService,
				WebhookBaseURL: deps.WebhookBaseURL,
				GitProvider:    deps.GitProvider,
			},
		); err != nil {
			return fmt.Errorf("publish %q repository update: %w", canvas.Name, err)
		}
	}

	return nil
}

func replaceTriggerRepository(nodes []models.Node, component, previousRepository, repository string) bool {
	changed := false
	for i := range nodes {
		if nodes[i].ComponentName() != component {
			continue
		}
		configuration, nodeChanged := replaceConfigurationValues(nodes[i].Configuration, []configurationReplacement{{from: previousRepository, to: repository}})
		if !nodeChanged {
			continue
		}
		nodes[i].Configuration = configuration.(map[string]any)
		changed = true
	}
	return changed
}

func replaceGitHubTriggerRepository(nodes []models.Node, previousRepository, repository string) bool {
	changed := false
	for i := range nodes {
		if !strings.HasPrefix(nodes[i].ComponentName(), "github.on") {
			continue
		}
		configuration, nodeChanged := replaceConfigurationValues(nodes[i].Configuration, []configurationReplacement{{from: previousRepository, to: repository}})
		if !nodeChanged {
			continue
		}
		nodes[i].Configuration = configuration.(map[string]any)
		changed = true
	}
	return changed
}

type configurationReplacement struct {
	from string
	to   string
}

func replaceNodeConfigurationValues(nodes []models.Node, replacements []configurationReplacement) bool {
	changed := false
	for i := range nodes {
		configuration, nodeChanged := replaceConfigurationValues(nodes[i].Configuration, replacements)
		if !nodeChanged {
			continue
		}
		nodes[i].Configuration = configuration.(map[string]any)
		changed = true
	}
	return changed
}

// replaceConfigurationValues updates only complete scalar values. Canvas
// configuration can contain shell scripts and prose, so replacing substrings
// would corrupt unrelated values such as a main branch fallback.
func replaceConfigurationValues(value any, replacements []configurationReplacement) (any, bool) {
	switch current := value.(type) {
	case map[string]any:
		cloned := maps.Clone(current)
		changed := false
		for key, child := range cloned {
			next, childChanged := replaceConfigurationValues(child, replacements)
			cloned[key] = next
			changed = changed || childChanged
		}
		return cloned, changed
	case []any:
		cloned := slices.Clone(current)
		changed := false
		for i, child := range cloned {
			next, childChanged := replaceConfigurationValues(child, replacements)
			cloned[i] = next
			changed = changed || childChanged
		}
		return cloned, changed
	case string:
		for _, replacement := range replacements {
			if replacement.from != "" && current == replacement.from {
				return replacement.to, true
			}
		}
		return current, false
	default:
		return value, false
	}
}
