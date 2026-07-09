package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CanvasService struct {
	registry       *registry.Registry
	encryptor      crypto.Encryptor
	authService    authorization.Authorization
	gitProvider    git.Provider
	webhookBaseURL string
	usageService   usage.Service
}

func NewCanvasService(
	authService authorization.Authorization,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	gitProvider git.Provider,
	webhookBaseURL string,
	usageService usage.Service,
) *CanvasService {
	return &CanvasService{
		registry:       registry,
		encryptor:      encryptor,
		authService:    authService,
		gitProvider:    gitProvider,
		webhookBaseURL: webhookBaseURL,
		usageService:   usageService,
	}
}

func (s *CanvasService) ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return canvases.ListCanvases(ctx, s.registry, organizationID, userID)
}

func (s *CanvasService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.DescribeCanvas(ctx, s.registry, organizationID, req.Id)
}

func (s *CanvasService) UpdateCanvas(ctx context.Context, req *pb.UpdateCanvasRequest) (*pb.UpdateCanvasResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.UpdateCanvas(ctx, organizationID, req.Id, req.Name, req.Description)
}

func (s *CanvasService) UpdateCanvasPreference(
	ctx context.Context,
	req *pb.UpdateCanvasPreferenceRequest,
) (*pb.UpdateCanvasPreferenceResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return canvases.UpdateCanvasPreference(ctx, organizationID, userID, req)
}

func (s *CanvasService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	return canvases.CreateCanvas(
		ctx,
		s.registry,
		s.encryptor,
		s.authService,
		s.gitProvider,
		s.webhookBaseURL,
		uuid.MustParse(organizationID),
		req.GetName(),
		req.GetDescription(),
		s.usageService,
	)
}

func (s *CanvasService) ListCanvasVersions(ctx context.Context, req *pb.ListCanvasVersionsRequest) (*pb.ListCanvasVersionsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.ListCanvasVersionsPaginated(ctx, organizationID, req.CanvasId, req.Limit, req.Before)
}

func (s *CanvasService) DescribeCanvasVersion(ctx context.Context, req *pb.DescribeCanvasVersionRequest) (*pb.DescribeCanvasVersionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.DescribeCanvasVersion(ctx, organizationID, req.CanvasId, req.VersionId)
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

func (s *CanvasService) ReemitTriggerEvent(ctx context.Context, req *pb.ReemitTriggerEventRequest) (*pb.ReemitTriggerEventResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	return canvases.ReemitTriggerEvent(
		ctx,
		uuid.MustParse(organizationID),
		canvasID,
		req.NodeId,
		eventID,
	)
}

func (s *CanvasService) InvokeNodeExecutionHook(ctx context.Context, req *pb.InvokeNodeExecutionHookRequest) (*pb.InvokeNodeExecutionHookResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	return canvases.InvokeNodeExecutionHook(
		ctx,
		s.authService,
		s.encryptor,
		s.registry,
		uuid.MustParse(organizationID),
		canvasID,
		executionID,
		req.HookName,
		req.Parameters.AsMap(),
	)
}

func (s *CanvasService) InvokeNodeTriggerHook(ctx context.Context, req *pb.InvokeNodeTriggerHookRequest) (*pb.InvokeNodeTriggerHookResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	if req.HookName == "" {
		return nil, status.Error(codes.InvalidArgument, "hook_name is required")
	}

	return canvases.InvokeNodeTriggerHook(
		ctx,
		s.authService,
		s.encryptor,
		s.registry,
		uuid.MustParse(organizationID),
		canvasID,
		req.NodeId,
		req.HookName,
		req.Parameters.AsMap(),
		s.webhookBaseURL,
	)
}

func (s *CanvasService) ListRuns(ctx context.Context, req *pb.ListRunsRequest) (*pb.ListRunsResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	return canvases.ListRuns(ctx, s.registry, canvasID, req.Limit, req.Before, req.States, req.Results)
}

func (s *CanvasService) DescribeRun(ctx context.Context, req *pb.DescribeRunRequest) (*pb.DescribeRunResponse, error) {
	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas id")
	}

	return canvases.DescribeRun(ctx, s.registry, canvasID, req.RunId)
}

