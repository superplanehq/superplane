package factories

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func SelectFactoryVCSProviderRepository(
	ctx context.Context,
	organizationID string,
	req *pb.SelectFactoryVCSProviderRepositoryRequest,
) (*pb.SelectFactoryVCSProviderRepositoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select VCS repository")
	}
	provider := strings.ToLower(strings.TrimSpace(req.GetProvider()))
	if provider == models.ProviderBitbucket {
		return selectBitbucketFactoryRepository(ctx, orgID, organizationID, req)
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
	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err = findFactory(tx, orgID, req.GetId())
		if err != nil {
			return err
		}
		if err := rejectFactoryProviderSwitch(factory, provider); err != nil {
			return err
		}

		repository, findErr := findGitHubCatalogRepository(ctx, tx, organizationID, req.GetRepositoryId())
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return grpcerrors.PermissionDenied(findErr, "VCS repository is not accessible")
		}
		if findErr != nil {
			return findErr
		}

		installation, findErr := models.FindVCSProviderInstallation(tx, provider, repository.InstallationID)
		if findErr != nil {
			return findErr
		}
		integration, bindErr := models.FindOrCreateVCSProviderBinding(
			tx,
			orgID,
			provider,
			repository.InstallationID,
			installation.AccountLogin,
		)
		if bindErr != nil {
			return bindErr
		}

		previousIntegrationID := factory.OnboardingConfigValue().VCSIntegrationID
		integrationID := integration.ID.String()
		repositoryID := repository.RepositoryID
		if updateErr := factory.UpdateOnboarding(tx, models.FactoryOnboardingPatch{
			VCSIntegrationID:    &integrationID,
			VCSProvider:         &provider,
			AppRepository:       &repository.FullName,
			AppRepositoryID:     &repositoryID,
			BacklogRepository:   &repository.FullName,
			BacklogRepositoryID: &repositoryID,
			DefaultBranch:       &repository.DefaultBranch,
		}); updateErr != nil {
			return updateErr
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

func findGitHubCatalogRepository(
	ctx context.Context,
	tx *gorm.DB,
	organizationID string,
	repositoryID int64,
) (*models.AccessibleVCSProviderRepository, error) {
	if !githubapp.UserConnectReady(ctx) {
		return models.FindInstalledVCSProviderRepository(tx, models.ProviderGitHub, repositoryID)
	}
	providerUserID, err := factoryVCSProviderUserID(ctx, tx, organizationID, models.ProviderGitHub)
	if err != nil {
		return nil, err
	}
	return models.FindAccessibleVCSProviderRepository(tx, models.ProviderGitHub, providerUserID, repositoryID)
}

func findGitHubCatalogRepositoryByName(
	ctx context.Context,
	tx *gorm.DB,
	organizationID string,
	fullName string,
) (*models.AccessibleVCSProviderRepository, error) {
	if !githubapp.UserConnectReady(ctx) {
		return models.FindInstalledVCSProviderRepositoryByName(tx, models.ProviderGitHub, fullName)
	}
	providerUserID, err := factoryVCSProviderUserID(ctx, tx, organizationID, models.ProviderGitHub)
	if err != nil {
		return nil, err
	}
	return models.FindAccessibleVCSProviderRepositoryByName(tx, models.ProviderGitHub, providerUserID, fullName)
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
	if strings.TrimSpace(previousIntegrationID) == "" || previousIntegrationID == currentIntegrationID.String() {
		return models.SyncVCSProviderBindingRepositories(tx, currentIntegrationID, provider)
	}

	previousID, err := uuid.Parse(previousIntegrationID)
	if err != nil {
		return err
	}
	if _, err := models.FindVCSProviderIntegrationBinding(tx, previousID); errors.Is(err, gorm.ErrRecordNotFound) {
		return models.SyncVCSProviderBindingRepositories(tx, currentIntegrationID, provider)
	} else if err != nil {
		return err
	}
	integrationIDs = append(integrationIDs, previousID)
	if integrationIDs[1].String() < integrationIDs[0].String() {
		integrationIDs[0], integrationIDs[1] = integrationIDs[1], integrationIDs[0]
	}
	for _, integrationID := range integrationIDs {
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
