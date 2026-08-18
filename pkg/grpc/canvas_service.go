package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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

func (s *CanvasService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	var factoryID *uuid.UUID
	if req.FactoryId != nil {
		factoryIDStr := strings.TrimSpace(req.GetFactoryId())
		if factoryIDStr != "" {
			id, err := uuid.Parse(factoryIDStr)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid factory_id")
			}
			factoryID = &id
		}
	}

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
		factoryID,
		s.usageService,
	)
}

/*
 * We always fetch the canvas first, to ensure the canvas being operated on belongs to the authenticated org.
 */
func (s *CanvasService) findCanvas(ctx context.Context, db *gorm.DB, canvasID string) (*models.Canvas, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	id, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	return canvas, nil
}

func (s *CanvasService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.Id)
	if err != nil {
		return nil, err
	}
	return canvases.DescribeCanvas(ctx, db, canvas)
}

func (s *CanvasService) UpdateCanvas(ctx context.Context, req *pb.UpdateCanvasRequest) (*pb.UpdateCanvasResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.Id)
	if err != nil {
		return nil, err
	}
	return canvases.UpdateCanvas(ctx, db, canvas, req.Name, req.Description, req.DismissAgentSuggestionId)
}

func (s *CanvasService) UpdateCanvasPreference(
	ctx context.Context,
	req *pb.UpdateCanvasPreferenceRequest,
) (*pb.UpdateCanvasPreferenceResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return canvases.UpdateCanvasPreference(ctx, db, canvas, userID, req)
}

func (s *CanvasService) ListCanvasVersions(ctx context.Context, req *pb.ListCanvasVersionsRequest) (*pb.ListCanvasVersionsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.ListCanvasVersionsPaginated(ctx, db, canvas, req.Limit, req.Before)
}

func (s *CanvasService) DescribeCanvasVersion(ctx context.Context, req *pb.DescribeCanvasVersionRequest) (*pb.DescribeCanvasVersionResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.DescribeCanvasVersion(ctx, db, canvas, req.VersionId)
}

func (s *CanvasService) DeleteCanvas(ctx context.Context, req *pb.DeleteCanvasRequest) (*pb.DeleteCanvasResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.Id)
	if err != nil {
		return nil, err
	}
	return canvases.DeleteCanvas(ctx, db, canvas)
}

func (s *CanvasService) ListNodeQueueItems(ctx context.Context, req *pb.ListNodeQueueItemsRequest) (*pb.ListNodeQueueItemsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.ListNodeQueueItems(ctx, db, canvas, req.NodeId, req.Limit, req.Before)
}

func (s *CanvasService) DeleteNodeQueueItem(ctx context.Context, req *pb.DeleteNodeQueueItemRequest) (*pb.DeleteNodeQueueItemResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.DeleteNodeQueueItem(ctx, db, canvas, req.NodeId, req.ItemId)
}

func (s *CanvasService) ListNodeExecutions(ctx context.Context, req *pb.ListNodeExecutionsRequest) (*pb.ListNodeExecutionsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.ListNodeExecutions(ctx, db, canvas, req.NodeId, req.States, req.Results, req.Limit, req.Before)
}

func (s *CanvasService) ListNodeEvents(ctx context.Context, req *pb.ListNodeEventsRequest) (*pb.ListNodeEventsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	return canvases.ListNodeEvents(ctx, db, canvas, req.NodeId, req.Limit, req.Before)
}

func (s *CanvasService) ReemitTriggerEvent(ctx context.Context, req *pb.ReemitTriggerEventRequest) (*pb.ReemitTriggerEventResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event_id")
	}

	return canvases.ReemitTriggerEvent(ctx, db, canvas, req.NodeId, eventID)
}

func (s *CanvasService) InvokeNodeExecutionHook(ctx context.Context, req *pb.InvokeNodeExecutionHookRequest) (*pb.InvokeNodeExecutionHookResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
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
		db,
		canvas,
		executionID,
		req.HookName,
		req.Parameters.AsMap(),
	)
}

func (s *CanvasService) InvokeNodeTriggerHook(ctx context.Context, req *pb.InvokeNodeTriggerHookRequest) (*pb.InvokeNodeTriggerHookResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
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
		db,
		canvas,
		req.NodeId,
		req.HookName,
		req.Parameters.AsMap(),
		s.webhookBaseURL,
	)
}

func (s *CanvasService) ListRuns(ctx context.Context, req *pb.ListRunsRequest) (*pb.ListRunsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	return canvases.ListRuns(ctx, db, canvas, req.Limit, req.Before, req.States, req.Results, req.TriggeredByUserId)
}

func (s *CanvasService) DescribeRun(ctx context.Context, req *pb.DescribeRunRequest) (*pb.DescribeRunResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	return canvases.DescribeRun(ctx, db, canvas, req.RunId)
}

