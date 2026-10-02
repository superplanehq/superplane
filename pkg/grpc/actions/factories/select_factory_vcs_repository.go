package factories

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
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
	if provider != models.ProviderGitHub {
		return nil, grpcerrors.InvalidArgument(nil, "VCS provider is not supported")
	}
	if req.GetRepositoryId() <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "repository id is required")
	}
	app := config.LoadGitHubHostedAppConfig()
	if !app.Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}

	db := database.DB(ctx)
	available, err := models.IsIntakeAvailableForOrganization(db, provider, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select VCS repository")
	}
	if !available {
		return nil, grpcerrors.FailedPrecondition(nil, "VCS provider is not available for the organization")
	}
	providerUserID, err := factoryVCSProviderUserID(ctx, db, organizationID, provider)
	if err != nil {
		return nil, err
	}

	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err = findFactory(tx, orgID, req.GetId())
		if err != nil {
			return err
		}

		repository, findErr := models.FindAccessibleVCSProviderRepository(tx, provider, providerUserID, req.GetRepositoryId())
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
