package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/me"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MeService struct {
	authService authorization.Authorization
}

func NewMeService(authService authorization.Authorization) *MeService {
	return &MeService{
		authService: authService,
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
