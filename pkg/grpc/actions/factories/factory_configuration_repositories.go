package factories

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type factoryConfigurationRepositories struct {
	Binding           factoryConfigurationBinding             `yaml:"binding"`
	Primary           factoryConfigurationRepository          `yaml:"primary"`
	BacklogRepository string                                  `yaml:"backlog_repository,omitempty"`
	Additional        []factoryConfigurationContextRepository `yaml:"additional,omitempty"`
}

type factoryConfigurationRepository struct {
	Repository string `yaml:"repository"`
	Ref        string `yaml:"ref,omitempty"`
}

type factoryConfigurationContextRepository struct {
	Repository string `yaml:"repository"`
	Ref        string `yaml:"ref,omitempty"`
	Access     string `yaml:"access"`
	Enabled    *bool  `yaml:"enabled,omitempty"`
	Provider   string `yaml:"provider,omitempty"`
}

type preparedFactoryConfigurationRepositories struct {
	Desired             factoryConfigurationRepositories
	Integration         configurationIntegrationState
	Configuration       models.FactoryOnboardingConfig
	GitHubRepositoryIDs []int64
	patch               models.FactoryOnboardingPatch
	factory             *models.Factory
	previousBranch      string
}

func validateConfigurationBindingYAML(node *yaml.Node, field string, allowed []string) (map[string]*yaml.Node, *pb.FactoryConfigurationDiagnostic) {
	fields, diagnostic := configurationFields(node, field, allowed, nil)
	if diagnostic != nil {
		return nil, diagnostic
	}
	if fields["integration_id"] != nil && fields["integration_name"] != nil {
		return nil, configurationDiagnostic(field+".integration_name", "Use either integration_name or integration_id.")
	}
	for key, value := range fields {
		if diagnostic := configurationStringNode(value, field+"."+key); diagnostic != nil {
			return nil, diagnostic
		}
		if (key == "integration_name" || key == "integration_id") && strings.TrimSpace(value.Value) == "" {
			return nil, configurationDiagnostic(field+"."+key, "Select an existing integration.")
		}
	}
	return fields, nil
}

func validateConfigurationRepositoriesYAML(node *yaml.Node) *pb.FactoryConfigurationDiagnostic {
	fields, diagnostic := configurationFields(node, "repositories", []string{"binding", "primary", "additional", "backlog_repository"}, []string{"binding", "primary"})
	if diagnostic != nil {
		return diagnostic
	}
	binding, diagnostic := validateConfigurationBindingYAML(fields["binding"], "repositories.binding", []string{"integration_id", "integration_name"})
	if diagnostic != nil {
		return diagnostic
	}
	if binding["integration_id"] == nil && binding["integration_name"] == nil {
		return configurationDiagnostic("repositories.binding", "Select an existing integration with integration_name or integration_id.")
	}
	primary, diagnostic := configurationFields(fields["primary"], "repositories.primary", []string{"repository", "ref"}, []string{"repository"})
	if diagnostic != nil {
		return diagnostic
	}
	for field, value := range primary {
		if diagnostic := configurationStringNode(value, "repositories.primary."+field); diagnostic != nil {
			return diagnostic
		}
	}
	if value := fields["backlog_repository"]; value != nil {
		if diagnostic := configurationStringNode(value, "repositories.backlog_repository"); diagnostic != nil {
			return diagnostic
		}
	}
	if additional := fields["additional"]; additional != nil {
		if additional.Kind != yaml.SequenceNode {
			return configurationDiagnostic("repositories.additional", "Use a repository list, including an empty list to remove additional repositories.")
		}
		for i, node := range additional.Content {
			field := fmt.Sprintf("repositories.additional[%d]", i)
			entry, diagnostic := configurationFields(node, field, []string{"repository", "ref", "access", "enabled", "provider"}, []string{"repository", "access"})
			if diagnostic != nil {
				return diagnostic
			}
			for key, value := range entry {
				if key == "enabled" {
					if value.Kind != yaml.ScalarNode || value.Tag != "!!bool" {
						return configurationDiagnostic(field+".enabled", "Use an explicit boolean value.")
					}
					continue
				}
				if diagnostic := configurationStringNode(value, field+"."+key); diagnostic != nil {
					return diagnostic
				}
			}
			if access := entry["access"].Value; access != "read" && access != "write" {
				return configurationDiagnostic(field+".access", "Use read or write.")
			}
		}
	}
	return nil
}

