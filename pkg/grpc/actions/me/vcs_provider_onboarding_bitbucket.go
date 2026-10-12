package me

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
)

func describeBitbucketOnboarding(ctx context.Context) (*pb.DescribeVCSProviderOnboardingResponse, error) {
	cfg := config.LoadBitbucketForgeAppConfig()
	response := &pb.DescribeVCSProviderOnboardingResponse{ProviderConfigured: cfg.Enabled()}
	if cfg.Enabled() {
		response.InstallUrl = cfg.InstallURL
	}

	identities, err := vcsProviderIdentities(ctx, models.ProviderBitbucket)
	if err != nil {
		return nil, err
	}
	if len(identities) == 0 {
		return response, nil
	}

	response.Identities = make([]*pb.VCSProviderIdentity, 0, len(identities))
	var identity *vcsProviderIdentity
	for index := range identities {
		candidate := &identities[index]
		response.Identities = append(response.Identities, serializeVCSProviderIdentity(candidate))
		if candidate.active {
			identity = candidate
		}
	}
	if identity == nil {
		return nil, grpcerrors.FailedPrecondition(nil, "linked provider accounts have no active identity")
	}
	response.Identity = serializeVCSProviderIdentity(identity)
	if !cfg.Enabled() {
		return response, nil
	}

	catalog, err := bitbucketCatalogForAccount(ctx, identity.providerUserID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to check Bitbucket workspace access")
	}
	response.Repositories = serializeBitbucketRepositories(catalog.Repositories)
	for _, workspace := range catalog.Workspaces {
		response.InstalledWorkspaces = append(response.InstalledWorkspaces, &pb.VCSProviderInstalledWorkspace{
			InstallationId: workspace.ID, ExternalId: workspace.WorkspaceUUID, Slug: workspace.WorkspaceSlug,
		})
	}
	return response, nil
}

func startBitbucketInstallation(ctx context.Context) (*pb.StartVCSProviderInstallationResponse, error) {
	cfg := config.LoadBitbucketForgeAppConfig()
	if !cfg.Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public Bitbucket app is not configured")
	}
	if _, err := currentVCSProviderIdentity(ctx, models.ProviderBitbucket); err != nil {
		return nil, vcsProviderIdentityError(err)
	}
	return &pb.StartVCSProviderInstallationResponse{Url: cfg.InstallURL}, nil
}

func bitbucketCatalogForAccount(ctx context.Context, accountUUID string) (bitbucket.VisibleCatalog, error) {
	installations, err := models.ListActiveBitbucketForgeInstallations(database.DB(ctx))
	if err != nil {
		return bitbucket.VisibleCatalog{}, err
	}
	refs := make([]bitbucket.InstallationRef, 0, len(installations))
	for _, installation := range installations {
		refs = append(refs, bitbucket.InstallationRef{
			ID:            installation.InstallationID,
			WorkspaceUUID: installation.WorkspaceUUID,
			WorkspaceSlug: installation.WorkspaceSlug,
		})
	}
	return bitbucket.CurrentDirectory().CatalogVisibleTo(ctx, accountUUID, refs, forgeSystemToken)
}

func serializeBitbucketRepositories(visible []bitbucket.VisibleRepository) []*pb.VCSProviderRepository {
	repositories := make([]*pb.VCSProviderRepository, 0, len(visible))
	for _, repository := range visible {
		repositories = append(repositories, &pb.VCSProviderRepository{
			FullName:      repository.FullName,
			Private:       repository.Private,
			DefaultBranch: repository.DefaultBranch,
			AccountLogin:  repository.WorkspaceSlug,
			AccountType:   "workspace",
			ExternalId:    repository.UUID,
		})
	}
	return repositories
}

func forgeSystemToken(installationID string) (string, error) {
	token, _, err := bitbucketapp.CurrentSystemToken(installationID)
	if err != nil {
		log.WithError(err).WithField("installation_id", installationID).Warn("skipped Bitbucket Forge installation")
		return "", err
	}
	return token, nil
}

func normalizeBitbucketAccountID(value string) (string, error) {
	return bitbucket.NormalizeAccountID(strings.TrimSpace(value))
}
