package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/workflows"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WorkflowService struct {
	registry *registry.Registry
}

func NewWorkflowService(registry *registry.Registry) *WorkflowService {
	return &WorkflowService{registry: registry}
}

func (s *WorkflowService) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return workflows.ListWorkflows(ctx, s.registry, organizationID)
}

func (s *WorkflowService) DescribeWorkflow(ctx context.Context, req *pb.DescribeWorkflowRequest) (*pb.DescribeWorkflowResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return workflows.DescribeWorkflow(ctx, s.registry, organizationID, req.Id)
}

func (s *WorkflowService) CreateWorkflow(ctx context.Context, req *pb.CreateWorkflowRequest) (*pb.CreateWorkflowResponse, error) {
	if req.Workflow == nil {
		return nil, status.Error(codes.InvalidArgument, "workflow is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return workflows.CreateWorkflow(ctx, s.registry, organizationID, req.Workflow)
}

func (s *WorkflowService) UpdateWorkflow(ctx context.Context, req *pb.UpdateWorkflowRequest) (*pb.UpdateWorkflowResponse, error) {
	if req.Workflow == nil {
		return nil, status.Error(codes.InvalidArgument, "workflow is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return workflows.UpdateWorkflow(ctx, s.registry, organizationID, req.Id, req.Workflow)
}

func (s *WorkflowService) DeleteWorkflow(ctx context.Context, req *pb.DeleteWorkflowRequest) (*pb.DeleteWorkflowResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return workflows.DeleteWorkflow(ctx, s.registry, organizationID, req.Id)
}

func (s *WorkflowService) ListNodeExecutions(ctx context.Context, req *pb.ListNodeExecutionsRequest) (*pb.ListNodeExecutionsResponse, error) {
	return workflows.ListNodeExecutions(ctx, s.registry, req.WorkflowId, req.NodeId, req.States, req.Results, req.Limit, req.Before)
}

func (s *WorkflowService) InvokeNodeExecutionAction(ctx context.Context, req *pb.InvokeNodeExecutionActionRequest) (*pb.InvokeNodeExecutionActionResponse, error) {
	return workflows.InvokeNodeExecutionAction(ctx, s.registry, req.ExecutionId, req.ActionName, req.Parameters.AsMap())
}

func (s *WorkflowService) ListWorkflowEvents(ctx context.Context, req *pb.ListWorkflowEventsRequest) (*pb.ListWorkflowEventsResponse, error) {
	return workflows.ListWorkflowEvents(ctx, s.registry, req.WorkflowId, req.Limit, req.Before)
}

func (s *WorkflowService) ListEventExecutions(ctx context.Context, req *pb.ListEventExecutionsRequest) (*pb.ListEventExecutionsResponse, error) {
	return workflows.ListEventExecutions(ctx, s.registry, req.WorkflowId, req.EventId)
}
