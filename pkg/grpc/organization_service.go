package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type OrganizationService struct {
	authorizationService authorization.Authorization
	registry             *registry.Registry
	oidcProvider         oidc.Provider
	baseURL              string
	webhooksBaseURL      string
}

func NewOrganizationService(
	authorizationService authorization.Authorization,
	registry *registry.Registry,
	oidcProvider oidc.Provider,
	baseURL string,
	webhooksBaseURL string,
) *OrganizationService {
	return &OrganizationService{
		registry:             registry,
		oidcProvider:         oidcProvider,
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

func (s *OrganizationService) GetInviteLink(ctx context.Context, req *pb.GetInviteLinkRequest) (*pb.GetInviteLinkResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.GetInviteLink(orgID)
}

func (s *OrganizationService) UpdateInviteLink(ctx context.Context, req *pb.UpdateInviteLinkRequest) (*pb.UpdateInviteLinkResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateInviteLink(orgID, req.Enabled)
}

func (s *OrganizationService) ResetInviteLink(ctx context.Context, req *pb.ResetInviteLinkRequest) (*pb.ResetInviteLinkResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ResetInviteLink(orgID)
}

func (s *OrganizationService) AcceptInviteLink(ctx context.Context, req *pb.InviteLink) (*structpb.Struct, error) {
	accountID, err := accountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return organizations.AcceptInviteLink(ctx, s.authorizationService, accountID, req.Token)
}

func (s *OrganizationService) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListIntegrations(ctx, s.registry, orgID)
}

func (s *OrganizationService) DescribeIntegration(ctx context.Context, req *pb.DescribeIntegrationRequest) (*pb.DescribeIntegrationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.DescribeIntegration(ctx, s.registry, orgID, req.IntegrationId)
}

func (s *OrganizationService) ListIntegrationResources(ctx context.Context, req *pb.ListIntegrationResourcesRequest) (*pb.ListIntegrationResourcesResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListIntegrationResources(ctx, s.registry, orgID, req.IntegrationId, req.Parameters)
}

func (s *OrganizationService) CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.CreateIntegrationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.CreateIntegration(
		ctx,
		s.registry,
		s.oidcProvider,
		s.baseURL,
		s.webhooksBaseURL,
		orgID,
		req.IntegrationName,
		req.Name,
		req.Configuration,
	)
}

func (s *OrganizationService) UpdateIntegration(ctx context.Context, req *pb.UpdateIntegrationRequest) (*pb.UpdateIntegrationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	configuration := map[string]any{}
	if req.Configuration != nil {
		configuration = req.Configuration.AsMap()
	}

	return organizations.UpdateIntegration(
		ctx,
		s.registry,
		s.oidcProvider,
		s.baseURL,
		s.webhooksBaseURL,
		orgID,
		req.IntegrationId,
		configuration,
		req.Name,
	)
}

func (s *OrganizationService) DeleteIntegration(ctx context.Context, req *pb.DeleteIntegrationRequest) (*pb.DeleteIntegrationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.DeleteIntegration(ctx, orgID, req.IntegrationId)
}

func accountIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "account not found")
	}

	accountMeta := md.Get("x-account-id")
	if len(accountMeta) == 0 || accountMeta[0] == "" {
		return "", status.Error(codes.Unauthenticated, "account not found")
	}

	return accountMeta[0], nil
}
