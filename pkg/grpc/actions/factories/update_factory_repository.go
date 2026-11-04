package factories

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/runner"
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
	orderRepositoryExpression     = "{{ task().repository }}"
	orderDefaultBranchExpression  = "{{ task().default_branch }}"
	// Exact token baked into older Implement commands. Replace the token only.
	// A substring replace of "main" would change unrelated text.
	legacyCheckoutBaseToken = "${BASE:-main}"
	remoteHeadCheckoutBase  = "${BASE:-$(git symbolic-ref -q --short refs/remotes/origin/HEAD | sed 's#^origin/##' || true)}"
)

// UpdateFactoryRepository changes the repository that factory-managed
// automations use. It snapshots work orders before publishing canvas changes.
// Closed tasks keep that snapshot. When retarget_tasks is unset or true,
// unfinished tasks that still use the previous repository, and have no
// branch or the previous branch, move to the saved repository and branch.
// A false value keeps those tasks unchanged. A task with a running line
// stays unchanged.
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
	repository := strings.TrimSpace(req.GetRepository())
	if repository == "" {
		return nil, factoryErrorToStatus(invalidArgument("repository is required"), "failed to update factory repository")
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
	factory, err := findFactory(db, orgID, req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	if factory.OnboardingConfigValue().EffectiveVCSProvider() == models.ProviderBitbucket {
		return updateBitbucketFactoryRepository(ctx, deps, orgID, organizationID, actorID, req)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		loaded, err := models.FindFactory(tx, orgID, factory.ID)
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

		requestedBranch := strings.TrimSpace(req.GetDefaultBranch())
		repositoryChanged := !sameRepository(previous.AppRepository, repository)
		selectedRepository := repository
		hostDefault := ""
		selectedRepositoryID := int64(0)
		installationID := int64(0)
		selectedIntegration := integration
		_, bindingErr := models.FindVCSProviderIntegrationBinding(tx, integration.ID)
		if bindingErr == nil {
			catalogRepository, findErr := findGitHubCatalogRepositoryByName(ctx, tx, organizationID, repository)
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return grpcerrors.PermissionDenied(findErr, "VCS repository is not accessible")
			}
			if findErr != nil {
				return findErr
			}
			installation, findErr := models.FindVCSProviderInstallation(
				tx,
				models.ProviderGitHub,
				catalogRepository.InstallationID,
			)
			if findErr != nil {
				return findErr
			}
			selectedIntegration, findErr = models.FindOrCreateVCSProviderBinding(
				tx,
				orgID,
				models.ProviderGitHub,
				catalogRepository.InstallationID,
				installation.AccountLogin,
			)
			if findErr != nil {
				return findErr
			}
			selectedRepository = catalogRepository.FullName
			selectedRepositoryID = catalogRepository.RepositoryID
			installationID = catalogRepository.InstallationID
			hostDefault = catalogRepository.DefaultBranch
		} else if !errors.Is(bindingErr, gorm.ErrRecordNotFound) {
			return bindingErr
		}
		selectedDefaultBranch, err := chooseSavedBranch(repositoryChanged, requestedBranch, hostDefault)
		if err != nil {
			return err
		}
		if requestedBranch != "" {
			if err := ensureGitHubBranch(
				ctx,
				deps,
				selectedIntegration,
				selectedRepository,
				installationID,
				selectedRepositoryID,
				requestedBranch,
			); err != nil {
				return err
			}
		}
		if !repositoryChanged {
			selectedRepository = previous.AppRepository
			selectedRepositoryID = previous.AppRepositoryID
			selectedIntegration = integration
		}

		previousBranch, err := currentFactoryDefaultBranch(tx, factory, previous.DefaultBranch)
		if err != nil {
			return err
		}
		if err := factory.SnapshotWorkOrderRepository(tx, previous.AppRepository, previousBranch, previous.EffectiveVCSProvider()); err != nil {
			return err
		}

		selectedIntegrationID := selectedIntegration.ID.String()
		patch := models.FactoryOnboardingPatch{DefaultBranch: &selectedDefaultBranch}
		if repositoryChanged {
			patch.VCSIntegrationID = &selectedIntegrationID
			patch.AppRepository = &selectedRepository
			patch.AppRepositoryID = &selectedRepositoryID
			patch.BacklogRepository = &selectedRepository
			patch.BacklogRepositoryID = &selectedRepositoryID
		}
		if err := factory.UpdateOnboarding(tx, patch); err != nil {
			return err
		}
		if repositoryChanged && bindingErr == nil {
			if err := syncFactoryVCSProviderBindings(
				tx,
				models.ProviderGitHub,
				previous.VCSIntegrationID,
				selectedIntegration.ID,
			); err != nil {
				return err
			}
		}

		return reconcileFactoryRepository(
			ctx,
			tx,
			deps,
			factory,
			actorID,
			previous.VCSIntegrationID,
			selectedIntegrationID,
			previous.AppRepository,
			previous.BacklogRepository,
			previousBranch,
			selectedRepository,
			retargetTasksRequested(req.RetargetTasks),
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

func retargetTasksRequested(flag *bool) bool {
	if flag == nil {
		return true
	}
	return *flag
}

func reconcileFactoryRepository(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	actorID uuid.UUID,
	previousVCSIntegrationID, vcsIntegrationID string,
	previousAppRepository, previousBacklogRepository, previousDefaultBranch, repository string,
	retargetTasks bool,
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
	config := factory.OnboardingConfigValue()
	nextBranch := strings.TrimSpace(config.DefaultBranch)
	previousGitHubHost := githubHostForVCSIntegration(tx, factory.OrganizationID, previousVCSIntegrationID)
	nextGitHubHost := githubHostForVCSIntegration(tx, factory.OrganizationID, vcsIntegrationID)
	if retargetTasks {
		if err := factory.RetargetUnfinishedWorkOrderBranches(
			tx,
			previousAppRepository,
			repository,
			previousDefaultBranch,
			nextBranch,
			config.EffectiveVCSProvider(),
		); err != nil {
			return err
		}
	}
	previousIntegrationName, integrationName := "", ""
	if previousVCSIntegrationID != "" && previousVCSIntegrationID != vcsIntegrationID {
		previousID, err := uuid.Parse(previousVCSIntegrationID)
		if err != nil {
			return err
		}
		previousIntegration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, previousID)
		if err != nil {
			return err
		}
		selectedID, err := uuid.Parse(vcsIntegrationID)
		if err != nil {
			return err
		}
		integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, selectedID)
		if err != nil {
			return err
		}
		previousIntegrationName, integrationName = previousIntegration.InstallationName, integration.InstallationName
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
		managedVCSNodeIDs := map[string]bool{}
		managedRunnerNodeIDs := map[string]bool{}
		changed := false

		if template, ok := resolveFactoryTemplate(nodes); ok {
			for _, nodeID := range factoryTemplateVCSNodeIDs(template.id) {
				managedVCSNodeIDs[nodeID] = true
			}
			switch template.id {
			case "line-implementation":
				managedRunnerNodeIDs[implementationAgentNodeID] = true
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
			case "risk-score":
				managedRunnerNodeIDs["assess-risk"] = true
				changed = replaceNodeConfigurationValues(nodes, []configurationReplacement{{from: previousAppRepository, to: repository}}) || changed
			case "visual-evidence":
				managedRunnerNodeIDs["capture-visual-evidence"] = true
				changed = replaceNodeConfigurationValues(nodes, []configurationReplacement{{from: previousAppRepository, to: repository}}) || changed
			case scheduledIntakeTemplateID:
				changed = replaceWorkspaceCheckoutTarget(nodes, config.EffectiveVCSProvider(), previousAppRepository, repository, previousDefaultBranch, nextBranch, previousGitHubHost, nextGitHubHost) || changed
			}
			if template.id == "line-implementation" {
				changed = replaceLegacyCheckoutBaseToken(nodes, managedRunnerNodeIDs) || changed
			}
		}
		if models.FactoryAppTemplateID(nodes) == models.FactoryAppTemplateCustomAutomationID {
			changed = replaceWorkspaceCheckoutTarget(nodes, config.EffectiveVCSProvider(), previousAppRepository, repository, previousDefaultBranch, nextBranch, previousGitHubHost, nextGitHubHost) || changed
		}
		if _, ok := intakeCanvasIDs[canvas.ID]; ok {
			graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, models.LiveCanvasSpec{
				Nodes: nodes,
				Edges: liveVersion.Edges,
			})
			managedVCSNodeIDs[graph.TriggerNodeID] = true
			managedVCSNodeIDs[graph.AuthorPermissionNodeID] = true
			managedRunnerNodeIDs[graph.AnalysisNodeID] = true
			changed = replaceTriggerRepository(nodes, "github.onIssue", previousBacklogRepository, repository) || changed
			changed = replaceTriggerRepository(nodes, "github.onDependabotAlert", previousBacklogRepository, repository) || changed
		}
		if _, ok := handlerCanvasIDs[canvas.ID]; ok {
			spec := models.LiveCanvasSpec{
				Nodes: nodes,
				Edges: liveVersion.Edges,
			}
			graph := resolvePRFeedbackGraph(spec)
			addPRFeedbackGitHubNodeIDs(managedVCSNodeIDs, graph)
			if graph.isChecks() {
				managedRunnerNodeIDs[graph.RunnerNodeID] = true
			} else {
				maps.Copy(managedRunnerNodeIDs, graph.discussionRunnerNodeIDs(spec))
			}
			changed = replaceGitHubTriggerRepository(nodes, previousAppRepository, repository) || changed
		}
		changed = replaceVCSNodeIntegration(
			nodes,
			managedVCSNodeIDs,
			previousVCSIntegrationID,
			vcsIntegrationID,
			previousAppRepository,
			previousBacklogRepository,
			repository,
		) || changed
		changed = replaceRunnerVCSIntegration(nodes, managedRunnerNodeIDs, previousIntegrationName, integrationName) || changed
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
			},
		); err != nil {
			return fmt.Errorf("publish %q repository update: %w", canvas.Name, err)
		}
	}

	return ensureFactoryMergeabilityWebhook(ctx, tx, deps, factory)
}

