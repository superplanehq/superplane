package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	actions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

type FactoryService struct {
	pb.UnimplementedFactoriesServer

	registry *registry.Registry
	// An intake owns a canvas, so creating one needs everything canvas
	// creation needs.
	intakeDeps actions.IntakeDependencies
}

func NewFactoryService(
	reg *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	gitProvider git.Provider,
	webhookBaseURL string,
	usageService usage.Service,
) *FactoryService {
	return &FactoryService{
		registry: reg,
		intakeDeps: actions.IntakeDependencies{
			Registry:       reg,
			Encryptor:      encryptor,
			AuthService:    authService,
			GitProvider:    gitProvider,
			WebhookBaseURL: webhookBaseURL,
			UsageService:   usageService,
		},
	}
}

func (s *FactoryService) ListFactories(ctx context.Context, req *pb.ListFactoriesRequest) (*pb.ListFactoriesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactories(ctx, organizationID)
}

func (s *FactoryService) CreateFactory(ctx context.Context, req *pb.CreateFactoryRequest) (*pb.CreateFactoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateFactory(ctx, organizationID, req)
}

func (s *FactoryService) DescribeFactory(ctx context.Context, req *pb.DescribeFactoryRequest) (*pb.DescribeFactoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DescribeFactory(ctx, organizationID, req.GetId())
}

func (s *FactoryService) UpdateFactory(ctx context.Context, req *pb.UpdateFactoryRequest) (*pb.UpdateFactoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactory(ctx, organizationID, req)
}

func (s *FactoryService) UpdateFactoryOnboarding(ctx context.Context, req *pb.UpdateFactoryOnboardingRequest) (*pb.UpdateFactoryOnboardingResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryOnboarding(ctx, organizationID, req)
}

