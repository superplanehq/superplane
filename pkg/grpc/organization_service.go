package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

type OrganizationService struct {
	authorizationService authorization.Authorization
}

func NewOrganizationService(authorizationService authorization.Authorization) *OrganizationService {
	return &OrganizationService{
		authorizationService: authorizationService,
	}
}

func (s *OrganizationService) DescribeOrganization(ctx context.Context, req *pb.DescribeOrganizationRequest) (*pb.DescribeOrganizationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.DescribeOrganization(ctx, orgID)
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateOrganization(ctx, orgID, req.Organization)
}

func (s *OrganizationService) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (*pb.RemoveUserResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.RemoveUser(ctx, s.authorizationService, orgID, req.UserId)
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.DeleteOrganization(ctx, s.authorizationService, orgID)
}

func (s *OrganizationService) CreateInvitation(ctx context.Context, req *pb.CreateInvitationRequest) (*pb.CreateInvitationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.CreateInvitation(ctx, s.authorizationService, orgID, req.Email)
}

func (s *OrganizationService) ListInvitations(ctx context.Context, req *pb.ListInvitationsRequest) (*pb.ListInvitationsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListInvitations(ctx, orgID)
}
