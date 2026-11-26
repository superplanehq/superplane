package factories

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gopkg.in/yaml.v3"
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
			return configurationDiagnostic("repositories.additional", "Use the repository list from the workspace export.")
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

func (r factoryConfigurationContextRepository) enabled() bool { return r.Enabled == nil || *r.Enabled }

func configurationRepositoryKey(provider, repository string) string {
	if provider == "" {
		provider = models.ProviderGitHub
	}
	return provider + ":" + models.CanonicalContextRepo(repository)
}

func normalizeConfigurationRepositories(repositories factoryConfigurationRepositories, provider string) factoryConfigurationRepositories {
	repositories.Primary.Repository = models.CanonicalContextRepo(repositories.Primary.Repository)
	repositories.Primary.Ref = strings.TrimSpace(repositories.Primary.Ref)
	repositories.BacklogRepository = models.CanonicalContextRepo(repositories.BacklogRepository)
	if repositories.BacklogRepository == "" {
		repositories.BacklogRepository = repositories.Primary.Repository
	}
	repositories.Additional = slices.Clone(repositories.Additional)
	for i := range repositories.Additional {
		entry := &repositories.Additional[i]
		entry.Repository = models.CanonicalContextRepo(entry.Repository)
		entry.Ref = strings.TrimSpace(entry.Ref)
		entry.Provider = strings.ToLower(strings.TrimSpace(entry.Provider))
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
	slices.SortFunc(repositories.Additional, func(a, b factoryConfigurationContextRepository) int {
		return strings.Compare(configurationRepositoryKey(a.Provider, a.Repository), configurationRepositoryKey(b.Provider, b.Repository))
	})
	repositories.Binding = *portableConfigurationBinding(target.reads.integrations, &repositories.Binding)
	target.repositories = repositories
	return nil
}

// Repository settings are informational. Resolve connection names locally and
// compare their meaning before preparing or publishing any automation.
func validateFactoryConfigurationRepositories(tx *gorm.DB, target *factoryConfigurationTarget, requested *factoryConfigurationRepositories) (*pb.FactoryConfigurationDiagnostic, error) {
	if requested == nil {
		return nil, nil
	}
	reject := func() (*pb.FactoryConfigurationDiagnostic, error) {
		return configurationDiagnostic("repositories", "Repository settings cannot change through YAML. Change repositories in workspace settings and export again."), nil
	}
	if target.repositories == nil {
		return reject()
	}
	resolved, err := resolveConfigurationBindingName(tx, target.factory.OrganizationID, &requested.Binding)
	if err != nil {
		var setting *factoryConfigurationSettingError
		if errors.As(err, &setting) {
			return reject()
		}
		return nil, err
	}
	desired := *requested
	desired.Binding = *resolved
	current := *target.repositories
	current.Binding = factoryConfigurationBinding{IntegrationID: target.factory.OnboardingConfigValue().VCSIntegrationID}
	provider := target.factory.OnboardingConfigValue().EffectiveVCSProvider()
	desired = normalizeConfigurationRepositories(desired, provider)
	current = normalizeConfigurationRepositories(current, provider)
	if !equalCanvasJSON(current, desired) {
		return reject()
	}
	// Equivalent list order, defaults, and connection references yield one token.
	*requested = desired
	return nil, nil
}
