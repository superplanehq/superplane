package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type hostedGitHubIdentity struct {
	ID                          int64
	Login                       string
	AllowUnverifiedRepositories bool
}

type repositoryPermissionLookup func(context.Context, string, string, string) (*gh.RepositoryPermissionLevel, error)

const (
	allowUnverifiedDevelopmentRepositoriesEnv = "SUPERPLANE_GITHUB_APP_ALLOW_UNVERIFIED_REPOSITORIES"
	hostedInstallationVerificationConcurrency = 8
	hostedInstallRequestFallbackPageSize      = 8
	maxHostedDiscoveryErrors                  = 3
)

func findStartedByGitHubIdentity(ctx context.Context, organizationID, userID string) (*hostedGitHubIdentity, error) {
	if organizationID == "" || userID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	db := database.DB(ctx)
	user, err := models.FindActiveUserByIDInTransaction(db, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if user.AccountID == nil {
		return nil, gorm.ErrRecordNotFound
	}

	identity, err := models.FindAccountGitHubIdentity(db, *user.AccountID)
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(identity.ProviderID, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid GitHub identity ID")
	}

	return &hostedGitHubIdentity{ID: id, Login: strings.TrimSpace(identity.Username)}, nil
}

func useDevelopmentGitHubDiscovery() bool {
	return os.Getenv("APP_ENV") == "development" &&
		os.Getenv(allowUnverifiedDevelopmentRepositoriesEnv) == "yes"
}

func hostedGitHubDiscoveryIdentity(
	ctx context.Context,
	organizationID, userID string,
) (*hostedGitHubIdentity, error) {
	if useDevelopmentGitHubDiscovery() {
		return &hostedGitHubIdentity{Login: "development", AllowUnverifiedRepositories: true}, nil
	}

	return findStartedByGitHubIdentity(ctx, organizationID, userID)
}

func hostedIdentityConnectURL(baseURL, returnPath string) string {
	if baseURL == "" {
		return ""
	}
	if returnPath == "" {
		returnPath = "/"
	}

	values := url.Values{}
	values.Set("intent", "connect")
	values.Set("redirect", returnPath)
	return strings.TrimRight(baseURL, "/") + "/auth/github?" + values.Encode()
}

func discoverAccessibleInstallations(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
) ([]common.PendingInstallation, error) {
	appClient, err := newAppJWTClient(integration, app.ID)
	if err != nil {
		return nil, fmt.Errorf("create GitHub App client: %w", err)
	}

	installations, err := listAppInstallations(ctx, appClient)
	if err != nil {
		return nil, fmt.Errorf("list GitHub App installations: %w", err)
	}
	return verifyAccessibleInstallations(ctx, integration, app, identity, installations, nil)
}

func discoverAccessibleRecentInstallations(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	since time.Time,
	page int,
) ([]common.PendingInstallation, []common.PendingInstallation, int, error) {
	appClient, err := newAppJWTClient(integration, app.ID)
	if err != nil {
		return nil, nil, -1, fmt.Errorf("create GitHub App client: %w", err)
	}

	installations, nextPage, err := listRecentAppInstallations(
		ctx,
		appClient,
		since,
		page,
		hostedInstallRequestFallbackPageSize,
	)
	if err != nil {
		return nil, nil, -1, fmt.Errorf("list recent GitHub App installations: %w", err)
	}
	accessible, retries, err := verifyAccessibleInstallationsWithFailures(
		ctx,
		integration,
		app,
		identity,
		installations,
		nil,
	)
	return accessible, retries, nextPage, err
}

func discoverAccessibleInstallationByID(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	installationID string,
	repositoryIDs []int64,
) ([]common.PendingInstallation, error) {
	id, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid GitHub App installation ID")
	}

	appClient, err := newAppJWTClient(integration, app.ID)
	if err != nil {
		return nil, fmt.Errorf("create GitHub App client: %w", err)
	}
	installation, err := getAppInstallation(ctx, appClient, id)
	if err != nil {
		return nil, fmt.Errorf("get GitHub App installation %s: %w", installationID, err)
	}
	pending, err := pendingInstallationFromGitHub(installation)
	if err != nil {
		return nil, err
	}

	return verifyAccessibleInstallations(
		ctx,
		integration,
		app,
		identity,
		[]common.PendingInstallation{pending},
		repositoryIDs,
	)
}