func factoryTemplateVCSNodeIDs(templateID string) []string {
	switch templateID {
	case "line-implementation":
		return []string{"find-pr", "create-pr", "update-pr", "comment-visual-evidence", "comment-visual-evidence-updated"}
	case "pr-closure":
		return []string{"on-pr-closed", "comment-source-issue", "close-source-issue"}
	case "issue-intake":
		return []string{"on-issue-labeled", "on-issue-assigned"}
	case "risk-score":
		return []string{"on-pr-risk"}
	case "visual-evidence":
		return []string{"on-pr-visual-evidence", "read-pr-description", "update-pr-description"}
	default:
		return nil
	}
}

func addPRFeedbackGitHubNodeIDs(nodeIDs map[string]bool, graph prFeedbackGraph) {
	for _, nodeID := range graph.triggerNodeIDs() {
		nodeIDs[nodeID] = true
	}
	if graph.isChecks() {
		nodeIDs[graph.WaitChecksNodeID] = true
		return
	}

	nodeIDs[prFeedbackAcknowledgeCommentNodeID] = true
	nodeIDs[prFeedbackAcknowledgeReviewNodeID] = true
	nodeIDs[prFeedbackAcknowledgeReviewReplyNodeID] = true
}

func replaceVCSNodeIntegration(
	nodes []models.Node,
	managedNodeIDs map[string]bool,
	previousIntegrationID, integrationID, previousAppRepository, previousBacklogRepository, repository string,
) bool {
	if previousIntegrationID == "" || previousIntegrationID == integrationID {
		return false
	}

	changed := false
	for i := range nodes {
		component := nodes[i].ComponentName()
		if (!strings.HasPrefix(component, "github.") && !strings.HasPrefix(component, "bitbucket.")) || nodes[i].IntegrationID == nil {
			continue
		}
		if strings.TrimSpace(*nodes[i].IntegrationID) != previousIntegrationID {
			continue
		}
		if !managedNodeIDs[nodes[i].ID] && !configurationContainsRepository(
			nodes[i].Configuration,
			previousAppRepository,
			previousBacklogRepository,
			repository,
		) {
			continue
		}
		nodes[i].IntegrationID = &integrationID
		changed = true
	}
	return changed
}

