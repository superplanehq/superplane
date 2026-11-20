package factories

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// ConnectVCSProviderInstallation creates the organization integration for the
// public GitHub App installation that owns a repository the caller can access.
// The caller's repository access is the proof that they may bind the
// installation, the same check that workspace repository selection uses.
func ConnectVCSProviderInstallation(
	ctx context.Context,
	organizationID string,
	req *pb.ConnectVCSProviderInstallationRequest,
) (*pb.ConnectVCSProviderInstallationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to connect the GitHub installation")
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

	var integration *models.Integration
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		repository, err := findGitHubCatalogRepository(ctx, tx, organizationID, req.GetRepositoryId())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return grpcerrors.PermissionDenied(err, "VCS repository is not accessible")
		}
		if err != nil {
			return err
		}
		installation, err := models.FindVCSProviderInstallation(tx, models.ProviderGitHub, repository.InstallationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return grpcerrors.PermissionDenied(err, "VCS repository is not accessible")
		}
		if err != nil {
			return err
		}
		integration, err = models.FindOrCreateVCSProviderBinding(
			tx,
			orgID,
			models.ProviderGitHub,
			repository.InstallationID,
			installation.AccountLogin,
		)
		return err
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, factoryErrorToStatus(err, "failed to connect the GitHub installation")
	}

	return &pb.ConnectVCSProviderInstallationResponse{IntegrationId: integration.ID.String()}, nil
}