func discoverAccessibleInstallationsByAccount(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	accounts []string,
) ([]common.PendingInstallation, error) {
	appClient, err := newAppJWTClient(integration, app.ID)
	if err != nil {
		return nil, fmt.Errorf("create GitHub App client: %w", err)
	}

	installations := make([]common.PendingInstallation, 0, len(accounts))
	lookupErrors := make([]error, 0)
	seen := map[string]struct{}{}
	for _, account := range accounts {
		account = strings.TrimSpace(account)
		key := strings.ToLower(account)
		if account == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		installation, findErr := findAppOrganizationInstallation(ctx, appClient, account)
		if githubErrorIsNotFound(findErr) {
			installation, findErr = findAppUserInstallation(ctx, appClient, account)
		}
		if githubErrorIsNotFound(findErr) {
			continue
		}
		if findErr != nil {
			lookupErrors = append(lookupErrors, fmt.Errorf("find GitHub App installation for %s: %w", account, findErr))
			continue
		}
		pending, conversionErr := pendingInstallationFromGitHub(installation)
		if conversionErr != nil {
			lookupErrors = append(lookupErrors, conversionErr)
			continue
		}
		installations = append(installations, pending)
	}

	accessible, verificationErr := verifyAccessibleInstallations(ctx, integration, app, identity, installations, nil)
	if verificationErr != nil {
		lookupErrors = append(lookupErrors, verificationErr)
	}
	return accessible, summarizeHostedDiscoveryErrors(lookupErrors)
}

func pendingInstallationFromGitHub(installation *gh.Installation) (common.PendingInstallation, error) {
	if installation == nil || installation.GetID() <= 0 || installation.GetAccount() == nil {
		return common.PendingInstallation{}, fmt.Errorf("GitHub App installation is incomplete")
	}
	account := installation.GetAccount()
	if strings.TrimSpace(account.GetLogin()) == "" {
		return common.PendingInstallation{}, fmt.Errorf("GitHub App installation account is empty")
	}

	return common.PendingInstallation{
		ID:           strconv.FormatInt(installation.GetID(), 10),
		AccountLogin: account.GetLogin(),
		AccountType:  account.GetType(),
	}, nil
}

func githubErrorIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var responseError *gh.ErrorResponse
	return errors.As(err, &responseError) &&
		responseError.Response != nil &&
		responseError.Response.StatusCode == http.StatusNotFound
}

func verifyAccessibleInstallations(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	installations []common.PendingInstallation,
	repositoryIDs []int64,
) ([]common.PendingInstallation, error) {
	accessible, _, err := verifyAccessibleInstallationsWithFailures(
		ctx,
		integration,
		app,
		identity,
		installations,
		repositoryIDs,
	)
	return accessible, err
}

func verifyAccessibleInstallationsWithFailures(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	installations []common.PendingInstallation,
	repositoryIDs []int64,
) ([]common.PendingInstallation, []common.PendingInstallation, error) {
	if err := ctx.Err(); err != nil {
		return nil, slices.Clone(installations), err
	}
	if !identity.AllowUnverifiedRepositories {
		resolved, ok, err := resolveHostedGitHubIdentity(ctx, integration, app, identity, installations)
		if err != nil {
			return nil, slices.Clone(installations), err
		}
		if !ok {
			return nil, nil, nil
		}
		identity = resolved
	}

	results := make([]*common.PendingInstallation, len(installations))
	failures := make([]error, len(installations))
	completed := make([]bool, len(installations))
	var group errgroup.Group
	group.SetLimit(hostedInstallationVerificationConcurrency)
	for index, installation := range installations {
		if err := ctx.Err(); err != nil {
			break
		}
		index := index
		installation := installation
		group.Go(func() error {
			verified, allowed, err := verifyAccessibleInstallation(
				ctx,
				integration,
				app,
				identity,
				installation,
				repositoryIDs,
			)
			completed[index] = true
			if err != nil {
				failures[index] = err
				return nil
			}
			if allowed {
				results[index] = &verified
			}
			return nil
		})
	}
	_ = group.Wait()
	retries := make([]common.PendingInstallation, 0)
	for index, failure := range failures {
		if failure != nil || !completed[index] {
			retries = append(retries, installations[index])
		}
	}
	if err := ctx.Err(); err != nil {
		return compactVerifiedInstallations(results), retries, err
	}

	return compactVerifiedInstallations(results), retries, summarizeHostedDiscoveryErrors(failures)
}

func resolveHostedGitHubIdentity(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	installations []common.PendingInstallation,
) (hostedGitHubIdentity, bool, error) {
	failures := make([]error, 0)
	for _, installation := range installations {
		if err := ctx.Err(); err != nil {
			return identity, false, err
		}
		client, err := newInstallationClient(integration, app.ID, installation.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("create client for installation %s: %w", installation.ID, err))
			continue
		}
		user, err := resolveInstallationIdentity(ctx, client, identity.ID)
		if err != nil {
			if githubErrorIsNotFound(err) {
				continue
			}
			failures = append(failures, fmt.Errorf("resolve GitHub identity for installation %s: %w", installation.ID, err))
			continue
		}
		if user == nil || user.GetID() != identity.ID || strings.TrimSpace(user.GetLogin()) == "" {
			continue
		}

		identity.Login = user.GetLogin()
		return identity, true, nil
	}

	if err := summarizeHostedDiscoveryErrors(failures); err != nil {
		return identity, false, err
	}
	return identity, false, nil
}

