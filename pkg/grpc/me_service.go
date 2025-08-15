package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/me"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MeService struct{}

func NewMeService() *MeService {
	return &MeService{}
}

func (s *MeService) Me(ctx context.Context, req *emptypb.Empty) (*pb.User, error) {
	return me.GetUser(ctx)
}

func (s *MeService) RegenerateToken(ctx context.Context, req *emptypb.Empty) (*pb.RegenerateTokenResponse, error) {
	return me.RegenerateToken(ctx)
}