func configurationContextRepository(resource models.FactoryAgentResource) factoryConfigurationContextRepository {
	config := resource.Config.Data().NormalizedContextRepo()
	enabled := resource.Enabled
	return factoryConfigurationContextRepository{Repository: config.Repository, Ref: config.Ref, Access: config.Access, Enabled: &enabled, Provider: config.VCSProvider}
}

func (r factoryConfigurationContextRepository) config(provider string) models.FactoryAgentResourceConfig {
	if r.Provider != "" {
		provider = r.Provider
	}
	return models.FactoryAgentResourceConfig{Repository: r.Repository, Ref: r.Ref, Access: r.Access, VCSProvider: provider}
}

func (r factoryConfigurationContextRepository) enabled() bool { return r.Enabled == nil || *r.Enabled }

func configurationRepositoryKey(provider, repository string) string {
	if provider == "" {
		provider = models.ProviderGitHub
	}
	return provider + ":" + models.CanonicalContextRepo(repository)
}

func normalizeConfigurationRepositories(repositories factoryConfigurationRepositories, provider string) factoryConfigurationRepositories {
	if repositories.BacklogRepository == "" {
		repositories.BacklogRepository = repositories.Primary.Repository
	}
	repositories.Additional = slices.Clone(repositories.Additional)
	for i := range repositories.Additional {
		entry := &repositories.Additional[i]
		if entry.Provider == "" {
			entry.Provider = provider
		}
		enabled := entry.enabled()
		entry.Enabled = &enabled
	}
	slices.SortFunc(repositories.Additional, func(a, b factoryConfigurationContextRepository) int {
		return strings.Compare(configurationRepositoryKey(a.Provider, a.Repository), configurationRepositoryKey(b.Provider, b.Repository))
	})
	if len(repositories.Additional) == 0 {
		repositories.Additional = nil
	}
	return repositories
}

func loadFactoryConfigurationRepositories(tx *gorm.DB, target *factoryConfigurationTarget) error {
	resources, err := target.factory.ListContextRepos(tx)
	if err != nil {
		return err
	}
	target.repositoryRecords = resources
	config := target.factory.OnboardingConfigValue()
	if config.AppRepository == "" {
		return nil
	}
	repositories := &factoryConfigurationRepositories{Binding: factoryConfigurationBinding{IntegrationID: config.VCSIntegrationID}, Primary: factoryConfigurationRepository{Repository: config.AppRepository, Ref: config.DefaultBranch}}
	if repositories.Primary.Ref == "" {
		for _, canvas := range target.reads.canvases {
			version := target.reads.versions[canvas.ID]
			if version == nil {
				continue
			}
			template, ok := resolveFactoryTemplate(version.Nodes)
			if ok && template.id == lineImplementationTemplateID {
				repositories.Primary.Ref = factoryDefaultBranchFromNodes(version.Nodes)
				if repositories.Primary.Ref != "" {
					break
				}
			}
		}
	}
	if config.BacklogRepository != config.AppRepository {
		repositories.BacklogRepository = config.BacklogRepository
	}
	for _, resource := range resources {
		repositories.Additional = append(repositories.Additional, configurationContextRepository(resource))
	}
	normalized := normalizeConfigurationRepositories(*repositories, config.EffectiveVCSProvider())
	if normalized.BacklogRepository == normalized.Primary.Repository {
		normalized.BacklogRepository = ""
	}
	repositories = &normalized
	repositories.Binding = *portableConfigurationBinding(target.reads.integrations, &repositories.Binding)
	target.repositories = repositories
	return nil
}

