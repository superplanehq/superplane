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