func (s *CanvasService) ListCanvasMemories(ctx context.Context, req *pb.ListCanvasMemoriesRequest) (*pb.ListCanvasMemoriesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.ListCanvasMemories(ctx, s.registry, organizationID, req.CanvasId)
}

func (s *CanvasService) DeleteCanvasMemory(ctx context.Context, req *pb.DeleteCanvasMemoryRequest) (*pb.DeleteCanvasMemoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.DeleteCanvasMemory(ctx, s.registry, organizationID, req.CanvasId, req.MemoryId)
}

func (s *CanvasService) CreateCanvasMemoryNamespace(ctx context.Context, req *pb.CreateCanvasMemoryNamespaceRequest) (*pb.CreateCanvasMemoryNamespaceResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.CreateCanvasMemoryNamespace(ctx, s.registry, organizationID, req.CanvasId, req.Namespace, req.Entries)
}

func (s *CanvasService) UpdateCanvasMemoryNamespace(ctx context.Context, req *pb.UpdateCanvasMemoryNamespaceRequest) (*pb.UpdateCanvasMemoryNamespaceResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.UpdateCanvasMemoryNamespace(ctx, s.registry, organizationID, req.CanvasId, req.Namespace, req.NewNamespace, req.Entries)
}

func (s *CanvasService) ListEventExecutions(ctx context.Context, req *pb.ListEventExecutionsRequest) (*pb.ListEventExecutionsResponse, error) {
	return canvases.ListEventExecutions(ctx, s.registry, req.CanvasId, req.EventId)
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

func (s *CanvasService) GetCanvasRepository(ctx context.Context, req *pb.GetCanvasRepositoryRequest) (*pb.GetCanvasRepositoryResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.GetCanvasRepository(ctx, s.gitProvider, organizationID, req.CanvasId)
}

func (s *CanvasService) ListCanvasRepositoryFiles(ctx context.Context, req *pb.ListCanvasRepositoryFilesRequest) (*pb.ListCanvasRepositoryFilesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.ListCanvasRepositoryFiles(ctx, s.gitProvider, organizationID, req.CanvasId)
}

func (s *CanvasService) PutCanvasStaging(ctx context.Context, req *pb.PutCanvasStagingRequest) (*pb.PutCanvasStagingResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	state, err := canvases.PutCanvasStaging(ctx, organizationID, req.CanvasId, req.Operations)
	if err != nil {
		return nil, err
	}
	return &pb.PutCanvasStagingResponse{Staging: state}, nil
}

func (s *CanvasService) GetCanvasStaging(ctx context.Context, req *pb.GetCanvasStagingRequest) (*pb.GetCanvasStagingResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	state, err := canvases.GetCanvasStaging(ctx, organizationID, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return &pb.GetCanvasStagingResponse{Staging: state}, nil
}

func (s *CanvasService) DeleteCanvasStaging(ctx context.Context, req *pb.DeleteCanvasStagingRequest) (*pb.DeleteCanvasStagingResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	state, err := canvases.DeleteCanvasStaging(ctx, organizationID, req.CanvasId, req.Paths)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteCanvasStagingResponse{Staging: state}, nil
}

func (s *CanvasService) CommitCanvasStaging(ctx context.Context, req *pb.CommitCanvasStagingRequest) (*pb.CommitCanvasStagingResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return canvases.CommitCanvasStaging(
		ctx,
		s.gitProvider,
		s.usageService,
		s.encryptor,
		s.registry,
		organizationID,
		req.CanvasId,
		req.CommitMessage,
		s.webhookBaseURL,
		s.authService,
	)
}