func (s *FactoryService) UpdateFactoryRepository(ctx context.Context, req *pb.UpdateFactoryRepositoryRequest) (*pb.UpdateFactoryRepositoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryRepository(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) DeleteFactory(ctx context.Context, req *pb.DeleteFactoryRequest) (*pb.DeleteFactoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DeleteFactory(ctx, organizationID, req.GetId())
}

func (s *FactoryService) CreateFactoryLine(ctx context.Context, req *pb.CreateFactoryLineRequest) (*pb.CreateFactoryLineResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateFactoryLine(ctx, organizationID, req)
}

func (s *FactoryService) UpdateFactoryLine(ctx context.Context, req *pb.UpdateFactoryLineRequest) (*pb.UpdateFactoryLineResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryLine(ctx, organizationID, req)
}

func (s *FactoryService) ListFactoryApps(ctx context.Context, req *pb.ListFactoryAppsRequest) (*pb.ListFactoryAppsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactoryApps(ctx, organizationID, req)
}

func (s *FactoryService) MaterializeFactoryAppTemplate(ctx context.Context, req *pb.MaterializeFactoryAppTemplateRequest) (*pb.MaterializeFactoryAppTemplateResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.MaterializeFactoryAppTemplate(ctx, organizationID, req)
}

func (s *FactoryService) MaterializeFactoryAppDefaults(ctx context.Context, req *pb.MaterializeFactoryAppDefaultsRequest) (*pb.MaterializeFactoryAppDefaultsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.MaterializeFactoryAppDefaults(ctx, organizationID, req)
}

func (s *FactoryService) ListFactoryIntakes(ctx context.Context, req *pb.ListFactoryIntakesRequest) (*pb.ListFactoryIntakesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactoryIntakes(ctx, organizationID, req)
}

func (s *FactoryService) CreateFactoryIntake(ctx context.Context, req *pb.CreateFactoryIntakeRequest) (*pb.CreateFactoryIntakeResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateFactoryIntake(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) UpdateFactoryIntake(ctx context.Context, req *pb.UpdateFactoryIntakeRequest) (*pb.UpdateFactoryIntakeResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryIntake(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) DeleteFactoryIntake(ctx context.Context, req *pb.DeleteFactoryIntakeRequest) (*pb.DeleteFactoryIntakeResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DeleteFactoryIntake(ctx, organizationID, req)
}

func (s *FactoryService) ListFactoryIntakeRuns(ctx context.Context, req *pb.ListFactoryIntakeRunsRequest) (*pb.ListFactoryIntakeRunsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactoryIntakeRuns(ctx, organizationID, req)
}

func (s *FactoryService) ListFactoryPRFeedbackHandlers(ctx context.Context, req *pb.ListFactoryPRFeedbackHandlersRequest) (*pb.ListFactoryPRFeedbackHandlersResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactoryPRFeedbackHandlers(ctx, organizationID, req)
}

func (s *FactoryService) CreateFactoryPRFeedbackHandler(ctx context.Context, req *pb.CreateFactoryPRFeedbackHandlerRequest) (*pb.CreateFactoryPRFeedbackHandlerResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateFactoryPRFeedbackHandler(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) UpdateFactoryPRFeedbackHandler(ctx context.Context, req *pb.UpdateFactoryPRFeedbackHandlerRequest) (*pb.UpdateFactoryPRFeedbackHandlerResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryPRFeedbackHandler(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) DeleteFactoryPRFeedbackHandler(ctx context.Context, req *pb.DeleteFactoryPRFeedbackHandlerRequest) (*pb.DeleteFactoryPRFeedbackHandlerResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DeleteFactoryPRFeedbackHandler(ctx, organizationID, req)
}

func (s *FactoryService) ListFactoryPullRequests(ctx context.Context, req *pb.ListFactoryPullRequestsRequest) (*pb.ListFactoryPullRequestsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactoryPullRequests(ctx, organizationID, req)
}

func (s *FactoryService) DescribeFactoryPullRequest(ctx context.Context, req *pb.DescribeFactoryPullRequestRequest) (*pb.DescribeFactoryPullRequestResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DescribeFactoryPullRequest(ctx, organizationID, req)
}

func (s *FactoryService) CreateFactoryPullRequest(ctx context.Context, req *pb.CreateFactoryPullRequestRequest) (*pb.CreateFactoryPullRequestResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateFactoryPullRequest(ctx, organizationID, req)
}

func (s *FactoryService) UpdateFactoryPullRequest(ctx context.Context, req *pb.UpdateFactoryPullRequestRequest) (*pb.UpdateFactoryPullRequestResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryPullRequest(ctx, organizationID, req)
}

func (s *FactoryService) SearchFactoryIntakeItems(ctx context.Context, req *pb.SearchFactoryIntakeItemsRequest) (*pb.SearchFactoryIntakeItemsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.SearchFactoryIntakeItems(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) ImportFactoryIntakeItem(ctx context.Context, req *pb.ImportFactoryIntakeItemRequest) (*pb.ImportFactoryIntakeItemResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ImportFactoryIntakeItem(ctx, s.intakeDeps, organizationID, req)
}

func (s *FactoryService) ListWorkOrders(ctx context.Context, req *pb.ListWorkOrdersRequest) (*pb.ListWorkOrdersResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListWorkOrders(ctx, organizationID, req)
}

func (s *FactoryService) CreateWorkOrder(ctx context.Context, req *pb.CreateWorkOrderRequest) (*pb.CreateWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) DescribeWorkOrder(ctx context.Context, req *pb.DescribeWorkOrderRequest) (*pb.DescribeWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DescribeWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) ListWorkOrderEvents(ctx context.Context, req *pb.ListWorkOrderEventsRequest) (*pb.ListWorkOrderEventsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListWorkOrderEvents(ctx, organizationID, req)
}

func (s *FactoryService) UpdateWorkOrderAssignees(
	ctx context.Context,
	req *pb.UpdateWorkOrderAssigneesRequest,
) (*pb.UpdateWorkOrderAssigneesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateWorkOrderAssignees(ctx, organizationID, req)
}

func (s *FactoryService) UpdateWorkOrder(
	ctx context.Context,
	req *pb.UpdateWorkOrderRequest,
) (*pb.UpdateWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) DispatchWorkOrder(ctx context.Context, req *pb.DispatchWorkOrderRequest) (*pb.DispatchWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DispatchWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) CloseWorkOrder(ctx context.Context, req *pb.CloseWorkOrderRequest) (*pb.CloseWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CloseWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) UpdateWorkOrderStatus(ctx context.Context, req *pb.UpdateWorkOrderStatusRequest) (*pb.UpdateWorkOrderStatusResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateWorkOrderStatus(ctx, organizationID, req)
}

func (s *FactoryService) AddWorkOrderComment(ctx context.Context, req *pb.AddWorkOrderCommentRequest) (*pb.AddWorkOrderCommentResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.AddWorkOrderComment(ctx, organizationID, req)
}

func (s *FactoryService) ListWorkOrderArtifacts(ctx context.Context, req *pb.ListWorkOrderArtifactsRequest) (*pb.ListWorkOrderArtifactsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListWorkOrderArtifacts(ctx, organizationID, req)
}

func (s *FactoryService) CreateWorkOrderArtifact(ctx context.Context, req *pb.CreateWorkOrderArtifactRequest) (*pb.CreateWorkOrderArtifactResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreateWorkOrderArtifact(ctx, organizationID, req)
}

func (s *FactoryService) DescribeFactoryVelocity(ctx context.Context, req *pb.DescribeFactoryVelocityRequest) (*pb.DescribeFactoryVelocityResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DescribeFactoryVelocity(ctx, organizationID, req)
}

func (s *FactoryService) SyncFactoryVelocity(ctx context.Context, req *pb.SyncFactoryVelocityRequest) (*pb.SyncFactoryVelocityResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.SyncFactoryVelocity(ctx, organizationID, req)
}

func (s *FactoryService) ListWorkOrderChecks(ctx context.Context, req *pb.ListWorkOrderChecksRequest) (*pb.ListWorkOrderChecksResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListWorkOrderChecks(ctx, organizationID, req)
}

func (s *FactoryService) DescribeFactoryUsage(ctx context.Context, req *pb.DescribeFactoryUsageRequest) (*pb.DescribeFactoryUsageResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DescribeFactoryUsage(ctx, organizationID, req)
}

func (s *FactoryService) ListFactoryLLMModels(ctx context.Context, req *pb.ListFactoryLLMModelsRequest) (*pb.ListFactoryLLMModelsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.ListFactoryLLMModels(ctx, organizationID, req)
}

func (s *FactoryService) UpdateFactoryLLMModels(ctx context.Context, req *pb.UpdateFactoryLLMModelsRequest) (*pb.UpdateFactoryLLMModelsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdateFactoryLLMModels(ctx, organizationID, req)
}

func (s *FactoryService) StartPlanningSession(ctx context.Context, req *pb.StartPlanningSessionRequest) (*pb.StartPlanningSessionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.StartPlanningSession(ctx, organizationID, req)
}

func (s *FactoryService) DescribePlanningSession(ctx context.Context, req *pb.DescribePlanningSessionRequest) (*pb.DescribePlanningSessionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.DescribePlanningSession(ctx, organizationID, req)
}

func (s *FactoryService) HeartbeatPlanningSession(ctx context.Context, req *pb.HeartbeatPlanningSessionRequest) (*pb.HeartbeatPlanningSessionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.HeartbeatPlanningSession(ctx, organizationID, req)
}

func (s *FactoryService) EndPlanningSession(ctx context.Context, req *pb.EndPlanningSessionRequest) (*pb.EndPlanningSessionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.EndPlanningSession(ctx, organizationID, req)
}

func (s *FactoryService) SendPlanningSessionMessage(ctx context.Context, req *pb.SendPlanningSessionMessageRequest) (*pb.SendPlanningSessionMessageResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.SendPlanningSessionMessage(ctx, organizationID, req)
}

func (s *FactoryService) UpdatePlanningSessionDraft(ctx context.Context, req *pb.UpdatePlanningSessionDraftRequest) (*pb.UpdatePlanningSessionDraftResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.UpdatePlanningSessionDraft(ctx, organizationID, req)
}

func (s *FactoryService) CreatePlanningSessionWorkOrder(ctx context.Context, req *pb.CreatePlanningSessionWorkOrderRequest) (*pb.CreatePlanningSessionWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.CreatePlanningSessionWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) SkipPlanningSessionDraft(ctx context.Context, req *pb.SkipPlanningSessionDraftRequest) (*pb.SkipPlanningSessionDraftResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.SkipPlanningSessionDraft(ctx, organizationID, req)
}
