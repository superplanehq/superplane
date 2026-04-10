package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
)

type UsersService struct {
	pb.UnimplementedUsersServer
	authService authorization.Authorization
}

func NewUsersService(authService authorization.Authorization) *UsersService {
	return &UsersService{
		authService: authService,
	}
}

func (s *UsersService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.ListUsers(ctx, domainType, domainID, req.IncludeRoles, s.authService)
}
