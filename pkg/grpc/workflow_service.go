package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/workflows"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WorkflowService struct {
	registry    *registry.Registry
	encryptor   crypto.Encryptor
	authService authorization.Authorization
}

func NewWorkflowService(authService authorization.Authorization, registry *registry.Registry, encryptor crypto.Encryptor) *WorkflowService {
	return &WorkflowService{
		registry:    registry,
		encryptor:   encryptor,
		authService: authService,
	}
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
	return workflows.UpdateWorkflow(ctx, s.encryptor, s.registry, organizationID, req.Id, req.Workflow)
}

func (s *WorkflowService) DeleteWorkflow(ctx context.Context, req *pb.DeleteWorkflowRequest) (*pb.DeleteWorkflowResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return workflows.DeleteWorkflow(ctx, s.registry, uuid.MustParse(organizationID), req.Id)
}

func (s *WorkflowService) ListNodeQueueItems(ctx context.Context, req *pb.ListNodeQueueItemsRequest) (*pb.ListNodeQueueItemsResponse, error) {
	return workflows.ListNodeQueueItems(ctx, s.registry, req.WorkflowId, req.NodeId, req.Limit, req.Before)
}

func (s *WorkflowService) DeleteNodeQueueItem(ctx context.Context, req *pb.DeleteNodeQueueItemRequest) (*pb.DeleteNodeQueueItemResponse, error) {
	return workflows.DeleteNodeQueueItem(ctx, s.registry, req.WorkflowId, req.NodeId, req.ItemId)
}

func (s *WorkflowService) ListNodeExecutions(ctx context.Context, req *pb.ListNodeExecutionsRequest) (*pb.ListNodeExecutionsResponse, error) {
	return workflows.ListNodeExecutions(ctx, s.registry, req.WorkflowId, req.NodeId, req.States, req.Results, req.Limit, req.Before)
}

func (s *WorkflowService) ListNodeEvents(ctx context.Context, req *pb.ListNodeEventsRequest) (*pb.ListNodeEventsResponse, error) {
	workflowID, err := uuid.Parse(req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	return workflows.ListNodeEvents(ctx, s.registry, workflowID, req.NodeId, req.Limit, req.Before)
}

func (s *WorkflowService) EmitNodeEvent(ctx context.Context, req *pb.EmitNodeEventRequest) (*pb.EmitNodeEventResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	workflowID, err := uuid.Parse(req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	if req.Channel == "" {
		return nil, status.Error(codes.InvalidArgument, "channel is required")
	}

	return workflows.EmitNodeEvent(
		ctx,
		uuid.MustParse(organizationID),
		workflowID,
		req.NodeId,
		req.Channel,
		req.Data.AsMap(),
	)
}

func (s *WorkflowService) InvokeNodeExecutionAction(ctx context.Context, req *pb.InvokeNodeExecutionActionRequest) (*pb.InvokeNodeExecutionActionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	workflowID, err := uuid.Parse(req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	return workflows.InvokeNodeExecutionAction(
		ctx,
		s.authService,
		s.registry,
		s.encryptor,
		uuid.MustParse(organizationID),
		workflowID,
		executionID,
		req.ActionName,
		req.Parameters.AsMap(),
	)
}

func (s *WorkflowService) ListWorkflowEvents(ctx context.Context, req *pb.ListWorkflowEventsRequest) (*pb.ListWorkflowEventsResponse, error) {
	workflowID, err := uuid.Parse(req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	return workflows.ListWorkflowEvents(ctx, s.registry, workflowID, req.Limit, req.Before)
}

func (s *WorkflowService) ListEventExecutions(ctx context.Context, req *pb.ListEventExecutionsRequest) (*pb.ListEventExecutionsResponse, error) {
	return workflows.ListEventExecutions(ctx, s.registry, req.WorkflowId, req.EventId)
}

func (s *WorkflowService) ListChildExecutions(ctx context.Context, req *pb.ListChildExecutionsRequest) (*pb.ListChildExecutionsResponse, error) {
	workflowID, err := uuid.Parse(req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	return workflows.ListChildExecutions(ctx, s.registry, workflowID, executionID)
}

func (s *WorkflowService) CancelExecution(ctx context.Context, req *pb.CancelExecutionRequest) (*pb.CancelExecutionResponse, error) {
	workflowID, err := uuid.Parse(req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	return workflows.CancelExecution(ctx, s.authService, organizationID, s.registry, workflowID, executionID)
}
