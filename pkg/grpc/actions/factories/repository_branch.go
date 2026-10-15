package factories

import (
	"context"
	"slices"
	"strings"

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
	gitHubBranches func(repository string) ([]string, error)
	bitbucketHead  func(repository, branch string) (string, error)
}

type repositoryBranchResolverKey struct{}

func withRepositoryBranchResolver(ctx context.Context, resolver repositoryBranchResolver) context.Context {
	return context.WithValue(ctx, repositoryBranchResolverKey{}, resolver)
}

func repositoryBranchResolverFrom(ctx context.Context) repositoryBranchResolver {
	resolver, _ := ctx.Value(repositoryBranchResolverKey{}).(repositoryBranchResolver)
	return resolver
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
