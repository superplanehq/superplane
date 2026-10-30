package factories

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	githubintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

type repositoryBranchResolver struct {
	gitHubBranches       func(repository string) ([]string, error)
	gitHubDefault        func(repository string) (string, error)
	bitbucketHead        func(repository, branch string) (string, error)
	bitbucketDefault     func(repository string) (string, error)
	bitbucketHasBranches func(repository string) (bool, error)
}

type repositoryBranchResolverKey struct{}

// onboardingBranchCheck is the provider result used to choose a saved branch.
// A failed lookup keeps a saved branch only when the request names that
// same branch. A different requested name is rejected.
type onboardingBranchCheck struct {
	exists          bool
	live            string
	repositoryEmpty bool
	verifyErr       error
	liveErr         error
}

func withRepositoryBranchResolver(ctx context.Context, resolver repositoryBranchResolver) context.Context {
	return context.WithValue(ctx, repositoryBranchResolverKey{}, resolver)
}

func repositoryBranchResolverFrom(ctx context.Context) repositoryBranchResolver {
	resolver, _ := ctx.Value(repositoryBranchResolverKey{}).(repositoryBranchResolver)
	return resolver
}

// chooseVerifiedDefaultBranch decides which branch to save.
// keepSaved means the caller must leave the stored branch unchanged.
// repositoryEmpty is true only when the provider listed no branches.
// An empty repository accepts the requested name so checkout can create it.
// A repository that has other branches still rejects a missing name.
// A failed check keeps the saved branch only when the request names it.
// A different requested name returns an error so the caller can retry.
func chooseVerifiedDefaultBranch(requested, saved string, check onboardingBranchCheck) (string, bool, error) {
	requested = strings.TrimSpace(requested)
	saved = strings.TrimSpace(saved)
	live := strings.TrimSpace(check.live)
	if requested == "" {
		return "", true, nil
	}
	if check.verifyErr != nil {
		if saved != "" && requested == saved {
			return "", true, nil
		}
		return "", false, invalidArgument("could not verify the default branch")
	}
	if check.exists || check.repositoryEmpty {
		return requested, false, nil
	}
	if check.liveErr != nil {
		if saved != "" && requested == saved {
			return "", true, nil
		}
		return "", false, invalidArgument("could not verify the default branch")
	}
	if live != "" {
		return live, false, nil
	}
	if saved != "" {
		return "", true, nil
	}
	return "", false, invalidArgument("default branch is required")
}

func chooseSavedBranch(repositoryChanged bool, requested, hostDefault string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested, nil
	}
	if repositoryChanged {
		if hostDefault = strings.TrimSpace(hostDefault); hostDefault != "" {
			return hostDefault, nil
		}
	}
	return "", invalidArgument("default branch is required")
}