func verifyAccessibleInstallation(
	ctx context.Context,
	integration core.IntegrationContext,
	app common.HostedApp,
	identity hostedGitHubIdentity,
	installation common.PendingInstallation,
	repositoryIDs []int64,
) (common.PendingInstallation, bool, error) {
	if err := ctx.Err(); err != nil {
		return installation, false, err
	}
	client, err := newInstallationClient(integration, app.ID, installation.ID)
	if err != nil {
		return installation, false, fmt.Errorf("create client for installation %s: %w", installation.ID, err)
	}

	repositories, err := listInstallationRepos(ctx, client)
	if err != nil {
		return installation, false, fmt.Errorf("list repositories for installation %s: %w", installation.ID, err)
	}
	repositories = repositoriesByID(repositories, repositoryIDs)
	if identity.AllowUnverifiedRepositories {
		installation.Repositories = qualifyRepositoryNames(installation.AccountLogin, repositories)
		return installation, len(installation.Repositories) > 0, nil
	}

	writable, err := filterWritableRepositories(
		ctx,
		installation.AccountLogin,
		identity.Login,
		repositories,
		func(ctx context.Context, owner, repository, username string) (*gh.RepositoryPermissionLevel, error) {
			return getRepositoryPermission(ctx, client, owner, repository, username)
		},
	)
	if err != nil {
		return installation, false, fmt.Errorf("check repository access for installation %s: %w", installation.ID, err)
	}
	installation.Repositories = writable
	return installation, len(writable) > 0, nil
}

func repositoriesByID(repositories []common.Repository, ids []int64) []common.Repository {
	if len(ids) == 0 {
		return repositories
	}

	selected := make([]common.Repository, 0, len(ids))
	for _, id := range ids {
		for _, repository := range repositories {
			if repository.ID == id {
				selected = append(selected, repository)
				break
			}
		}
	}
	return selected
}

func compactVerifiedInstallations(results []*common.PendingInstallation) []common.PendingInstallation {
	accessible := make([]common.PendingInstallation, 0, len(results))
	for _, result := range results {
		if result != nil {
			accessible = append(accessible, *result)
		}
	}
	return accessible
}

func summarizeHostedDiscoveryErrors(failures []error) error {
	summary := make([]error, 0, maxHostedDiscoveryErrors+1)
	total := 0
	for _, failure := range failures {
		if failure == nil {
			continue
		}
		total++
		if len(summary) < maxHostedDiscoveryErrors {
			summary = append(summary, failure)
		}
	}
	if total > maxHostedDiscoveryErrors {
		summary = append(summary, fmt.Errorf("%d additional installation checks failed", total-maxHostedDiscoveryErrors))
	}
	if len(summary) == 0 {
		return nil
	}

	return errors.Join(summary...)
}

func qualifyRepositoryNames(owner string, repositories []common.Repository) []common.Repository {
	qualified := make([]common.Repository, 0, len(repositories))
	for _, repository := range repositories {
		repository.Name = owner + "/" + repository.Name
		qualified = append(qualified, repository)
	}
	return qualified
}

func filterWritableRepositories(
	ctx context.Context,
	owner string,
	username string,
	repositories []common.Repository,
	lookup repositoryPermissionLookup,
) ([]common.Repository, error) {
	writable := make([]common.Repository, 0, len(repositories))
	for _, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		permission, err := lookup(ctx, owner, repository.Name, username)
		if err != nil {
			return nil, err
		}
		if permission == nil || !hasRepositoryWritePermission(permission.GetPermission()) {
			continue
		}

		repository.Name = owner + "/" + repository.Name
		writable = append(writable, repository)
	}
	return writable, nil
}

func hasRepositoryWritePermission(permission string) bool {
	return permission == "write" || permission == "admin"
}

func retainInstalledRepositories(granted, installed []common.Repository) []common.Repository {
	installedIDs := make(map[int64]struct{}, len(installed))
	for _, repository := range installed {
		installedIDs[repository.ID] = struct{}{}
	}

	remaining := make([]common.Repository, 0, len(granted))
	for _, repository := range granted {
		if _, ok := installedIDs[repository.ID]; ok {
			remaining = append(remaining, repository)
		}
	}
	return remaining
}
