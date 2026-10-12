package grpc

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/grpc/actions/me"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/emptypb"
)

const vcsProviderInstallationCheckInterval = 10 * time.Second

type MeService struct {
	authService        authorization.Authorization
	installationChecks *me.VCSProviderInstallationChecks
}

func NewMeService(authService authorization.Authorization) *MeService {
	return &MeService{
		authService:        authService,
		installationChecks: me.NewVCSProviderInstallationChecks(vcsProviderInstallationCheckInterval),
	}
}

func (s *MeService) Me(ctx context.Context, req *pb.MeRequest) (*pb.MeResponse, error) {
	return me.GetUser(ctx, s.authService, req.GetIncludePermissions())
}

func (s *MeService) RegenerateToken(ctx context.Context, req *emptypb.Empty) (*pb.RegenerateTokenResponse, error) {
	return me.RegenerateToken(ctx)
}

func (s *MeService) ListTokens(ctx context.Context, req *pb.ListTokensRequest) (*pb.ListTokensResponse, error) {
	return me.ListTokens(ctx)
}

func (s *MeService) CreateToken(ctx context.Context, req *pb.CreateTokenRequest) (*pb.CreateTokenResponse, error) {
	return me.CreateToken(ctx, req)
}

func (s *MeService) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	return me.RevokeToken(ctx, req)
}

func (s *MeService) DescribeNotificationSettings(ctx context.Context, req *pb.DescribeNotificationSettingsRequest) (*pb.DescribeNotificationSettingsResponse, error) {
	return me.DescribeNotificationSettings(ctx)
}

func (s *MeService) UpdateNotificationSettings(ctx context.Context, req *pb.UpdateNotificationSettingsRequest) (*pb.UpdateNotificationSettingsResponse, error) {
	return me.UpdateNotificationSettings(ctx, req)
}

func (s *MeService) DescribeLastLocation(ctx context.Context, req *pb.DescribeLastLocationRequest) (*pb.DescribeLastLocationResponse, error) {
	return me.DescribeLastLocation(ctx)
}

func (s *MeService) SaveLastLocation(ctx context.Context, req *pb.SaveLastLocationRequest) (*pb.SaveLastLocationResponse, error) {
	return me.SaveLastLocation(ctx, req)
}

func (s *MeService) DescribeVCSProviderOnboarding(ctx context.Context, req *pb.DescribeVCSProviderOnboardingRequest) (*pb.DescribeVCSProviderOnboardingResponse, error) {
	return me.DescribeVCSProviderOnboarding(ctx, req.GetProvider())
}

func (s *MeService) SelectVCSProviderOnboardingIdentity(ctx context.Context, req *pb.SelectVCSProviderOnboardingIdentityRequest) (*pb.SelectVCSProviderOnboardingIdentityResponse, error) {
	return me.SelectVCSProviderOnboardingIdentity(ctx, req.GetProvider(), req.GetUserId())
}

func (s *MeService) StartVCSProviderInstallation(ctx context.Context, req *pb.StartVCSProviderInstallationRequest) (*pb.StartVCSProviderInstallationResponse, error) {
	return me.StartVCSProviderInstallation(ctx, req.GetProvider())
}

func (s *MeService) ConfigureVCSProviderInstallation(ctx context.Context, req *pb.ConfigureVCSProviderInstallationRequest) (*pb.ConfigureVCSProviderInstallationResponse, error) {
	return me.ConfigureVCSProviderInstallation(ctx, req.GetProvider(), req.GetInstallationId())
}

func (s *MeService) RefreshVCSProviderOnboarding(ctx context.Context, req *pb.RefreshVCSProviderOnboardingRequest) (*pb.RefreshVCSProviderOnboardingResponse, error) {
	return me.RefreshVCSProviderOnboarding(ctx, req.GetProvider(), req.RepositoryId)
}

func (s *MeService) VerifyVCSProviderInstallations(ctx context.Context, req *pb.VerifyVCSProviderInstallationsRequest) (*pb.VerifyVCSProviderInstallationsResponse, error) {
	return me.VerifyVCSProviderInstallations(ctx, req.GetProvider(), s.installationChecks, githubInstallationVerifier{})
}

type githubInstallationVerifier struct{}

func (githubInstallationVerifier) VerifyInstallation(ctx context.Context, installationID int64) error {
	cfg, err := githubapp.ResolveProcess(ctx)
	if err != nil {
		return err
	}
	catalog, err := githubapp.NewCatalog(database.DB(ctx), cfg)
	if err != nil {
		return err
	}
	return catalog.VerifyInstallation(ctx, installationID)
}
