package me

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeVCSProviderOnboarding(ctx context.Context, provider string) (*pb.DescribeVCSProviderOnboardingResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}

	if provider == models.ProviderBitbucket {
		return describeBitbucketOnboarding(ctx)
	}

	response := &pb.DescribeVCSProviderOnboardingResponse{ProviderConfigured: vcsProviderConfigured(ctx, provider)}
	identities, err := vcsProviderIdentities(ctx, provider)
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
	repositories, err := models.ListAccessibleVCSProviderRepositories(database.DB(ctx), provider, identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list accessible repositories")
	}
	response.Repositories = make([]*pb.VCSProviderRepository, 0, len(repositories))
	for _, repository := range repositories {
		response.Repositories = append(response.Repositories, &pb.VCSProviderRepository{
			RepositoryId:   repository.RepositoryID,
			InstallationId: repository.InstallationID,
			FullName:       repository.FullName,
			Private:        repository.Private,
			DefaultBranch:  repository.DefaultBranch,
			AccountLogin:   repository.AccountLogin,
			AccountType:    repository.AccountType,
		})
	}

	requests, err := models.ListVCSProviderInstallRequests(database.DB(ctx), provider, identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list pending installation requests")
	}
	response.PendingRequests = make([]*pb.VCSProviderInstallRequest, 0, len(requests))
	for _, request := range requests {
		response.PendingRequests = append(response.PendingRequests, &pb.VCSProviderInstallRequest{
			RequestId:    request.RequestID,
			AccountLogin: request.AccountLogin,
			AccountType:  request.AccountType,
			RequestedAt:  timestamppb.New(request.RequestedAt),
		})
	}

	organizationID, err := currentOrganizationID(ctx)
	if err != nil {
		return nil, err
	}
	response.Synchronizing, err = models.VCSProviderCatalogSynchronizing(
		database.DB(ctx),
		provider,
		identity.userID,
		organizationID,
	)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to inspect repository synchronization")
	}
	return response, nil
}

func SelectVCSProviderOnboardingIdentity(
	ctx context.Context,
	provider string,
	userID int64,
) (*pb.SelectVCSProviderOnboardingIdentityResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "provider user id is required")
	}
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, err
	}
	err = models.SelectAccountLinkedAccount(database.DB(ctx), accountID, provider, strconv.FormatInt(userID, 10))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.PermissionDenied(err, "provider account is not linked")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to select provider account")
	}
	return &pb.SelectVCSProviderOnboardingIdentityResponse{}, nil
}

func StartVCSProviderInstallation(ctx context.Context, provider string) (*pb.StartVCSProviderInstallationResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	if provider == models.ProviderBitbucket {
		return startBitbucketInstallation(ctx)
	}
	cfg, err := githubapp.ResolveProcess(ctx)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load public GitHub App")
	}
	if !cfg.Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}
	if _, err := currentVCSProviderIdentity(ctx, provider); err != nil {
		return nil, vcsProviderIdentityError(err)
	}

	organizationID, ok := authentication.GetOrganizationIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}
	state, err := githubcommon.SignHostedAppInstallState(cfg.WebhookSecret, organizationUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create GitHub App setup state")
	}
	return &pb.StartVCSProviderInstallationResponse{
		Url: githubcommon.HostedAppInstallURL(cfg.Slug, state),
	}, nil
}

func ConfigureVCSProviderInstallation(
	ctx context.Context,
	provider string,
	installationID int64,
) (*pb.ConfigureVCSProviderInstallationResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	if installationID <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "installation id is required")
	}
	identity, err := currentVCSProviderIdentity(ctx, provider)
	if err != nil {
		return nil, vcsProviderIdentityError(err)
	}
	repositories, err := models.ListAccessibleVCSProviderRepositories(database.DB(ctx), provider, identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to verify installation access")
	}
	accessible := false
	for _, repository := range repositories {
		if repository.InstallationID == installationID {
			accessible = true
			break
		}
	}
	if !accessible {
		return nil, grpcerrors.PermissionDenied(nil, "provider installation is not accessible")
	}
	installation, err := models.FindVCSProviderInstallation(database.DB(ctx), provider, installationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load provider installation")
	}
	if installation.HTMLURL == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "provider installation settings are still synchronizing")
	}
	return &pb.ConfigureVCSProviderInstallationResponse{Url: installation.HTMLURL}, nil
}