func replaceRunnerVCSIntegration(nodes []models.Node, managedNodeIDs map[string]bool, previousName, name string) bool {
	if previousName == "" || previousName == name {
		return false
	}
	changed := false
	for i := range nodes {
		if !managedNodeIDs[nodes[i].ID] || !strings.HasPrefix(nodes[i].ComponentName(), "runner") {
			continue
		}
		environmentFrom, ok := nodes[i].Configuration["environmentFrom"].([]any)
		if !ok {
			continue
		}
		updated := slices.Clone(environmentFrom)
		nodeChanged := false
		for j, value := range environmentFrom {
			entry, ok := value.(map[string]any)
			if !ok || entry["source"] != "integration" {
				continue
			}
			ref, ok := entry["integration"].(map[string]any)
			if !ok || ref["name"] != previousName {
				continue
			}
			ref = maps.Clone(ref)
			ref["name"] = name
			entry = maps.Clone(entry)
			entry["integration"] = ref
			updated[j] = entry
			nodeChanged = true
		}
		if nodeChanged {
			nodes[i].Configuration = maps.Clone(nodes[i].Configuration)
			nodes[i].Configuration["environmentFrom"] = updated
			changed = true
		}
	}
	return changed
}

func configurationContainsRepository(value any, repositories ...string) bool {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if key == "repository" {
				repository, ok := child.(string)
				if ok && slices.ContainsFunc(repositories, func(candidate string) bool {
					return candidate != "" && repository == candidate
				}) {
					return true
				}
			}
			if configurationContainsRepository(child, repositories...) {
				return true
			}
		}
		return false
	case []any:
		return slices.ContainsFunc(current, func(child any) bool {
			return configurationContainsRepository(child, repositories...)
		})
	default:
		return false
	}
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
			if replacement.from != "" && replacement.from != replacement.to && current == replacement.from {
				return replacement.to, true
			}
		}
		return current, false
	default:
		return value, false
	}
}

