package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type OrganizationService struct {
	pb.UnimplementedOrganizationsServer
	invitations          organizations.InvitationService
	authorizationService authorization.Authorization
	registry             *registry.Registry
	oidcProvider         oidc.Provider
	baseURL              string
	webhooksBaseURL      string
	billing              enterprise.BillingOperations
}

func NewOrganizationService(
	authorizationService authorization.Authorization,
	registry *registry.Registry,
	oidcProvider oidc.Provider,
	baseURL string,
	webhooksBaseURL string,
	billingImplementation enterprise.BillingOperations,
) *OrganizationService {
	return &OrganizationService{
		billing:              billingImplementation,
		registry:             registry,
		oidcProvider:         oidcProvider,
		baseURL:              baseURL,
		webhooksBaseURL:      webhooksBaseURL,
		authorizationService: authorizationService,
		invitations:          organizations.InvitationService{AuthService: authorizationService, BaseURL: baseURL, EmailService: services.NewNoopEmailService()},
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

func (s *OrganizationService) SetUserOwner(ctx context.Context, req *pb.SetUserOwnerRequest) (*pb.SetUserOwnerResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.SetUserOwner(ctx, orgID, req.UserId, req.IsOwner)
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.DeleteOrganization(ctx, s.authorizationService, orgID)
}

func (s *OrganizationService) DescribeOrganizationWorkspaceUsage(
	ctx context.Context,
	req *pb.DescribeOrganizationWorkspaceUsageRequest,
) (*pb.DescribeOrganizationWorkspaceUsageResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.DescribeOrganizationWorkspaceUsage(ctx, orgID, req)
}

func (s *OrganizationService) DescribeOrganizationSpendingReport(
	ctx context.Context,
	req *pb.DescribeOrganizationSpendingReportRequest,
) (*pb.DescribeOrganizationSpendingReportResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.DescribeOrganizationSpendingReport(ctx, orgID, req)
}

func (s *OrganizationService) ListOrganizationRunnerFleets(
	ctx context.Context,
	_ *pb.ListOrganizationRunnerFleetsRequest,
) (*pb.ListOrganizationRunnerFleetsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListOrganizationRunnerFleets(ctx, orgID)
}

func (s *OrganizationService) ListHostedLLMModels(
	ctx context.Context,
	req *pb.ListHostedLLMModelsRequest,
) (*pb.ListHostedLLMModelsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListHostedLLMModels(ctx, orgID, req)
}

func (s *OrganizationService) ListSelectableLLMModels(
	ctx context.Context,
	req *pb.ListSelectableLLMModelsRequest,
) (*pb.ListSelectableLLMModelsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListSelectableLLMModels(ctx, s.registry, orgID, req)
}

func (s *OrganizationService) ListBYOKLLMModels(
	ctx context.Context,
	req *pb.ListBYOKLLMModelsRequest,
) (*pb.ListBYOKLLMModelsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListBYOKLLMModels(ctx, s.registry, orgID, req)
}

func (s *OrganizationService) UpdateBYOKLLMModels(
	ctx context.Context,
	req *pb.UpdateBYOKLLMModelsRequest,
) (*pb.UpdateBYOKLLMModelsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateBYOKLLMModels(ctx, orgID, req)
}

func (s *OrganizationService) ListOrganizationHostedLLMModels(ctx context.Context, _ *pb.ListOrganizationHostedLLMModelsRequest) (*pb.ListOrganizationHostedLLMModelsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.ListOrganizationHostedLLMModels(ctx, orgID)
}

func (s *OrganizationService) UpdateOrganizationHostedLLMModels(ctx context.Context, req *pb.UpdateOrganizationHostedLLMModelsRequest) (*pb.UpdateOrganizationHostedLLMModelsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateOrganizationHostedLLMModels(ctx, orgID, req)
}

func (s *OrganizationService) ListHostedCreditProducts(
	ctx context.Context,
	req *pb.ListHostedCreditProductsRequest,
) (*pb.ListHostedCreditProductsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.ListHostedCreditProducts(ctx, orgID, req)
}

func (s *OrganizationService) QuoteHostedCreditCheckout(
	ctx context.Context,
	req *pb.QuoteHostedCreditCheckoutRequest,
) (*pb.QuoteHostedCreditCheckoutResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.QuoteHostedCreditCheckout(ctx, orgID, req)
}

func (s *OrganizationService) CreateHostedCreditCheckout(
	ctx context.Context,
	req *pb.CreateHostedCreditCheckoutRequest,
) (*pb.CreateHostedCreditCheckoutResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	accountID, _ := accountIDFromContext(ctx)
	return s.billing.CreateHostedCreditCheckout(ctx, orgID, req, accountID, s.baseURL)
}

func (s *OrganizationService) CreateBillingPortalSession(
	ctx context.Context,
	req *pb.CreateBillingPortalSessionRequest,
) (*pb.CreateBillingPortalSessionResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.CreateBillingPortalSession(ctx, orgID, req)
}

func (s *OrganizationService) ListOrganizationCreditGrants(
	ctx context.Context,
	req *pb.ListOrganizationCreditGrantsRequest,
) (*pb.ListOrganizationCreditGrantsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.ListOrganizationCreditGrants(ctx, orgID, req)
}

func (s *OrganizationService) DescribeOrganizationBilling(
	ctx context.Context,
	req *pb.DescribeOrganizationBillingRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.DescribeOrganizationBilling(ctx, orgID, req)
}

func (s *OrganizationService) CreateBusinessCheckout(
	ctx context.Context,
	req *pb.CreateBusinessCheckoutRequest,
) (*pb.CreateBusinessCheckoutResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	accountID, _ := accountIDFromContext(ctx)
	return s.billing.CreateBusinessCheckout(ctx, orgID, req, accountID, s.baseURL)
}

func (s *OrganizationService) SelectBillingPlan(ctx context.Context, req *pb.SelectBillingPlanRequest) (*pb.CreateBusinessCheckoutResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	accountID, _ := accountIDFromContext(ctx)
	return s.billing.SelectBillingPlan(ctx, orgID, req, accountID, s.baseURL)
}

func (s *OrganizationService) CreatePaymentMethodSession(ctx context.Context, req *pb.CreatePaymentMethodSessionRequest) (*pb.CreateBillingPortalSessionResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	accountID, _ := accountIDFromContext(ctx)
	return s.billing.CreatePaymentMethodSession(ctx, orgID, req, accountID)
}

func (s *OrganizationService) PreviewSeatPurchase(ctx context.Context, req *pb.PreviewSeatPurchaseRequest) (*pb.PreviewSeatPurchaseResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.PreviewSeatPurchase(ctx, orgID, req)
}

func (s *OrganizationService) PurchaseSeats(ctx context.Context, req *pb.PurchaseSeatsRequest) (*pb.PurchaseSeatsResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	userID, _ := userIDFromContext(ctx)
	return s.billing.PurchaseSeats(ctx, orgID, req, userID)
}

func (s *OrganizationService) SyncOrganizationBilling(
	ctx context.Context,
	req *pb.SyncOrganizationBillingRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.SyncOrganizationBilling(ctx, orgID, req)
}

func (s *OrganizationService) CancelOrganizationSubscription(
	ctx context.Context,
	req *pb.CancelOrganizationSubscriptionRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.CancelOrganizationSubscription(ctx, orgID, req)
}

func (s *OrganizationService) ResumeOrganizationSubscription(
	ctx context.Context,
	req *pb.ResumeOrganizationSubscriptionRequest,
) (*pb.DescribeOrganizationBillingResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.billing.ResumeOrganizationSubscription(ctx, orgID, req)
}

func (s *OrganizationService) AcceptInviteLink(ctx context.Context, req *pb.InviteLink) (*structpb.Struct, error) {
	accountID, err := accountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return organizations.AcceptInviteLink(ctx, s.authorizationService, accountID, req.Token)
}

func (s *OrganizationService) AcceptInvitation(ctx context.Context, req *pb.AcceptInvitationRequest) (*structpb.Struct, error) {
	accountID, err := accountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.invitations.Accept(ctx, accountID, req.Token)
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

func (s *OrganizationService) NextIntegrationSetupStep(ctx context.Context, req *pb.NextIntegrationSetupStepRequest) (*pb.NextIntegrationSetupStepResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.NextIntegrationSetupStep(ctx, s.registry, s.baseURL, s.webhooksBaseURL, orgID, req.IntegrationId, req.Inputs, req.Capabilities)
}

func (s *OrganizationService) PreviousIntegrationSetupStep(ctx context.Context, req *pb.PreviousIntegrationSetupStepRequest) (*pb.PreviousIntegrationSetupStepResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.PreviousIntegrationSetupStep(ctx, s.registry, orgID, req.IntegrationId)
}

func (s *OrganizationService) UpdateIntegrationCapabilities(ctx context.Context, req *pb.UpdateIntegrationCapabilitiesRequest) (*pb.UpdateIntegrationCapabilitiesResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateIntegrationCapabilities(ctx, s.registry, orgID, req.IntegrationId, req.Capabilities)
}

func (s *OrganizationService) UpdateIntegrationProperty(ctx context.Context, req *pb.UpdateIntegrationPropertyRequest) (*pb.UpdateIntegrationPropertyResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateIntegrationProperty(ctx, s.registry, orgID, req.IntegrationId, req.PropertyName, req.Value)
}

func (s *OrganizationService) UpdateIntegrationSecret(ctx context.Context, req *pb.UpdateIntegrationSecretRequest) (*pb.UpdateIntegrationSecretResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return organizations.UpdateIntegrationSecret(ctx, s.registry, orgID, req.IntegrationId, req.SecretName, req.Value)
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

func userIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "user not found")
	}

	userMeta := md.Get("x-user-id")
	if len(userMeta) == 0 || userMeta[0] == "" {
		return "", status.Error(codes.Unauthenticated, "user not found")
	}

	return userMeta[0], nil
}

func (s *OrganizationService) CreateInvitation(ctx context.Context, req *pb.CreateInvitationRequest) (*pb.CreateInvitationResponse, error) {
	inviterID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.invitations.Create(ctx, ctx.Value(authorization.DomainIdContextKey).(string), inviterID, req)
}

func (s *OrganizationService) ListInvitations(ctx context.Context, req *pb.ListInvitationsRequest) (*pb.ListInvitationsResponse, error) {
	return s.invitations.List(ctx, ctx.Value(authorization.DomainIdContextKey).(string))
}

func (s *OrganizationService) ResendInvitation(ctx context.Context, req *pb.ResendInvitationRequest) (*pb.ResendInvitationResponse, error) {
	actorID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.invitations.Resend(ctx, ctx.Value(authorization.DomainIdContextKey).(string), actorID, req.InvitationId)
}

func (s *OrganizationService) CancelInvitation(ctx context.Context, req *pb.CancelInvitationRequest) (*pb.CancelInvitationResponse, error) {
	return s.invitations.Cancel(ctx, ctx.Value(authorization.DomainIdContextKey).(string), req.InvitationId)
}