func prepareFactoryConfigurationRepositories(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, requested *factoryConfigurationRepositories) (*pb.FactoryConfigurationDiagnostic, error) {
	if requested == nil {
		return nil, nil
	}
	resolved, err := resolveConfigurationBindingName(tx, target.factory.OrganizationID, &requested.Binding)
	if err != nil {
		return repositoryConfigurationDiagnostic(err, "repositories", "")
	}
	selection := *requested
	selection.Binding = *resolved
	selection.Binding.ResourceID = selection.Primary.Repository
	_, integration, err := resolveConfigurationBuiltInBinding(tx, target.factory, "line-implementation", &selection.Binding)
	if err != nil {
		return repositoryConfigurationDiagnostic(err, "repositories", requested.Binding.IntegrationName)
	}
	if err := rejectFactoryProviderSwitch(target.factory, integration.AppName); err != nil {
		return configurationDiagnostic("repositories.binding", err.Error()), nil
	}
	selection.Binding.ResourceID = ""
	if selection.BacklogRepository == "" {
		selection.BacklogRepository = selection.Primary.Repository
	}
	selection.Additional = slices.Clone(selection.Additional)
	selection.Primary.Ref = strings.TrimSpace(selection.Primary.Ref)
	selection.BacklogRepository = strings.TrimSpace(selection.BacklogRepository)
	primary := models.FactoryAgentResourceConfig{Repository: selection.Primary.Repository, Ref: selection.Primary.Ref, VCSProvider: integration.AppName}
	if err := primary.ValidateContextRepo(); err != nil {
		return configurationDiagnostic("repositories.primary", err.Error()), nil
	}
	if err := (models.FactoryAgentResourceConfig{Repository: selection.BacklogRepository}).ValidateContextRepo(); err != nil {
		return configurationDiagnostic("repositories.backlog_repository", err.Error()), nil
	}
	previous := target.factory.OnboardingConfigValue()
	primaryID, externalID, err := authorizeConfigurationPrimaryRepository(ctx, tx, target.factory, integration, selection.Primary.Repository)
	if err != nil {
		return configurationRepositoryAccessDiagnostic(err, "repositories.primary.repository")
	}
	backlogID := primaryID
	if !sameRepository(selection.Primary.Repository, selection.BacklogRepository) {
		backlogBinding := selection.Binding
		backlogBinding.ResourceID = selection.BacklogRepository
		if _, _, err := resolveConfigurationBuiltInBinding(tx, target.factory, lineImplementationTemplateID, &backlogBinding); err != nil {
			return configurationRepositoryAccessDiagnostic(err, "repositories.backlog_repository")
		}
		backlogID, _, err = authorizeConfigurationPrimaryRepository(ctx, tx, target.factory, integration, selection.BacklogRepository)
		if err != nil {
			return configurationRepositoryAccessDiagnostic(err, "repositories.backlog_repository")
		}
	}
	patch := models.FactoryOnboardingPatch{VCSIntegrationID: &selection.Binding.IntegrationID, VCSProvider: &integration.AppName, AppRepository: &selection.Primary.Repository, AppRepositoryID: &primaryID, AppRepositoryExternalID: &externalID, BacklogRepository: &selection.BacklogRepository, BacklogRepositoryID: &backlogID, DefaultBranch: &selection.Primary.Ref}
	// Older workspaces already follow the provider default without a saved branch.
	keepsProviderDefault := previous.DefaultBranch == "" && sameRepository(selection.Primary.Repository, previous.AppRepository) && selection.Binding.IntegrationID == previous.VCSIntegrationID
	if selection.Primary.Ref == "" && !keepsProviderDefault {
		selection.Primary.Ref = onboardingProviderDefaultBranch(ctx, deps, tx, target.factory.OrganizationID, target.factory.OrganizationID.String(), target.factory, patch)
		if selection.Primary.Ref == "" {
			return configurationDiagnostic("repositories.primary.ref", "The provider default branch is unavailable. Set ref or sync the connection and plan again."), nil
		}
	}
	savedBranch := previous.DefaultBranch
	if target.repositories != nil {
		savedBranch = target.repositories.Primary.Ref
	}
	if requested.Primary.Ref != "" && (selection.Primary.Ref != savedBranch || !sameRepository(selection.Primary.Repository, previous.AppRepository) || selection.Binding.IntegrationID != previous.VCSIntegrationID) {
		if integration.AppName == models.ProviderBitbucket {
			err = ensureBitbucketBranch(ctx, tx, deps, integration, selection.Primary.Repository, selection.Primary.Ref)
		} else {
			var installationID, repositoryID int64
			installationID, repositoryID, err = githubInstallationForRepository(ctx, tx, target.factory.OrganizationID.String(), integration, selection.Primary.Repository)
			if err != nil {
				return configurationRepositoryAccessDiagnostic(err, "repositories.primary.repository")
			}
			err = ensureGitHubBranch(ctx, deps, integration, selection.Primary.Repository, installationID, repositoryID, selection.Primary.Ref)
		}
		if err != nil {
			return configurationDiagnostic("repositories.primary.ref", err.Error()), nil
		}
	}
	merged, err := target.factory.OnboardingConfigAfter(patch)
	if err != nil {
		return configurationDiagnostic("repositories", err.Error()), nil
	}
	desiredFactory := *target.factory
	desiredFactory.OnboardingConfig = datatypes.NewJSONType(merged)
	seen := map[string]bool{}
	enabledCount := 0
	var githubRepositoryIDs []int64
	for i, entry := range selection.Additional {
		field := fmt.Sprintf("repositories.additional[%d]", i)
		config := entry.config(integration.AppName)
		if err := config.ValidateContextRepo(); err != nil {
			return configurationDiagnostic(field, err.Error()), nil
		}
		config = config.NormalizedContextRepo()
		selection.Additional[i] = factoryConfigurationContextRepository{Repository: config.Repository, Ref: config.Ref, Access: config.Access, Enabled: entry.Enabled, Provider: config.VCSProvider}
		key := configurationRepositoryKey(config.VCSProvider, config.Repository)
		if seen[key] {
			return configurationDiagnostic(field+".repository", "Use each repository once per provider."), nil
		}
		seen[key] = true
		if entry.enabled() {
			enabledCount++
		}
		if config.VCSProvider == models.ProviderGitHub && integration.AppName == models.ProviderGitHub {
			repositoryID, _, err := authorizeConfigurationPrimaryRepository(ctx, tx, &desiredFactory, integration, config.Repository)
			if err != nil {
				return configurationRepositoryAccessDiagnostic(err, field+".repository")
			}
			if repositoryID != 0 {
				githubRepositoryIDs = append(githubRepositoryIDs, repositoryID)
			}
			continue
		}
		if err := authorizeContextRepository(ctx, tx, deps, target.factory.OrganizationID.String(), &desiredFactory, config); err != nil {
			return configurationRepositoryAccessDiagnostic(err, field+".repository")
		}
	}
	if enabledCount > models.MaxEnabledFactoryContextRepos {
		return configurationDiagnostic("repositories.additional", models.ErrFactoryAgentResourceContextRepoCapReached.Error()), nil
	}
	selection = normalizeConfigurationRepositories(selection, integration.AppName)
	target.repositoryUpdate = &preparedFactoryConfigurationRepositories{Desired: selection, Integration: configurationIntegrationSnapshot(integration), Configuration: merged, GitHubRepositoryIDs: githubRepositoryIDs, patch: patch, factory: &desiredFactory}
	return nil, nil
}

