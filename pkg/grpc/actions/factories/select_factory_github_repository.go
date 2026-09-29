package factories

import (
	"context"
	"errors"
	"strconv"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func SelectFactoryGitHubRepository(
	ctx context.Context,
	organizationID string,
	req *pb.SelectFactoryGitHubRepositoryRequest,
) (*pb.SelectFactoryGitHubRepositoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select GitHub repository")
	}
	if req.GetRepositoryId() <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "repository id is required")
	}
	app := config.LoadGitHubHostedAppConfig()
	if !app.Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}

	db := database.DB(ctx)
	githubUserID, err := factoryGitHubUserID(ctx, db, organizationID)
	if err != nil {
		return nil, err
	}

	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err = findFactory(tx, orgID, req.GetId())
		if err != nil {
			return err
		}

		repository, findErr := models.FindAccessibleGitHubAppRepository(tx, githubUserID, req.GetRepositoryId())
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return grpcerrors.PermissionDenied(findErr, "GitHub repository is not accessible")
		}
		if findErr != nil {
			return findErr
		}

		installation, findErr := models.FindGitHubAppInstallation(tx, repository.InstallationID)
		if findErr != nil {
			return findErr
		}
		integration, bindErr := models.FindOrCreateHostedGitHubBinding(
			tx,
			orgID,
			repository.InstallationID,
			installation.AccountLogin,
		)
		if bindErr != nil {
			return bindErr
		}

		integrationID := integration.ID.String()
		repositoryID := repository.RepositoryID
		return factory.UpdateOnboarding(tx, models.FactoryOnboardingPatch{
			VCSIntegrationID:    &integrationID,
			AppRepository:       &repository.FullName,
			AppRepositoryID:     &repositoryID,
			BacklogRepository:   &repository.FullName,
			BacklogRepositoryID: &repositoryID,
			DefaultBranch:       &repository.DefaultBranch,
		})
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select GitHub repository")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select GitHub repository")
	}
	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to select GitHub repository")
	}
	return &pb.SelectFactoryGitHubRepositoryResponse{Factory: serialized}, nil
}

func factoryGitHubUserID(ctx context.Context, db *gorm.DB, organizationID string) (int64, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return 0, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := models.FindActiveUserByIDInTransaction(db, organizationID, userID)
	if err != nil {
		return 0, factoryErrorToStatus(err, "failed to load user")
	}
	if user.AccountID == nil {
		return 0, grpcerrors.FailedPrecondition(nil, "connect a GitHub account first")
	}
	linked, err := models.FindAccountLinkedAccount(db, *user.AccountID, models.ProviderGitHub)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, grpcerrors.FailedPrecondition(err, "connect a GitHub account first")
	}
	if err != nil {
		return 0, grpcerrors.Internal(err, "failed to load linked GitHub account")
	}
	numericID, err := strconv.ParseInt(linked.ProviderID, 10, 64)
	if err != nil || numericID <= 0 {
		return 0, grpcerrors.FailedPrecondition(err, "linked GitHub account has an invalid user id")
	}
	return numericID, nil
}