func RefreshVCSProviderOnboarding(
	ctx context.Context,
	provider string,
	repositoryID *int64,
) (*pb.RefreshVCSProviderOnboardingResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	if provider == models.ProviderBitbucket {
		return &pb.RefreshVCSProviderOnboardingResponse{}, nil
	}
	identity, err := currentVCSProviderIdentity(ctx, provider)
	if err != nil {
		return nil, vcsProviderIdentityError(err)
	}
	repositories, err := models.ListAccessibleVCSProviderRepositories(database.DB(ctx), provider, identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list accessible repositories")
	}

	now := time.Now()
	if err := models.EnqueueVCSProviderReconciliation(database.DB(ctx), provider, now); err != nil {
		return nil, grpcerrors.Internal(err, "failed to queue provider reconciliation")
	}
	found := repositoryID == nil
	for _, repository := range repositories {
		if repositoryID != nil && repository.RepositoryID != *repositoryID {
			continue
		}
		found = true
		if err := models.EnqueueVCSProviderRepositorySync(
			database.DB(ctx),
			provider,
			repository.RepositoryID,
			now,
			models.VCSProviderRepositorySyncPriorityInteractive,
		); err != nil {
			return nil, grpcerrors.Internal(err, "failed to queue repository refresh")
		}
	}
	if !found {
		return nil, grpcerrors.PermissionDenied(nil, "repository is not accessible")
	}
	return &pb.RefreshVCSProviderOnboardingResponse{}, nil
}

type vcsProviderIdentity struct {
	userID         int64
	providerUserID string
	login          string
	active         bool
}

func currentAccountID(ctx context.Context) (uuid.UUID, error) {
	userID, userSet := authentication.GetUserIdFromMetadata(ctx)
	organizationID, organizationSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !userSet || !organizationSet {
		return uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := loadUser(ctx, organizationID, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if user.AccountID == nil {
		return uuid.Nil, gorm.ErrRecordNotFound
	}
	return *user.AccountID, nil
}

func vcsProviderIdentities(ctx context.Context, provider string) ([]vcsProviderIdentity, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, err
	}
	linkedAccounts, err := models.ListAccountLinkedAccounts(database.DB(ctx), accountID)
	if err != nil {
		return nil, err
	}
	identities := make([]vcsProviderIdentity, 0, len(linkedAccounts))
	for _, linked := range linkedAccounts {
		if linked.Provider != provider {
			continue
		}
		if provider == models.ProviderBitbucket {
			bitbucketAccountID, parseErr := normalizeBitbucketAccountID(linked.ProviderID)
			if parseErr != nil {
				return nil, grpcerrors.FailedPrecondition(parseErr, "linked Bitbucket account has an invalid user id")
			}
			identities = append(identities, vcsProviderIdentity{
				providerUserID: bitbucketAccountID,
				login:          linked.Username,
				active:         linked.Active,
			})
			continue
		}
		numericID, err := strconv.ParseInt(linked.ProviderID, 10, 64)
		if err != nil || numericID <= 0 {
			return nil, grpcerrors.FailedPrecondition(err, "linked provider account has an invalid user id")
		}
		identities = append(identities, vcsProviderIdentity{
			userID: numericID,
			login:  linked.Username,
			active: linked.Active,
		})
	}
	return identities, nil
}

func currentVCSProviderIdentity(ctx context.Context, provider string) (*vcsProviderIdentity, error) {
	identities, err := vcsProviderIdentities(ctx, provider)
	if err != nil {
		return nil, err
	}
	for index := range identities {
		if identities[index].active {
			return &identities[index], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func serializeVCSProviderIdentity(identity *vcsProviderIdentity) *pb.VCSProviderIdentity {
	return &pb.VCSProviderIdentity{
		UserId:         identity.userID,
		Login:          identity.login,
		ProviderUserId: identity.providerUserID,
	}
}

func supportedVCSProvider(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case models.ProviderGitHub, models.ProviderBitbucket:
		return provider, nil
	default:
		return "", grpcerrors.InvalidArgument(nil, "VCS provider is not supported")
	}
}

func vcsProviderConfigured(ctx context.Context, provider string) bool {
	switch provider {
	case models.ProviderGitHub:
		return githubapp.UserConnectReady(ctx)
	case models.ProviderBitbucket:
		return config.LoadBitbucketForgeAppConfig().Enabled()
	default:
		return false
	}
}

func vcsProviderIdentityError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return grpcerrors.FailedPrecondition(err, "connect a provider account first")
	}
	return err
}
