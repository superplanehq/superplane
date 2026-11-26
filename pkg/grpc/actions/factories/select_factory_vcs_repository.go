package factories

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func SelectFactoryVCSProviderRepository(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.SelectFactoryVCSProviderRepositoryRequest,
) (*pb.SelectFactoryVCSProviderRepositoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select VCS repository")
	}
	provider := strings.ToLower(strings.TrimSpace(req.GetProvider()))
	if provider == models.ProviderBitbucket {
		return selectBitbucketFactoryRepository(ctx, deps, orgID, organizationID, req)
	}
	if provider != models.ProviderGitHub {
		return nil, grpcerrors.InvalidArgument(nil, "VCS provider is not supported")
	}
	if req.GetRepositoryId() <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "repository id is required")
	}
	app, err := githubapp.ResolveProcess(ctx)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load public GitHub App")
	}
	if !app.Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}

	db := database.DB(ctx)
	catalog, findErr := findGitHubCatalogRepository(ctx, db, organizationID, req.GetRepositoryId())
	if findErr != nil {
		return nil, factoryErrorToStatus(findErr, "failed to select VCS repository")
	}
	var httpCtx core.HTTPContext
	if deps.Registry != nil {
		httpCtx = deps.Registry.HTTPContext()
	}
	liveBranch := defaultBranchFromCacheOrProvider(catalog.DefaultBranch, func() (string, error) {
		return common.InstallationRepositoryDefaultBranch(
			httpCtx,
			app.ID,
			app.PrivateKey,
			catalog.InstallationID,
			catalog.RepositoryID,
			catalog.FullName,
		)
	})

	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := models.LockOrganizationVCSProviderBindingRepositories(tx, orgID); err != nil {
			return err
		}
		factory, err = findFactory(tx, orgID, req.GetId())
		if err != nil {
			return err
		}
		if err := factory.LockForUpdate(tx); err != nil {
			return err
		}
		if err := rejectFactoryProviderSwitch(factory, provider); err != nil {
			return err
		}

		repository, findErr := findGitHubCatalogRepository(ctx, tx, organizationID, req.GetRepositoryId())
		if findErr != nil {
			return findErr
		}

		integration, bindErr := gitHubRepositoryIntegration(tx, orgID, repository)
		if bindErr != nil {
			return bindErr
		}

		previousIntegrationID := factory.OnboardingConfigValue().VCSIntegrationID
		integrationID := integration.ID.String()
		repositoryID := repository.RepositoryID
		backlogRepositoryID := &repositoryID
		if config := factory.OnboardingConfigValue(); config.BacklogRepository != "" && config.BacklogRepository != config.AppRepository {
			backlogRepositoryID = nil
		}
		branch := keepSavedDefaultBranch(factory.OnboardingConfigValue(), repository.FullName, liveBranch)
		patch := models.FactoryOnboardingPatch{
			VCSIntegrationID:    &integrationID,
			VCSProvider:         &provider,
			AppRepository:       &repository.FullName,
			AppRepositoryID:     &repositoryID,
			BacklogRepository:   onboardingBacklogRepository(factory, repository.FullName),
			BacklogRepositoryID: backlogRepositoryID,
			DefaultBranch:       &branch,
		}
		patch, err = authorizeFactoryRepositoryPatch(ctx, tx, deps, factory, patch)
		if err != nil {
			return err
		}
		if err := factory.UpdateOnboarding(tx, patch); err != nil {
			return err
		}
		return syncFactoryVCSProviderBindings(tx, provider, previousIntegrationID, integration.ID)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select VCS repository")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select VCS repository")
	}
	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select VCS repository")
	}
	return &pb.SelectFactoryVCSProviderRepositoryResponse{Factory: serialized}, nil
}

const organizationKeyRepositoryDenied = "This repository is not authorized for the selected connection. Use a personal API token with GitHub access to select it first."

func factoryRepositoryCaller(ctx context.Context, tx *gorm.DB, organizationID string) (*models.User, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := models.FindActiveUserByIDInTransaction(tx, organizationID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.PermissionDenied(err, "VCS repository is not accessible")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load user")
	}
	return user, nil
}

func gitHubRepositoryAccessError(err error, message string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return grpcerrors.PermissionDenied(err, message)
	}
	return err
}

func findGitHubCatalogRepository(
	ctx context.Context,
	tx *gorm.DB,
	organizationID string,
	repositoryID int64,
) (*models.AccessibleVCSProviderRepository, error) {
	user, err := factoryRepositoryCaller(ctx, tx, organizationID)
	if err != nil {
		return nil, err
	}
	if user.IsAPIKey() {
		repository, err := models.FindGrantedVCSProviderRepository(tx, user.OrganizationID, uuid.Nil, models.ProviderGitHub, repositoryID)
		return repository, gitHubRepositoryAccessError(err, organizationKeyRepositoryDenied)
	}
	var repository *models.AccessibleVCSProviderRepository
	if !githubapp.AccountConnectionRequired(ctx) {
		repository, err = models.FindInstalledVCSProviderRepository(tx, models.ProviderGitHub, repositoryID)
	} else {
		var providerUserID int64
		providerUserID, err = factoryVCSProviderUserID(ctx, tx, organizationID, models.ProviderGitHub)
		if err != nil {
			return nil, err
		}
		repository, err = models.FindAccessibleVCSProviderRepository(tx, models.ProviderGitHub, providerUserID, repositoryID)
	}
	return repository, gitHubRepositoryAccessError(err, "VCS repository is not accessible")
}

