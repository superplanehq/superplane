package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func selectBitbucketFactoryRepository(
	ctx context.Context,
	orgID uuid.UUID,
	organizationID string,
	req *pb.SelectFactoryVCSProviderRepositoryRequest,
) (*pb.SelectFactoryVCSProviderRepositoryResponse, error) {
	repositoryName := strings.TrimSpace(req.GetRepository())
	if repositoryName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "repository is required")
	}
	if !config.LoadBitbucketForgeAppConfig().Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public Bitbucket app is not configured")
	}

	db := database.DB(ctx)
	accountID, err := linkedBitbucketAccountID(ctx, db, organizationID)
	if err != nil {
		return nil, err
	}
	match, err := findVisibleBitbucketRepository(ctx, db, accountID, repositoryName)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list Bitbucket repositories")
	}
	if match == nil {
		return nil, grpcerrors.PermissionDenied(nil, "VCS repository is not accessible")
	}

	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err = findFactory(tx, orgID, req.GetId())
		if err != nil {
			return err
		}
		integration, bindErr := models.FindOrCreateBitbucketForgeIntegration(
			tx,
			orgID,
			match.InstallationID,
			match.WorkspaceSlug,
		)
		if bindErr != nil {
			return bindErr
		}
		integrationID := integration.ID.String()
		provider := models.ProviderBitbucket
		repositoryID := int64(0)
		fullName := match.FullName
		externalID := match.UUID
		branch := match.DefaultBranch
		return factory.UpdateOnboarding(tx, models.FactoryOnboardingPatch{
			VCSIntegrationID:        &integrationID,
			VCSProvider:             &provider,
			AppRepository:           &fullName,
			AppRepositoryID:         &repositoryID,
			AppRepositoryExternalID: &externalID,
			BacklogRepository:       &fullName,
			BacklogRepositoryID:     &repositoryID,
			DefaultBranch:           &branch,
		})
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
