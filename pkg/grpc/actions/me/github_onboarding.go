package me

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeGitHubOnboarding(ctx context.Context) (*pb.DescribeGitHubOnboardingResponse, error) {
	cfg := config.LoadGitHubHostedAppConfig()
	response := &pb.DescribeGitHubOnboardingResponse{AppConfigured: cfg.Enabled()}
	identity, err := currentGitHubIdentity(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response, nil
	}
	if err != nil {
		return nil, err
	}

	response.Identity = &pb.GitHubIdentity{UserId: identity.userID, Login: identity.login}
	repositories, err := models.ListAccessibleGitHubAppRepositories(database.DB(ctx), identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list accessible GitHub repositories")
	}
	response.Repositories = make([]*pb.GitHubOnboardingRepository, 0, len(repositories))
	for _, repository := range repositories {
		response.Repositories = append(response.Repositories, &pb.GitHubOnboardingRepository{
			RepositoryId:   repository.RepositoryID,
			InstallationId: repository.InstallationID,
			FullName:       repository.FullName,
			Private:        repository.Private,
			DefaultBranch:  repository.DefaultBranch,
			AccountLogin:   repository.AccountLogin,
			AccountType:    repository.AccountType,
		})
	}

	requests, err := models.ListGitHubAppInstallRequests(database.DB(ctx), identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list pending GitHub installation requests")
	}
	response.PendingRequests = make([]*pb.GitHubAppInstallRequest, 0, len(requests))
	for _, request := range requests {
		response.PendingRequests = append(response.PendingRequests, &pb.GitHubAppInstallRequest{
			RequestId:    request.RequestID,
			AccountLogin: request.AccountLogin,
			AccountType:  request.AccountType,
			RequestedAt:  timestamppb.New(request.RequestedAt),
		})
	}

	response.Synchronizing, err = models.GitHubAppCatalogSynchronizing(database.DB(ctx))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to inspect GitHub repository synchronization")
	}
	return response, nil
}

func StartGitHubAppInstallation(ctx context.Context) (*pb.StartGitHubAppInstallationResponse, error) {
	cfg := config.LoadGitHubHostedAppConfig()
	if !cfg.Enabled() {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}
	if _, err := currentGitHubIdentity(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.FailedPrecondition(err, "connect a GitHub account first")
		}
		return nil, err
	}

	organizationID, ok := authentication.GetOrganizationIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	state := "o_" + organizationID
	return &pb.StartGitHubAppInstallationResponse{
		Url: githubcommon.HostedAppInstallURL(cfg.Slug, state),
	}, nil
}

func ConfigureGitHubAppInstallation(ctx context.Context, installationID int64) (*pb.ConfigureGitHubAppInstallationResponse, error) {
	if installationID <= 0 {
		return nil, grpcerrors.InvalidArgument(nil, "installation id is required")
	}
	identity, err := currentGitHubIdentity(ctx)
	if err != nil {
		return nil, githubIdentityError(err)
	}
	repositories, err := models.ListAccessibleGitHubAppRepositories(database.DB(ctx), identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to verify GitHub installation access")
	}
	accessible := false
	for _, repository := range repositories {
		if repository.InstallationID == installationID {
			accessible = true
			break
		}
	}
	if !accessible {
		return nil, grpcerrors.PermissionDenied(nil, "GitHub installation is not accessible")
	}
	installation, err := models.FindGitHubAppInstallation(database.DB(ctx), installationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load GitHub installation")
	}
	if installation.HTMLURL == "" {
		return nil, grpcerrors.FailedPrecondition(nil, "GitHub installation settings are still synchronizing")
	}
	return &pb.ConfigureGitHubAppInstallationResponse{Url: installation.HTMLURL}, nil
}

func RefreshGitHubOnboarding(ctx context.Context, repositoryID *int64) (*pb.RefreshGitHubOnboardingResponse, error) {
	identity, err := currentGitHubIdentity(ctx)
	if err != nil {
		return nil, githubIdentityError(err)
	}
	repositories, err := models.ListAccessibleGitHubAppRepositories(database.DB(ctx), identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list accessible GitHub repositories")
	}

	now := time.Now()
	if err := models.EnqueueGitHubAppReconciliation(database.DB(ctx), now); err != nil {
		return nil, grpcerrors.Internal(err, "failed to queue GitHub App reconciliation")
	}
	found := repositoryID == nil
	for _, repository := range repositories {
		if repositoryID != nil && repository.RepositoryID != *repositoryID {
			continue
		}
		found = true
		if err := models.EnqueueGitHubAppRepositorySync(database.DB(ctx), repository.RepositoryID, now); err != nil {
			return nil, grpcerrors.Internal(err, "failed to queue GitHub repository refresh")
		}
	}
	if !found {
		return nil, grpcerrors.PermissionDenied(nil, "GitHub repository is not accessible")
	}
	return &pb.RefreshGitHubOnboardingResponse{}, nil
}

type githubIdentity struct {
	userID int64
	login  string
}

func currentGitHubIdentity(ctx context.Context) (*githubIdentity, error) {
	userID, userSet := authentication.GetUserIdFromMetadata(ctx)
	organizationID, organizationSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !userSet || !organizationSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := loadUser(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if user.AccountID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	linked, err := models.FindAccountLinkedAccount(database.DB(ctx), *user.AccountID, models.ProviderGitHub)
	if err != nil {
		return nil, err
	}
	numericID, err := strconv.ParseInt(linked.ProviderID, 10, 64)
	if err != nil || numericID <= 0 {
		return nil, grpcerrors.FailedPrecondition(err, "linked GitHub account has an invalid user id")
	}
	return &githubIdentity{userID: numericID, login: linked.Username}, nil
}

func githubIdentityError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return grpcerrors.FailedPrecondition(err, "connect a GitHub account first")
	}
	return err
}
