package factories

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

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
	gitHubBranches   func(repository string) ([]string, error)
	gitHubDefault    func(repository string) (string, error)
	bitbucketHead    func(repository, branch string) (string, error)
	bitbucketDefault func(repository string) (string, error)
}

type repositoryBranchResolverKey struct{}

func withRepositoryBranchResolver(ctx context.Context, resolver repositoryBranchResolver) context.Context {
	return context.WithValue(ctx, repositoryBranchResolverKey{}, resolver)
}

func repositoryBranchResolverFrom(ctx context.Context) repositoryBranchResolver {
	resolver, _ := ctx.Value(repositoryBranchResolverKey{}).(repositoryBranchResolver)
	return resolver
}

// chooseVerifiedDefaultBranch decides which branch to save.
// keepSaved means the caller must leave the stored branch unchanged.
func chooseVerifiedDefaultBranch(requested, saved string, exists bool, verifyErr error, live string, liveErr error) (string, bool, error) {
	requested = strings.TrimSpace(requested)
	saved = strings.TrimSpace(saved)
	live = strings.TrimSpace(live)
	if requested == "" {
		return "", true, nil
	}
	if verifyErr != nil {
		if saved != "" {
			return "", true, nil
		}
		return "", false, invalidArgument("could not verify the default branch")
	}
	if exists {
		return requested, false, nil
	}
	if liveErr != nil {
		if saved != "" {
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

type coreHTTPClient struct {
	client *http.Client
}

func (c coreHTTPClient) Do(req *http.Request) (*http.Response, error) {
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func providerHTTP() coreHTTPClient {
	return coreHTTPClient{client: &http.Client{Timeout: 15 * time.Second}}
}

// defaultBranchFromCacheOrProvider keeps a cached branch. An empty cache
// uses the provider value, or stays empty when that call fails.
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

func defaultBranchFromCacheOrProvider(cached string, live func() (string, error)) string {
	if branch := strings.TrimSpace(cached); branch != "" {
		return branch
	}
	if live == nil {
		return ""
	}
	branch, err := live()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(branch)
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
	if patch.DefaultBranch == nil || factory == nil {
		return patch, nil
	}
	requested := strings.TrimSpace(*patch.DefaultBranch)
	if requested == "" {
		patch.DefaultBranch = nil
		return patch, nil
	}

	merged, err := factory.OnboardingConfigAfter(patch)
	if err != nil {
		return patch, err
	}
	saved := strings.TrimSpace(factory.OnboardingConfigValue().DefaultBranch)
	exists, verifyErr, live, liveErr := onboardingBranchVerification(ctx, deps, db, orgID, organizationID, merged, requested)
	chosen, keepSaved, err := chooseVerifiedDefaultBranch(requested, saved, exists, verifyErr, live, liveErr)
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

func onboardingBranchVerification(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	organizationID string,
	config models.FactoryOnboardingConfig,
	requested string,
) (bool, error, string, error) {
	repository := strings.TrimSpace(config.AppRepository)
	if repository == "" {
		return false, invalidArgument("could not verify the default branch"), "", nil
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
) (bool, error, string, error) {
	integration, err := readyOnboardingVCSIntegration(db, orgID, config.VCSIntegrationID)
	if err != nil {
		return false, err, "", nil
	}
	installationID, repositoryID := githubInstallationForRepository(ctx, db, organizationID, integration, repository)
	names, err := gitHubBranchNames(ctx, deps, integration, repository, installationID, repositoryID)
	if err != nil {
		return false, err, "", nil
	}
	if slices.Contains(names, requested) {
		return true, nil, "", nil
	}
	live, liveErr := gitHubRepositoryDefaultBranch(ctx, deps, integration, repository, installationID, repositoryID)
	return false, nil, live, liveErr
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
) (bool, error, string, error) {
	integration, err := readyOnboardingVCSIntegration(db, orgID, config.VCSIntegrationID)
	if err != nil {
		return false, err, "", nil
	}
	exists, verifyErr := bitbucketBranchExists(ctx, db, deps, integration, repository, requested)
	if verifyErr != nil || exists {
		return exists, verifyErr, "", nil
	}
	live, liveErr := bitbucketRepositoryDefaultBranch(ctx, db, deps, integration, repository)
	return false, nil, live, liveErr
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
	branch, err := client.GetMainBranch(repository)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(branch), nil
}

func readyOnboardingVCSIntegration(db *gorm.DB, orgID uuid.UUID, integrationID string) (*models.Integration, error) {
	integrationID = strings.TrimSpace(integrationID)
	if integrationID == "" {
		return nil, nil
	}
	return findReadyOnboardingIntegration(db, orgID, integrationID)
}
