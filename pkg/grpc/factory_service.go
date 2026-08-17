package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	actions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
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

func (s *FactoryService) AddWorkOrderCommentReaction(
	ctx context.Context,
	req *pb.AddWorkOrderCommentReactionRequest,
) (*pb.AddWorkOrderCommentReactionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.AddWorkOrderCommentReaction(ctx, organizationID, req)
}

func (s *FactoryService) RemoveWorkOrderCommentReaction(
	ctx context.Context,
	req *pb.RemoveWorkOrderCommentReactionRequest,
) (*pb.RemoveWorkOrderCommentReactionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return actions.RemoveWorkOrderCommentReaction(ctx, organizationID, req)
}
