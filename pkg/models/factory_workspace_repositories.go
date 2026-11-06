package models

import (
	"errors"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/gorm"
)

var ErrFactoryPullRequestRepositoryNotInWorkspace = errors.New("pull request repository is not in this workspace")

// WorkspaceRepositoriesExpression is the repository value of a node that
// listens on every repository the workspace may change.
const WorkspaceRepositoriesExpression = "{{ workspace().repositories }}"

var workspaceRepositoriesExpressionPattern = regexp.MustCompile(`^\{\{\s*workspace\(\)\.repositories\s*\}\}$`)

func IsWorkspaceRepositoriesExpression(value string) bool {
	return workspaceRepositoriesExpressionPattern.MatchString(strings.TrimSpace(value))
}

// MultiRepoTasksEnabled reports whether the organization of the workspace
// lets agents change context repositories.
func (f *Factory) MultiRepoTasksEnabled(tx *gorm.DB) (bool, error) {
	organization, err := FindOrganizationByIDInTransaction(tx, f.OrganizationID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// A deleted organization keeps the single-repository behavior.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return organization.HasExperimentalFeature(features.FeatureFactoryMultiRepoTasks), nil
}

// WorkspaceRepositories returns the owner/name values that the workspace may
// change. The primary repository comes first. Workspaces with multi-repo
// tasks add the enabled write-access context repositories of the workspace
// provider; other workspaces keep one repository.
func (f *Factory) WorkspaceRepositories(tx *gorm.DB) ([]string, error) {
	config := f.OnboardingConfigValue()
	repositories := []string{}
	seen := map[string]struct{}{}
	add := func(repository string) {
		repository = strings.TrimSpace(repository)
		canonical := CanonicalContextRepo(repository)
		if canonical == "" {
			return
		}
		if _, ok := seen[canonical]; ok {
			return
		}
		seen[canonical] = struct{}{}
		repositories = append(repositories, repository)
	}

	add(config.AppRepository)
	provider := config.EffectiveVCSProvider()
	if provider != ProviderGitHub && provider != ProviderBitbucket {
		return repositories, nil
	}
	multiRepo, err := f.MultiRepoTasksEnabled(tx)
	if err != nil || !multiRepo {
		return repositories, err
	}

	resources, err := f.ListEnabledContextRepos(tx)
	if err != nil {
		return nil, err
	}
	for i := range resources {
		resourceConfig := resources[i].Config.Data()
		if contextRepoProvider(resourceConfig) != provider || !resourceConfig.ContextRepoWritable() {
			continue
		}
		add(resourceConfig.Repository)
	}
	return repositories, nil
}

// IsWorkspaceRepository reports whether repository is one of the workspace
// repositories. The comparison ignores case.
func (f *Factory) IsWorkspaceRepository(tx *gorm.DB, repository string) (bool, error) {
	canonical := CanonicalContextRepo(repository)
	if canonical == "" {
		return false, nil
	}
	repositories, err := f.WorkspaceRepositories(tx)
	if err != nil {
		return false, err
	}
	for _, candidate := range repositories {
		if CanonicalContextRepo(candidate) == canonical {
			return true, nil
		}
	}
	return false, nil
}

// IsTaskRepository reports whether a task may have a pull request in
// repository. A task keeps its saved repository after the workspace primary
// repository changes, so that repository stays valid for the task.
func (f *Factory) IsTaskRepository(tx *gorm.DB, order *FactoryWorkOrder, repository string) (bool, error) {
	canonical := CanonicalContextRepo(repository)
	if canonical == "" {
		return false, nil
	}
	if order != nil && order.Repository != nil && CanonicalContextRepo(*order.Repository) == canonical {
		return true, nil
	}
	return f.IsWorkspaceRepository(tx, repository)
}

// RequireTaskPullRequest rejects a pull request outside the repositories of
// the task. A workspace without a repository accepts every pull request.
func (f *Factory) RequireTaskPullRequest(tx *gorm.DB, order *FactoryWorkOrder, params FactoryPullRequestParams) error {
	normalized, err := normalizeFactoryPullRequestParams(params)
	if err != nil {
		return err
	}
	repositories, err := f.WorkspaceRepositories(tx)
	if err != nil {
		return err
	}
	if len(repositories) == 0 {
		return nil
	}
	allowed, err := f.IsTaskRepository(tx, order, normalized.Repository)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrFactoryPullRequestRepositoryNotInWorkspace
	}
	return nil
}

// contextRepoProvider returns the host of a context repository. Context
// repositories saved before the provider field are GitHub repositories.
func contextRepoProvider(config FactoryAgentResourceConfig) string {
	provider := strings.ToLower(strings.TrimSpace(config.VCSProvider))
	if provider == "" {
		return ProviderGitHub
	}
	return provider
}
