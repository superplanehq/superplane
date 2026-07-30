package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	factoriesActions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

type FactoryService struct {
	pb.UnimplementedFactoriesServer
}

func NewFactoryService() *FactoryService {
	return &FactoryService{}
}

func (s *FactoryService) ListFactories(ctx context.Context, req *pb.ListFactoriesRequest) (*pb.ListFactoriesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListFactories(ctx, organizationID)
}

func (s *FactoryService) CreateFactory(ctx context.Context, req *pb.CreateFactoryRequest) (*pb.CreateFactoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.CreateFactory(ctx, organizationID, req)
}

func (s *FactoryService) DescribeFactory(ctx context.Context, req *pb.DescribeFactoryRequest) (*pb.DescribeFactoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.DescribeFactory(ctx, organizationID, req.GetId())
}

func (s *FactoryService) ListSources(ctx context.Context, req *pb.ListSourcesRequest) (*pb.ListSourcesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListSources(ctx, organizationID, req)
}

func (s *FactoryService) AddSource(ctx context.Context, req *pb.AddSourceRequest) (*pb.AddSourceResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.AddSource(ctx, organizationID, req)
}

func (s *FactoryService) ListWorkOrders(ctx context.Context, req *pb.ListWorkOrdersRequest) (*pb.ListWorkOrdersResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListWorkOrders(ctx, organizationID, req)
}

func (s *FactoryService) CreateWorkOrder(ctx context.Context, req *pb.CreateWorkOrderRequest) (*pb.CreateWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.CreateWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) DescribeWorkOrder(ctx context.Context, req *pb.DescribeWorkOrderRequest) (*pb.DescribeWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.DescribeWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) ListWorkOrderEvents(ctx context.Context, req *pb.ListWorkOrderEventsRequest) (*pb.ListWorkOrderEventsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListWorkOrderEvents(ctx, organizationID, req)
}

func (s *FactoryService) AssignWorkOrder(ctx context.Context, req *pb.AssignWorkOrderRequest) (*pb.AssignWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.AssignWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) CloseWorkOrder(ctx context.Context, req *pb.CloseWorkOrderRequest) (*pb.CloseWorkOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.CloseWorkOrder(ctx, organizationID, req)
}

func (s *FactoryService) ListAgentAssignmentsForOrder(
	ctx context.Context,
	req *pb.ListAgentAssignmentsForOrderRequest,
) (*pb.ListAgentAssignmentsForOrderResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListAgentAssignmentsForOrder(ctx, organizationID, req)
}

func (s *FactoryService) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.CreateAgentResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.CreateAgent(ctx, organizationID, req)
}

func (s *FactoryService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListAgents(ctx, organizationID, req)
}

func (s *FactoryService) DescribeAgent(ctx context.Context, req *pb.DescribeAgentRequest) (*pb.DescribeAgentResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.DescribeAgent(ctx, organizationID, req)
}

func (s *FactoryService) ListAgentAssignments(ctx context.Context, req *pb.ListAgentAssignmentsRequest) (*pb.ListAgentAssignmentsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.ListAgentAssignments(ctx, organizationID, req)
}

func (s *FactoryService) CreateAgentAssignment(ctx context.Context, req *pb.CreateAgentAssignmentRequest) (*pb.CreateAgentAssignmentResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.CreateAgentAssignment(ctx, organizationID, req)
}

func (s *FactoryService) DescribeAgentAssignment(ctx context.Context, req *pb.DescribeAgentAssignmentRequest) (*pb.DescribeAgentAssignmentResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return factoriesActions.DescribeAgentAssignment(ctx, organizationID, req)
}