func repositoryConfigurationDiagnostic(err error, field, integrationName string) (*pb.FactoryConfigurationDiagnostic, error) {
	var setting *factoryConfigurationSettingError
	if !errors.As(err, &setting) {
		return nil, err
	}
	child := setting.Field
	if child == "binding.resource_id" {
		child = "primary.repository"
	}
	if child == "binding.integration_id" && integrationName != "" {
		child = "binding.integration_name"
	}
	return configurationDiagnostic(field+"."+child, setting.Message), nil
}

func authorizeConfigurationPrimaryRepository(ctx context.Context, tx *gorm.DB, factory *models.Factory, integration *models.Integration, repository string) (int64, string, error) {
	if integration.AppName == models.ProviderBitbucket {
		if installation := forgeInstallationID(integration); installation != "" {
			account, err := linkedBitbucketAccountID(ctx, tx, factory.OrganizationID.String())
			if err != nil {
				return 0, "", err
			}
			visible, err := findVisibleBitbucketRepository(ctx, tx, account, repository)
			if err != nil {
				return 0, "", err
			}
			if visible == nil || visible.InstallationID != installation {
				return 0, "", invalidArgument("The repository is not accessible to this connection.")
			}
			return 0, visible.UUID, nil
		}
		return 0, "", validateBitbucketTokenRepository(integration, repository)
	}
	binding, err := factoryAutomationVCSBinding(tx, factory, integration.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	catalog, err := factoryAutomationPolicy(tx, factory, "configuration-repository-access", []any{integration.ID, repository, versionOwnerID(ctx), githubapp.AccountConnectionRequired(ctx)}, func() (*models.AccessibleVCSProviderRepository, error) {
		return findGitHubCatalogRepositoryByName(ctx, tx, factory.OrganizationID.String(), repository, integration.ID)
	})
	if err != nil {
		return 0, "", err
	}
	if catalog.InstallationID != binding.InstallationID {
		return 0, "", invalidArgument("The repository is not accessible to this connection.")
	}
	return catalog.RepositoryID, "", nil
}

func configurationPreparationTarget(target *factoryConfigurationTarget) *factoryConfigurationTarget {
	if !configurationRepositoriesChanged(target) {
		return target
	}
	prepared := *target
	prepared.factory = target.repositoryUpdate.factory
	reads := *target.reads
	reads.factory = prepared.factory
	reads.policies = map[string]any{}
	reads.agentResolved, reads.hostedAgentResolved, reads.vcsResolved = false, false, false
	repositories := []string{target.repositoryUpdate.Desired.Primary.Repository}
	seen := map[string]bool{models.CanonicalContextRepo(repositories[0]): true}
	for _, entry := range target.repositoryUpdate.Desired.Additional {
		if !entry.enabled() || entry.Access != "write" || entry.Provider != target.repositoryUpdate.Integration.Provider {
			continue
		}
		key := models.CanonicalContextRepo(entry.Repository)
		if !seen[key] {
			repositories = append(repositories, entry.Repository)
			seen[key] = true
		}
	}
	reads.workspaceRepositories = repositories
	prepared.reads = &reads
	return &prepared
}

func configurationRepositoriesChanged(target *factoryConfigurationTarget) bool {
	if target.repositoryUpdate == nil {
		return false
	}
	if target.repositories == nil {
		return true
	}
	current := *target.repositories
	current.Binding = factoryConfigurationBinding{IntegrationID: target.factory.OnboardingConfigValue().VCSIntegrationID}
	current = normalizeConfigurationRepositories(current, target.factory.OnboardingConfigValue().EffectiveVCSProvider())
	return !equalCanvasJSON(current, target.repositoryUpdate.Desired)
}

func applyFactoryConfigurationRepositories(tx *gorm.DB, target *factoryConfigurationTarget) error {
	if !configurationRepositoriesChanged(target) {
		return nil
	}
	previous := target.factory.OnboardingConfigValue()
	if previous.AppRepository != "" && (previous.AppRepository != target.repositoryUpdate.Desired.Primary.Repository || previous.DefaultBranch != target.repositoryUpdate.Desired.Primary.Ref) {
		branch, err := currentFactoryDefaultBranch(tx, target.factory, previous.DefaultBranch)
		if err != nil {
			return err
		}
		target.repositoryUpdate.previousBranch = branch
		if err := target.factory.SnapshotWorkOrderRepository(tx, previous.AppRepository, branch, previous.EffectiveVCSProvider()); err != nil {
			return err
		}
	}
	if err := target.factory.UpdateOnboarding(tx, target.repositoryUpdate.patch); err != nil {
		return err
	}
	desired := target.repositoryUpdate.Desired
	retained := map[string]factoryConfigurationContextRepository{}
	for _, entry := range desired.Additional {
		retained[configurationRepositoryKey(entry.Provider, entry.Repository)] = entry
	}
	existing := map[string]*models.FactoryAgentResource{}
	for i := range target.repositoryRecords {
		resource := &target.repositoryRecords[i]
		config := resource.Config.Data().NormalizedContextRepo()
		key := configurationRepositoryKey(config.VCSProvider, config.Repository)
		entry, keep := retained[key]
		if !keep {
			if err := resource.Delete(tx); err != nil {
				return err
			}
			continue
		}
		existing[key] = resource
		if resource.Enabled && !entry.enabled() {
			enabled := false
			if err := resource.Update(tx, nil, &enabled, nil); err != nil {
				return err
			}
		}
	}
	for _, entry := range desired.Additional {
		config := entry.config(target.repositoryUpdate.Integration.Provider).NormalizedContextRepo()
		enabled := entry.enabled()
		if resource := existing[configurationRepositoryKey(entry.Provider, entry.Repository)]; resource != nil {
			if resource.Enabled == enabled && equalCanvasJSON(resource.Config.Data(), config) {
				continue
			}
			if err := resource.Update(tx, &resource.Name, &enabled, &config); err != nil {
				return err
			}
			continue
		}
		if _, err := target.factory.CreateAgentResource(tx, models.FactoryAgentResourceKindContextRepo, "", enabled, config); err != nil {
			return err
		}
	}
	// Sync retains existing grants for saved additional repositories.
	for _, repositoryID := range target.repositoryUpdate.GitHubRepositoryIDs {
		if err := models.GrantVCSProviderBindingRepository(tx, uuid.MustParse(desired.Binding.IntegrationID), models.ProviderGitHub, repositoryID); err != nil {
			return err
		}
	}
	return syncFactoryVCSProviderBindings(tx, target.repositoryUpdate.Integration.Provider, previous.VCSIntegrationID, uuid.MustParse(desired.Binding.IntegrationID))
}

func reconcileFactoryConfigurationRepositories(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, previous models.FactoryOnboardingConfig, actor uuid.UUID) ([]func(), error) {
	options := intakePublisherOptions(deps, target.factory.OrganizationID)
	if err := options.Validate(); err != nil {
		return nil, err
	}
	branch := previous.DefaultBranch
	if target.repositoryUpdate.previousBranch != "" {
		branch = target.repositoryUpdate.previousBranch
	}
	if err := reconcileFactoryRepository(ctx, tx, deps, target.factory, actor, previous.VCSIntegrationID, target.factory.OnboardingConfigValue().VCSIntegrationID, previous.AppRepository, previous.BacklogRepository, branch, target.factory.OnboardingConfigValue().AppRepository, false); err != nil {
		return nil, err
	}
	if err := republishWorkspaceRepositoryAutomations(ctx, tx, deps, target.factory); err != nil {
		return nil, err
	}
	affected, err := target.factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	notifications := []func(){}
	for _, canvas := range affected {
		if err := validateConfigurationRepositorySetup(tx, target, &canvas); err != nil {
			return nil, err
		}
		notifications = append(notifications, func() { canvases.PublishCanvasUpdated(&canvas) })
	}
	return notifications, nil
}

func syncFactoryConfigurationRepositoryBindings(tx *gorm.DB, target *factoryConfigurationTarget, manifest *factoryConfigurationManifest, previous models.FactoryOnboardingConfig) error {
	ids := map[uuid.UUID]bool{}
	add := func(raw string) {
		if id, err := uuid.Parse(raw); err == nil {
			ids[id] = true
		}
	}
	add(previous.VCSIntegrationID)
	add(target.factory.OnboardingConfigValue().VCSIntegrationID)
	for _, owner := range target.ownership {
		id, _ := owner.Binding["integration_id"].(string)
		add(id)
	}
	for _, stage := range manifest.Stages {
		for _, automation := range stage.Automations {
			if automation.prepared != nil {
				add(automation.prepared.Binding.IntegrationID)
			}
		}
	}
	integrationIDs := make([]uuid.UUID, 0, len(ids))
	for id := range ids {
		integrationIDs = append(integrationIDs, id)
	}
	slices.SortFunc(integrationIDs, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	for _, id := range integrationIDs {
		binding, err := models.FindVCSProviderIntegrationBinding(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if binding.OrganizationID != target.factory.OrganizationID {
			continue
		}
		if err := models.SyncVCSProviderBindingRepositories(tx, id, binding.Provider); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigurationRepositorySetup(tx *gorm.DB, target *factoryConfigurationTarget, canvas *models.Canvas) error {
	live, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return err
	}
	previous := target.reads.versions[canvas.ID]
	for _, node := range live.Nodes {
		if node.ErrorMessage == nil {
			continue
		}
		if previous != nil {
			old := findIntakeNode(previous.Nodes, node.ID)
			if old != nil && old.ErrorMessage != nil && *old.ErrorMessage == *node.ErrorMessage && equalCanvasJSON(old.Configuration, node.Configuration) && equalCanvasJSON(old.Ref, node.Ref) && equalCanvasJSON(old.IntegrationID, node.IntegrationID) {
				continue
			}
		}
		return grpcerrors.FailedPrecondition(nil, fmt.Sprintf("Cannot set up %q, node %q: %s", canvas.Name, node.Name, *node.ErrorMessage))
	}
	ids := withWorkspaceListenerNodeIDs(nil, target.factory.OnboardingConfigValue().EffectiveVCSProvider(), live.Nodes)
	nodes, err := models.FindCanvasNodesByIDs(tx, canvas.ID, ids)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node.State == models.CanvasNodeStateError {
			reason := "Repository listener setup failed."
			if node.StateReason != nil {
				reason = *node.StateReason
			}
			return grpcerrors.FailedPrecondition(nil, fmt.Sprintf("Cannot set up %q, node %q: %s", canvas.Name, node.Name, reason))
		}
	}
	return nil
}

func configurationRepositoryAccessDiagnostic(err error, field string) (*pb.FactoryConfigurationDiagnostic, error) {
	var setting *factoryConfigurationSettingError
	if errors.As(err, &setting) {
		return configurationDiagnostic(field, setting.Message), nil
	}
	code, message, ok := grpcerrors.HandlerStatus(factoryErrorToStatus(err, "Failed to look up repository."))
	if ok && (code == codes.PermissionDenied || code == codes.InvalidArgument) {
		return configurationDiagnostic(field, message), nil
	}
	return nil, err
}