func (s *CanvasService) CancelRun(ctx context.Context, req *pb.CancelRunRequest) (*pb.CancelRunResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	runID, err := uuid.Parse(req.RunId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid run id")
	}

	return canvases.CancelRun(ctx, db, canvas, runID)
}

func (s *CanvasService) ListCanvasMemories(ctx context.Context, req *pb.ListCanvasMemoriesRequest) (*pb.ListCanvasMemoriesResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.ListCanvasMemories(ctx, db, canvas)
}

func (s *CanvasService) DeleteCanvasMemory(ctx context.Context, req *pb.DeleteCanvasMemoryRequest) (*pb.DeleteCanvasMemoryResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.DeleteCanvasMemory(ctx, db, canvas, req.MemoryId)
}

func (s *CanvasService) CreateCanvasMemoryNamespace(ctx context.Context, req *pb.CreateCanvasMemoryNamespaceRequest) (*pb.CreateCanvasMemoryNamespaceResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.CreateCanvasMemoryNamespace(ctx, db, canvas, req.Namespace, req.Entries)
}

func (s *CanvasService) UpdateCanvasMemoryNamespace(ctx context.Context, req *pb.UpdateCanvasMemoryNamespaceRequest) (*pb.UpdateCanvasMemoryNamespaceResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.UpdateCanvasMemoryNamespace(ctx, db, canvas, req.Namespace, req.NewNamespace, req.Entries)
}

func (s *CanvasService) ListEventExecutions(ctx context.Context, req *pb.ListEventExecutionsRequest) (*pb.ListEventExecutionsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.ListEventExecutions(ctx, db, canvas, req.EventId)
}

func (s *CanvasService) CancelExecution(ctx context.Context, req *pb.CancelExecutionRequest) (*pb.CancelExecutionResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	executionID, err := uuid.Parse(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	return canvases.CancelExecution(ctx, s.authService, s.encryptor, db, canvas, executionID)
}

func (s *CanvasService) ResolveExecutionErrors(ctx context.Context, req *pb.ResolveExecutionErrorsRequest) (*pb.ResolveExecutionErrorsResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	executionIDs := make([]uuid.UUID, 0, len(req.ExecutionIds))
	for _, executionID := range req.ExecutionIds {
		parsedID, err := uuid.Parse(executionID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
		}
		executionIDs = append(executionIDs, parsedID)
	}

	return canvases.ResolveExecutionErrors(ctx, db, canvas, executionIDs)
}

func (s *CanvasService) GetCanvasRepository(ctx context.Context, req *pb.GetCanvasRepositoryRequest) (*pb.GetCanvasRepositoryResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.GetCanvasRepository(ctx, s.gitProvider, canvas)
}

func (s *CanvasService) ListCanvasRepositoryFiles(ctx context.Context, req *pb.ListCanvasRepositoryFilesRequest) (*pb.ListCanvasRepositoryFilesResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}
	return canvases.ListCanvasRepositoryFiles(ctx, s.gitProvider, canvas)
}

func (s *CanvasService) PutCanvasStaging(ctx context.Context, req *pb.PutCanvasStagingRequest) (*pb.PutCanvasStagingResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	state, err := canvases.PutCanvasStaging(ctx, db, canvas, req.Operations)
	if err != nil {
		return nil, err
	}
	return &pb.PutCanvasStagingResponse{StagingSummary: state}, nil
}

func (s *CanvasService) GetCanvasStaging(ctx context.Context, req *pb.GetCanvasStagingRequest) (*pb.GetCanvasStagingResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	state, err := canvases.GetCanvasStaging(ctx, db, canvas)
	if err != nil {
		return nil, err
	}
	return &pb.GetCanvasStagingResponse{StagingSummary: state}, nil
}

func (s *CanvasService) DeleteCanvasStaging(ctx context.Context, req *pb.DeleteCanvasStagingRequest) (*pb.DeleteCanvasStagingResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	state, err := canvases.DeleteCanvasStaging(ctx, db, canvas, req.Paths)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteCanvasStagingResponse{StagingSummary: state}, nil
}

func (s *CanvasService) CommitCanvasStaging(ctx context.Context, req *pb.CommitCanvasStagingRequest) (*pb.CommitCanvasStagingResponse, error) {
	db := database.DB(ctx)
	canvas, err := s.findCanvas(ctx, db, req.CanvasId)
	if err != nil {
		return nil, err
	}

	return canvases.CommitCanvasStaging(
		ctx,
		db,
		s.gitProvider,
		s.usageService,
		s.encryptor,
		s.registry,
		canvas,
		req.CommitMessage,
		s.webhookBaseURL,
		s.authService,
	)
}
