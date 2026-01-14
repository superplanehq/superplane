package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
)

type OrganizationService struct {
	authorizationService authorization.Authorization
	registry             *registry.Registry
	baseURL              string
	webhooksBaseURL      string
}

func NewOrganizationService(
	authorizationService authorization.Authorization,
	registry *registry.Registry,
	baseURL string,
	webhooksBaseURL string,
) *OrganizationService {
	return &OrganizationService{
		registry:             registry,
		baseURL:              baseURL,
		webhooksBaseURL:      webhooksBaseURL,
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

func (s *OrganizationService) RemoveInvitation(ctx context.Context, req *pb.RemoveInvitationRequest) (*pb.RemoveInvitationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.RemoveInvitation(ctx, s.authorizationService, orgID, req.InvitationId)
}

func (s *OrganizationService) ListApplications(ctx context.Context, req *pb.ListApplicationsRequest) (*pb.ListApplicationsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListApplications(ctx, s.registry, orgID)
}

func (s *OrganizationService) DescribeApplication(ctx context.Context, req *pb.DescribeApplicationRequest) (*pb.DescribeApplicationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.DescribeApplication(ctx, s.registry, orgID, req.InstallationId)
}

func (s *OrganizationService) ListApplicationResources(ctx context.Context, req *pb.ListApplicationResourcesRequest) (*pb.ListApplicationResourcesResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListApplicationResources(ctx, s.registry, orgID, req.InstallationId, req.Type)
}

func (s *OrganizationService) InstallApplication(ctx context.Context, req *pb.InstallApplicationRequest) (*pb.InstallApplicationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.InstallApplication(
		ctx,
		s.registry,
		s.baseURL,
		s.webhooksBaseURL,
		orgID,
		req.AppName,
		req.InstallationName,
		req.Configuration,
	)
}

func (s *OrganizationService) UpdateApplication(ctx context.Context, req *pb.UpdateApplicationRequest) (*pb.UpdateApplicationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateApplication(
		ctx,
		s.registry,
		s.baseURL,
		s.webhooksBaseURL,
		orgID,
		req.InstallationId,
		req.Configuration.AsMap(),
	)
}

func (s *OrganizationService) UninstallApplication(ctx context.Context, req *pb.UninstallApplicationRequest) (*pb.UninstallApplicationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UninstallApplication(ctx, orgID, req.InstallationId)
}