func sameRepository(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func ensureGitHubBranch(
	ctx context.Context,
	deps IntakeDependencies,
	integration *models.Integration,
	repository string,
	installationID, repositoryID int64,
	branch string,
) error {
	names, err := gitHubBranchNames(ctx, deps, integration, repository, installationID, repositoryID)
	if err != nil {
		return invalidArgument("could not verify the branch")
	}
	if slices.Contains(names, branch) {
		return nil
	}
	return invalidArgument("branch does not exist")
}

func gitHubBranchNames(
	ctx context.Context,
	deps IntakeDependencies,
	integration *models.Integration,
	repository string,
	installationID, repositoryID int64,
) ([]string, error) {
	if resolver := repositoryBranchResolverFrom(ctx).gitHubBranches; resolver != nil {
		return resolver(repository)
	}
	if installationID > 0 && repositoryID > 0 {
		return hostedGitHubBranchNames(ctx, deps, installationID, repositoryID, repository)
	}
	return integrationGitHubBranchNames(ctx, deps, integration, repository)
}

func hostedGitHubBranchNames(
	ctx context.Context,
	deps IntakeDependencies,
	installationID, repositoryID int64,
	repository string,
) ([]string, error) {
	if deps.Registry == nil {
		return nil, invalidArgument("could not verify the branch")
	}
	app, ok := common.ResolveHostedApp(ctx)
	if !ok {
		return nil, invalidArgument("could not verify the branch")
	}
	return common.ListInstallationBranchNames(
		deps.Registry.HTTPContext(),
		app.ID,
		app.PrivateKey,
		installationID,
		repositoryID,
		repository,
	)
}

func integrationGitHubBranchNames(
	ctx context.Context,
	deps IntakeDependencies,
	integration *models.Integration,
	repository string,
) ([]string, error) {
	if deps.Registry == nil || deps.Encryptor == nil || integration == nil {
		return nil, invalidArgument("could not verify the branch")
	}
	integrationImpl, err := deps.Registry.GetIntegration((&githubintegration.GitHub{}).Name())
	if err != nil {
		return nil, err
	}
	resources, err := integrationImpl.ListResources("branch", core.ListResourcesContext{
		Logger: logrus.NewEntry(logrus.StandardLogger()),
		HTTP:   deps.Registry.HTTPContext(),
		Integration: contexts.NewIntegrationContext(
			database.DB(ctx),
			nil,
			integration,
			deps.Encryptor,
			deps.Registry,
			nil,
		),
		Parameters: map[string]string{"repository": repository},
	})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(resources))
	for _, resource := range resources {
		if resource.Type != "branch" {
			continue
		}
		name := strings.TrimSpace(resource.Name)
		if name == "" {
			name = strings.TrimSpace(resource.ID)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func ensureBitbucketBranch(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	integration *models.Integration,
	repository, branch string,
) error {
	head, err := bitbucketBranchHead(ctx, db, deps, integration, repository, branch)
	if err != nil {
		return err
	}
	if strings.TrimSpace(head) == "" {
		return invalidArgument("branch does not exist")
	}
	return nil
}

func bitbucketBranchHead(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	integration *models.Integration,
	repository, branch string,
) (string, error) {
	if resolver := repositoryBranchResolverFrom(ctx).bitbucketHead; resolver != nil {
		head, err := resolver(repository, branch)
		if err != nil {
			return "", invalidArgument("could not verify the branch")
		}
		if strings.TrimSpace(head) == "" {
			return "", invalidArgument("branch does not exist")
		}
		return head, nil
	}
	client, err := openBitbucketClient(db, deps, integration)
	if err != nil {
		return "", invalidArgument("could not verify the branch")
	}
	head, err := client.GetBranchHead(repository, branch)
	if err != nil {
		return "", invalidArgument("branch does not exist")
	}
	return head, nil
}

func openBitbucketClient(db *gorm.DB, deps IntakeDependencies, integration *models.Integration) (*bitbucketintegration.Client, error) {
	if deps.Registry == nil || deps.Encryptor == nil || integration == nil || integration.AppName != models.ProviderBitbucket {
		return nil, errFactoryBitbucketNotConnected
	}
	return buildBitbucketClient(db, deps, integration)
}

// keepSavedDefaultBranch leaves a branch the user already saved when the
// provider does not report one for the same repository.
func keepSavedDefaultBranch(current models.FactoryOnboardingConfig, repository, resolved string) string {
	if strings.TrimSpace(resolved) != "" {
		return strings.TrimSpace(resolved)
	}
	if !sameRepository(current.AppRepository, repository) {
		return ""
	}
	return strings.TrimSpace(current.DefaultBranch)
}

// defaultBranchFromCacheOrProvider prefers the live provider branch.
// The cached value is used only when the live call is missing, fails,
// or returns a blank name.
func defaultBranchFromCacheOrProvider(cached string, live func() (string, error)) string {
	cached = strings.TrimSpace(cached)
	if live != nil {
		branch, err := live()
		if err == nil {
			if branch = strings.TrimSpace(branch); branch != "" {
				return branch
			}
		}
	}
	return cached
}

func verifiedOnboardingDefaultBranch(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	organizationID string,
	factory *models.Factory,
	patch models.FactoryOnboardingPatch,
) (models.FactoryOnboardingPatch, error) {
	if factory == nil {
		return patch, nil
	}
	current := factory.OnboardingConfigValue()
	repositoryChanged := patch.AppRepository != nil && !sameRepository(current.AppRepository, *patch.AppRepository)
	if patch.DefaultBranch == nil || strings.TrimSpace(*patch.DefaultBranch) == "" {
		// A new repository must not keep the previous repository's branch.
		// Use the provider default when it is available. Leave the branch
		// empty when that lookup fails so the user can choose one.
		if repositoryChanged {
			branch := onboardingProviderDefaultBranch(ctx, deps, db, orgID, organizationID, factory, patch)
			if branch == "" {
				empty := ""
				patch.DefaultBranch = &empty
				return patch, nil
			}
			patch.DefaultBranch = &branch
			return patch, nil
		}
		patch.DefaultBranch = nil
		return patch, nil
	}
	requested := strings.TrimSpace(*patch.DefaultBranch)

	merged, err := factory.OnboardingConfigAfter(patch)
	if err != nil {
		return patch, err
	}
	// A saved branch belongs to the current repository. A different
	// repository must not keep that name when the provider lookup fails.
	saved := ""
	if sameRepository(current.AppRepository, merged.AppRepository) {
		saved = strings.TrimSpace(current.DefaultBranch)
	}
	check := onboardingBranchVerification(ctx, deps, db, orgID, organizationID, merged, requested)
	chosen, keepSaved, err := chooseVerifiedDefaultBranch(requested, saved, check)
	if err != nil {
		return patch, err
	}
	if keepSaved {
		patch.DefaultBranch = nil
		return patch, nil
	}
	patch.DefaultBranch = &chosen
	return patch, nil
}

// onboardingProviderDefaultBranch reads the default branch the provider
// reports for the repository in the patch. A failed lookup returns "".
func onboardingProviderDefaultBranch(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	organizationID string,
	factory *models.Factory,
	patch models.FactoryOnboardingPatch,
) string {
	if factory == nil {
		return ""
	}
	merged, err := factory.OnboardingConfigAfter(patch)
	if err != nil {
		return ""
	}
	repository := strings.TrimSpace(merged.AppRepository)
	if repository == "" {
		return ""
	}
	integration, err := readyOnboardingVCSIntegration(db, orgID, merged.VCSIntegrationID)
	if err != nil {
		return ""
	}
	if merged.EffectiveVCSProvider() == models.ProviderBitbucket {
		branch, lookupErr := bitbucketRepositoryDefaultBranch(ctx, db, deps, integration, repository)
		if lookupErr != nil {
			return ""
		}
		return strings.TrimSpace(branch)
	}
	installationID, repositoryID := githubInstallationForRepository(ctx, db, organizationID, integration, repository)
	branch, lookupErr := gitHubRepositoryDefaultBranch(ctx, deps, integration, repository, installationID, repositoryID)
	if lookupErr != nil {
		return ""
	}
	return strings.TrimSpace(branch)
}

func onboardingBranchVerification(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	organizationID string,
	config models.FactoryOnboardingConfig,
	requested string,
) onboardingBranchCheck {
	repository := strings.TrimSpace(config.AppRepository)
	if repository == "" {
		return onboardingBranchCheck{verifyErr: invalidArgument("could not verify the default branch")}
	}
	if config.EffectiveVCSProvider() == models.ProviderBitbucket {
		return bitbucketOnboardingBranchVerification(ctx, deps, db, orgID, config, repository, requested)
	}
	return githubOnboardingBranchVerification(ctx, deps, db, orgID, organizationID, config, repository, requested)
}

func githubOnboardingBranchVerification(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	organizationID string,
	config models.FactoryOnboardingConfig,
	repository, requested string,
) onboardingBranchCheck {
	integration, err := readyOnboardingVCSIntegration(db, orgID, config.VCSIntegrationID)
	if err != nil {
		return onboardingBranchCheck{verifyErr: err}
	}
	installationID, repositoryID := githubInstallationForRepository(ctx, db, organizationID, integration, repository)
	names, err := gitHubBranchNames(ctx, deps, integration, repository, installationID, repositoryID)
	if err != nil {
		return onboardingBranchCheck{verifyErr: err}
	}
	if slices.Contains(names, requested) {
		return onboardingBranchCheck{exists: true}
	}
	if len(names) == 0 {
		return onboardingBranchCheck{repositoryEmpty: true}
	}
	live, liveErr := gitHubRepositoryDefaultBranch(ctx, deps, integration, repository, installationID, repositoryID)
	return onboardingBranchCheck{live: live, liveErr: liveErr}
}

func gitHubRepositoryDefaultBranch(
	ctx context.Context,
	deps IntakeDependencies,
	integration *models.Integration,
	repository string,
	installationID, repositoryID int64,
) (string, error) {
	if resolver := repositoryBranchResolverFrom(ctx).gitHubDefault; resolver != nil {
		branch, err := resolver(repository)
		return strings.TrimSpace(branch), err
	}
	if installationID > 0 && repositoryID > 0 {
		if deps.Registry == nil {
			return "", invalidArgument("could not verify the default branch")
		}
		app, ok := common.ResolveHostedApp(ctx)
		if !ok {
			return "", invalidArgument("could not verify the default branch")
		}
		return common.InstallationRepositoryDefaultBranch(
			deps.Registry.HTTPContext(),
			app.ID,
			app.PrivateKey,
			installationID,
			repositoryID,
			repository,
		)
	}
	if deps.Registry == nil || integration == nil || integration.AppName != models.ProviderGitHub {
		return "", invalidArgument("could not verify the default branch")
	}
	client, err := common.NewClient(
		contexts.NewIntegrationContext(database.DB(ctx), nil, integration, deps.Encryptor, deps.Registry, nil),
		deps.Registry.HTTPContext(),
	)
	if err != nil {
		return "", err
	}
	repo, err := client.FindRepository(repository)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(repo.GetDefaultBranch()), nil
}

func githubInstallationForRepository(
	ctx context.Context,
	db *gorm.DB,
	organizationID string,
	integration *models.Integration,
	repository string,
) (int64, int64) {
	if integration == nil {
		return 0, 0
	}
	if _, err := models.FindVCSProviderIntegrationBinding(db, integration.ID); err != nil {
		return 0, 0
	}
	catalog, err := findGitHubCatalogRepositoryByName(ctx, db, organizationID, repository)
	if err != nil || catalog == nil {
		return 0, 0
	}
	return catalog.InstallationID, catalog.RepositoryID
}

func bitbucketOnboardingBranchVerification(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	config models.FactoryOnboardingConfig,
	repository, requested string,
) onboardingBranchCheck {
	integration, err := readyOnboardingVCSIntegration(db, orgID, config.VCSIntegrationID)
	if err != nil {
		return onboardingBranchCheck{verifyErr: err}
	}
	exists, verifyErr := bitbucketBranchExists(ctx, db, deps, integration, repository, requested)
	if verifyErr != nil || exists {
		return onboardingBranchCheck{exists: exists, verifyErr: verifyErr}
	}
	live, liveErr := bitbucketRepositoryDefaultBranch(ctx, db, deps, integration, repository)
	if liveErr != nil || strings.TrimSpace(live) != "" {
		return onboardingBranchCheck{live: live, liveErr: liveErr}
	}
	hasBranches, hasErr := bitbucketRepositoryHasBranches(ctx, db, deps, integration, repository)
	if hasErr != nil {
		return onboardingBranchCheck{verifyErr: hasErr}
	}
	return onboardingBranchCheck{repositoryEmpty: !hasBranches}
}

func bitbucketBranchExists(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	integration *models.Integration,
	repository, branch string,
) (bool, error) {
	if resolver := repositoryBranchResolverFrom(ctx).bitbucketHead; resolver != nil {
		head, err := resolver(repository, branch)
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(head) != "", nil
	}
	client, err := openBitbucketClient(db, deps, integration)
	if err != nil {
		return false, err
	}
	head, err := client.GetBranchHead(repository, branch)
	if err != nil {
		var apiErr *bitbucketintegration.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(head) != "", nil
}

func bitbucketRepositoryDefaultBranch(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	integration *models.Integration,
	repository string,
) (string, error) {
	if resolver := repositoryBranchResolverFrom(ctx).bitbucketDefault; resolver != nil {
		branch, err := resolver(repository)
		return strings.TrimSpace(branch), err
	}
	client, err := openBitbucketClient(db, deps, integration)
	if err != nil {
		return "", err
	}
	return bitbucketClientDefaultBranch(client, repository)
}

func bitbucketClientDefaultBranch(client *bitbucketintegration.Client, repository string) (string, error) {
	if client == nil {
		return "", errors.New("bitbucket client is required")
	}
	branch, err := client.GetMainBranch(repository)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(branch), nil
}

func bitbucketRepositoryHasBranches(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	integration *models.Integration,
	repository string,
) (bool, error) {
	if resolver := repositoryBranchResolverFrom(ctx).bitbucketHasBranches; resolver != nil {
		return resolver(repository)
	}
	client, err := openBitbucketClient(db, deps, integration)
	if err != nil {
		return false, err
	}
	return client.RepositoryHasBranches(repository)
}

func readyOnboardingVCSIntegration(db *gorm.DB, orgID uuid.UUID, integrationID string) (*models.Integration, error) {
	integrationID = strings.TrimSpace(integrationID)
	if integrationID == "" {
		return nil, nil
	}
	return findReadyOnboardingIntegration(db, orgID, integrationID)
}