func findGitHubCatalogRepositoryByName(
	ctx context.Context,
	tx *gorm.DB,
	organizationID string,
	fullName string,
	integrationID uuid.UUID,
) (*models.AccessibleVCSProviderRepository, error) {
	user, err := factoryRepositoryCaller(ctx, tx, organizationID)
	if err != nil {
		return nil, err
	}
	if user.IsAPIKey() {
		repository, err := models.FindGrantedVCSProviderRepositoryByName(tx, user.OrganizationID, integrationID, models.ProviderGitHub, fullName)
		return repository, gitHubRepositoryAccessError(err, organizationKeyRepositoryDenied)
	}
	var repository *models.AccessibleVCSProviderRepository
	if !githubapp.AccountConnectionRequired(ctx) {
		repository, err = models.FindInstalledVCSProviderRepositoryByName(tx, models.ProviderGitHub, fullName)
	} else {
		var providerUserID int64
		providerUserID, err = factoryVCSProviderUserID(ctx, tx, organizationID, models.ProviderGitHub)
		if err != nil {
			return nil, err
		}
		repository, err = models.FindAccessibleVCSProviderRepositoryByName(tx, models.ProviderGitHub, providerUserID, fullName)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.PermissionDenied(err, "VCS repository is not accessible")
	}
	if err != nil {
		return nil, err
	}
	if integrationID != uuid.Nil {
		binding, err := models.FindVCSProviderIntegrationBinding(tx, integrationID)
		if err != nil {
			return nil, err
		}
		if binding.OrganizationID != user.OrganizationID || binding.Provider != models.ProviderGitHub || binding.InstallationID != repository.InstallationID {
			return nil, grpcerrors.PermissionDenied(nil, "VCS repository is not accessible")
		}
	}
	return repository, nil
}

func factoryVCSProviderUserID(ctx context.Context, db *gorm.DB, organizationID, provider string) (int64, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return 0, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := models.FindActiveUserByIDInTransaction(db, organizationID, userID)
	if err != nil {
		return 0, factoryErrorToStatus(err, "failed to load user")
	}
	if user.AccountID == nil {
		return 0, grpcerrors.FailedPrecondition(nil, "connect a VCS provider account first")
	}
	linked, err := models.FindAccountLinkedAccount(db, *user.AccountID, provider)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, grpcerrors.FailedPrecondition(err, "connect a VCS provider account first")
	}
	if err != nil {
		return 0, grpcerrors.Internal(err, "failed to load linked VCS provider account")
	}
	numericID, err := strconv.ParseInt(linked.ProviderID, 10, 64)
	if err != nil || numericID <= 0 {
		return 0, grpcerrors.FailedPrecondition(err, "linked VCS provider account has an invalid user id")
	}
	return numericID, nil
}

func syncFactoryVCSProviderBindings(
	tx *gorm.DB,
	provider, previousIntegrationID string,
	currentIntegrationID uuid.UUID,
) error {
	integrationIDs := []uuid.UUID{currentIntegrationID}
	if strings.TrimSpace(previousIntegrationID) != "" && previousIntegrationID != currentIntegrationID.String() {
		previousID, err := uuid.Parse(previousIntegrationID)
		if err != nil {
			return err
		}
		integrationIDs = append(integrationIDs, previousID)
	}
	slices.SortFunc(integrationIDs, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	for _, integrationID := range integrationIDs {
		if _, err := models.FindVCSProviderIntegrationBinding(tx, integrationID); errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		} else if err != nil {
			return err
		}
		if err := models.SyncVCSProviderBindingRepositories(tx, integrationID, provider); err != nil {
			return err
		}
	}
	return nil
}

// rejectFactoryProviderSwitch locks the provider at the first saved VCS
// binding. Clearing and rebinding across hosts is rejected.
func rejectFactoryProviderSwitch(factory *models.Factory, provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return grpcerrors.InvalidArgument(nil, "VCS provider is not supported")
	}
	current := factory.OnboardingConfigValue()
	if strings.TrimSpace(current.VCSIntegrationID) == "" && strings.TrimSpace(current.VCSProvider) == "" {
		return nil
	}
	if !strings.EqualFold(provider, current.EffectiveVCSProvider()) {
		return grpcerrors.InvalidArgument(nil, "version control provider cannot be changed")
	}
	return nil
}

// Keep an explicit backlog binding when the application repository changes.
func onboardingBacklogRepository(factory *models.Factory, repository string) *string {
	config := factory.OnboardingConfigValue()
	if config.BacklogRepository != "" && config.BacklogRepository != config.AppRepository {
		return nil
	}
	return &repository
}

// An organization-key lookup supplies the granted connection; only humans can create one.
func gitHubRepositoryIntegration(tx *gorm.DB, orgID uuid.UUID, repository *models.AccessibleVCSProviderRepository) (*models.Integration, error) {
	if repository.IntegrationID != uuid.Nil {
		return models.FindIntegrationInTransaction(tx, orgID, repository.IntegrationID)
	}
	installation, err := models.FindVCSProviderInstallation(tx, models.ProviderGitHub, repository.InstallationID)
	if err != nil {
		return nil, err
	}
	return models.FindOrCreateVCSProviderBinding(tx, orgID, models.ProviderGitHub, repository.InstallationID, installation.AccountLogin)
}