// replaceWorkspaceCheckoutTarget keeps a baked checkout on the workspace
// repository and branch together. A repository change rewrites the repository
// slug and its clone URL. A branch change rewrites only BASE.
func replaceWorkspaceCheckoutTarget(nodes []models.Node, provider, previousRepository, nextRepository, previousBranch, nextBranch, previousGitHubHost, nextGitHubHost string) bool {
	changed := replaceLiteralCheckoutBase(nodes, previousBranch, nextBranch)
	previousRepository = strings.TrimSpace(previousRepository)
	nextRepository = strings.TrimSpace(nextRepository)
	if previousRepository == "" || nextRepository == "" || sameRepository(previousRepository, nextRepository) {
		return changed
	}
	return replaceNodeConfigurationValues(nodes, []configurationReplacement{
		{from: previousRepository, to: nextRepository},
		{from: runner.WorkspaceContextRepoURL(provider, previousRepository, previousGitHubHost), to: runner.WorkspaceContextRepoURL(provider, nextRepository, nextGitHubHost)},
	}) || changed
}

// replaceLiteralCheckoutBase rewrites a BASE value that is exactly the previous
// workspace branch. It leaves every other scalar unchanged.
func replaceLiteralCheckoutBase(nodes []models.Node, previousBranch, nextBranch string) bool {
	previousBranch = strings.TrimSpace(previousBranch)
	nextBranch = strings.TrimSpace(nextBranch)
	if nextBranch == "" || previousBranch == nextBranch {
		return false
	}

	changed := false
	for i := range nodes {
		environment, ok := nodes[i].Configuration["environment"].([]any)
		if !ok {
			continue
		}
		nextEnvironment := slices.Clone(environment)
		updated := false
		for j, item := range nextEnvironment {
			entry, ok := item.(map[string]any)
			if !ok || entry["name"] != "BASE" {
				continue
			}
			value, _ := entry["value"].(string)
			if strings.TrimSpace(value) != previousBranch {
				continue
			}
			cloned := maps.Clone(entry)
			cloned["value"] = nextBranch
			nextEnvironment[j] = cloned
			updated = true
		}
		if !updated {
			continue
		}
		nodes[i].Configuration = maps.Clone(nodes[i].Configuration)
		nodes[i].Configuration["environment"] = nextEnvironment
		changed = true
	}
	return changed
}

// replaceLegacyCheckoutBaseToken rewrites the exact ${BASE:-main} token inside
// managed Implement commands. It does not replace the word main.
func replaceLegacyCheckoutBaseToken(nodes []models.Node, managedNodeIDs map[string]bool) bool {
	changed := false
	for i := range nodes {
		if !managedNodeIDs[nodes[i].ID] {
			continue
		}
		configuration, nodeChanged := replaceCommandToken(nodes[i].Configuration, legacyCheckoutBaseToken, remoteHeadCheckoutBase)
		if !nodeChanged {
			continue
		}
		nodes[i].Configuration = configuration.(map[string]any)
		changed = true
	}
	return changed
}

func replaceCommandToken(value any, token, replacement string) (any, bool) {
	switch current := value.(type) {
	case map[string]any:
		cloned := maps.Clone(current)
		changed := false
		for key, child := range cloned {
			if key == "command" {
				text, ok := child.(string)
				if ok && strings.Contains(text, token) {
					cloned[key] = strings.ReplaceAll(text, token, replacement)
					changed = true
					continue
				}
			}
			next, childChanged := replaceCommandToken(child, token, replacement)
			cloned[key] = next
			changed = changed || childChanged
		}
		return cloned, changed
	case []any:
		cloned := slices.Clone(current)
		changed := false
		for i, child := range cloned {
			next, childChanged := replaceCommandToken(child, token, replacement)
			cloned[i] = next
			changed = changed || childChanged
		}
		return cloned, changed
	default:
		return value, false
	}
}
