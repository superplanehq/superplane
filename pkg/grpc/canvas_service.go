package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CanvasService struct {
	registry       *registry.Registry
	encryptor      crypto.Encryptor
	authService    authorization.Authorization
	webhookBaseURL string
}

func NewCanvasService(authService authorization.Authorization, registry *registry.Registry, encryptor crypto.Encryptor, webhookBaseURL string) *CanvasService {
	return &CanvasService{
		registry:       registry,
		encryptor:      encryptor,
		authService:    authService,
		webhookBaseURL: webhookBaseURL,
	}
}

func (s *CanvasService) ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.ListCanvases(ctx, s.registry, organizationID, req.IncludeTemplates)
}

func (s *CanvasService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.DescribeCanvas(ctx, s.registry, organizationID, req.Id)
}

func (s *CanvasService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	if req.Canvas == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.CreateCanvas(ctx, s.registry, organizationID, req.Canvas)
}

func (s *CanvasService) UpdateCanvas(ctx context.Context, req *pb.UpdateCanvasRequest) (*pb.UpdateCanvasResponse, error) {
	if req.Canvas == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.UpdateCanvasWithAutoLayout(
		ctx,
		s.encryptor,
		s.registry,
		organizationID,
		req.Id,
		req.Canvas,
		req.AutoLayout,
		s.webhookBaseURL,
	)
}

func (s *CanvasService) DeleteCanvas(ctx context.Context, req *pb.DeleteCanvasRequest) (*pb.DeleteCanvasResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.DeleteCanvas(ctx, s.registry, uuid.MustParse(organizationID), req.Id)
}

func (s *CanvasService) ListNodeQueueItems(ctx context.Context, req *pb.ListNodeQueueItemsRequest) (*pb.ListNodeQueueItemsResponse, error) {
	return canvases.ListNodeQueueItems(ctx, s.registry, req.CanvasId, req.NodeId, req.Limit, req.Before)
}

func (s *CanvasService) DeleteNodeQueueItem(ctx context.Context, req *pb.DeleteNodeQueueItemRequest) (*pb.DeleteNodeQueueItemResponse, error) {
	return canvases.DeleteNodeQueueItem(ctx, s.registry, req.CanvasId, req.NodeId, req.ItemId)
}

func (s *CanvasService) UpdateNodePause(ctx context.Context, req *pb.UpdateNodePauseRequest) (*pb.UpdateNodePauseResponse, error) {
	return canvases.UpdateNodePause(ctx, s.registry, req.CanvasId, req.NodeId, req.Paused)
}

func (s *CanvasService) ListNodeExecutions(ctx context.Context, req *pb.ListNodeExecutionsRequest) (*pb.ListNodeExecutionsResponse, error) {
	return canvases.ListNodeExecutions(ctx, s.registry, req.CanvasId, req.NodeId, req.States, req.Results, req.Limit, req.Before)
}

func (s *CanvasService) ListNodeEvents(ctx context.Context, req *pb.ListNodeEventsRequest) (*pb.ListNodeEventsResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	return canvases.ListNodeEvents(ctx, s.registry, canvasID, req.NodeId, req.Limit, req.Before)
}

func (s *CanvasService) EmitNodeEvent(ctx context.Context, req *pb.EmitNodeEventRequest) (*pb.EmitNodeEventResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	if req.Channel == "" {
		return nil, status.Error(codes.InvalidArgument, "channel is required")
	}

	return canvases.EmitNodeEvent(
		ctx,
		uuid.MustParse(organizationID),
		canvasID,
		req.NodeId,
		req.Channel,
		req.Data.AsMap(),
	)
}

func (s *CanvasService) InvokeNodeExecutionAction(ctx context.Context, req *pb.InvokeNodeExecutionActionRequest) (*pb.InvokeNodeExecutionActionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	return canvases.InvokeNodeExecutionAction(
		ctx,
		s.authService,
		s.encryptor,
		s.registry,
		uuid.MustParse(organizationID),
		canvasID,
		executionID,
		req.ActionName,
		req.Parameters.AsMap(),
	)
}

func (s *CanvasService) InvokeNodeTriggerAction(ctx context.Context, req *pb.InvokeNodeTriggerActionRequest) (*pb.InvokeNodeTriggerActionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	if req.ActionName == "" {
		return nil, status.Error(codes.InvalidArgument, "action_name is required")
	}

	return canvases.InvokeNodeTriggerAction(
		ctx,
		s.authService,
		s.encryptor,
		s.registry,
		uuid.MustParse(organizationID),
		canvasID,
		req.NodeId,
		req.ActionName,
		req.Parameters.AsMap(),
		s.webhookBaseURL,
	)
}

func (s *CanvasService) ListCanvasEvents(ctx context.Context, req *pb.ListCanvasEventsRequest) (*pb.ListCanvasEventsResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	return canvases.ListCanvasEvents(ctx, s.registry, canvasID, req.Limit, req.Before)
}

func (s *CanvasService) ListEventExecutions(ctx context.Context, req *pb.ListEventExecutionsRequest) (*pb.ListEventExecutionsResponse, error) {
	return canvases.ListEventExecutions(ctx, s.registry, req.CanvasId, req.EventId)
}

func (s *CanvasService) ListChildExecutions(ctx context.Context, req *pb.ListChildExecutionsRequest) (*pb.ListChildExecutionsResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	return canvases.ListChildExecutions(ctx, s.registry, canvasID, executionID)
}

func (s *CanvasService) CancelExecution(ctx context.Context, req *pb.CancelExecutionRequest) (*pb.CancelExecutionResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	return canvases.CancelExecution(ctx, s.authService, s.encryptor, organizationID, s.registry, canvasID, executionID)
}

func (s *CanvasService) ResolveExecutionErrors(ctx context.Context, req *pb.ResolveExecutionErrorsRequest) (*pb.ResolveExecutionErrorsResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionIDs := make([]uuid.UUID, 0, len(req.ExecutionIds))
	for _, executionID := range req.ExecutionIds {
		parsedID, err := uuid.Parse(executionID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
		}
		executionIDs = append(executionIDs, parsedID)
	}

	return canvases.ResolveExecutionErrors(ctx, canvasID, executionIDs)
}

func (s *CanvasService) SendAiMessage(ctx context.Context, req *pb.SendAiMessageRequest) (*pb.SendAiMessageResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.SendAiMessage(ctx, s.registry, s.encryptor, organizationID, req)
}
