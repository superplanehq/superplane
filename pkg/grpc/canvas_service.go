package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	groups "github.com/superplanehq/superplane/pkg/grpc/actions/connection_groups"
	eventsources "github.com/superplanehq/superplane/pkg/grpc/actions/event_sources"
	"github.com/superplanehq/superplane/pkg/grpc/actions/events"
	stageevents "github.com/superplanehq/superplane/pkg/grpc/actions/stage_events"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

type CanvasService struct {
	encryptor            crypto.Encryptor
	registry             *registry.Registry
	authorizationService authorization.Authorization
}

func NewCanvasService(encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry) *CanvasService {
	return &CanvasService{
		encryptor:            encryptor,
		authorizationService: authService,
		registry:             registry,
	}
}

//
// Methods for canvases
//

func (s *CanvasService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return canvases.CreateCanvas(ctx, s.authorizationService, orgID, req.Canvas)
}

func (s *CanvasService) DeleteCanvas(ctx context.Context, req *pb.DeleteCanvasRequest) (*pb.DeleteCanvasResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return canvases.DeleteCanvas(ctx, orgID, req, s.authorizationService)
}

func (s *CanvasService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return canvases.DescribeCanvas(ctx, orgID, req)
}

func (s *CanvasService) ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	orgID := ctx.Value(authorization.DomainIdContextKey).(string)
	return canvases.ListCanvases(ctx, orgID, s.authorizationService)
}

func (s *CanvasService) AddUser(ctx context.Context, req *pb.AddUserRequest) (*pb.AddUserResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return canvases.AddUser(ctx, s.authorizationService, orgID, canvasID, req.UserId)
}

func (s *CanvasService) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (*pb.RemoveUserResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return canvases.RemoveUser(ctx, s.authorizationService, orgID, canvasID, req.UserId)
}

//
// Methods for event sources
//

func (s *CanvasService) CreateEventSource(ctx context.Context, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return eventsources.CreateEventSource(ctx, s.encryptor, s.registry, orgID, canvasID, req.EventSource)
}

func (s *CanvasService) UpdateEventSource(ctx context.Context, req *pb.UpdateEventSourceRequest) (*pb.UpdateEventSourceResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return eventsources.UpdateEventSource(ctx, s.encryptor, s.registry, orgID, canvasID, req.IdOrName, req.EventSource)
}

func (s *CanvasService) DescribeEventSource(ctx context.Context, req *pb.DescribeEventSourceRequest) (*pb.DescribeEventSourceResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return eventsources.DescribeEventSource(ctx, canvasID, req.IdOrName)
}

func (s *CanvasService) ResetEventSourceKey(ctx context.Context, req *pb.ResetEventSourceKeyRequest) (*pb.ResetEventSourceKeyResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return eventsources.ResetEventSourceKey(ctx, s.encryptor, canvasID, req.IdOrName)
}

func (s *CanvasService) ListEventSources(ctx context.Context, req *pb.ListEventSourcesRequest) (*pb.ListEventSourcesResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return eventsources.ListEventSources(ctx, canvasID)
}

func (s *CanvasService) DeleteEventSource(ctx context.Context, req *pb.DeleteEventSourceRequest) (*pb.DeleteEventSourceResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return eventsources.DeleteEventSource(ctx, canvasID, req.IdOrName)
}

//
// Methods for stages
//

func (s *CanvasService) CreateStage(ctx context.Context, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stages.CreateStage(ctx, s.encryptor, s.registry, orgID, canvasID, req.Stage)
}

func (s *CanvasService) DescribeStage(ctx context.Context, req *pb.DescribeStageRequest) (*pb.DescribeStageResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stages.DescribeStage(ctx, canvasID, req.IdOrName)
}

func (s *CanvasService) UpdateStage(ctx context.Context, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stages.UpdateStage(ctx, s.encryptor, s.registry, orgID, canvasID, req.IdOrName, req.Stage)
}

func (s *CanvasService) ApproveStageEvent(ctx context.Context, req *pb.ApproveStageEventRequest) (*pb.ApproveStageEventResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stageevents.ApproveStageEvent(ctx, canvasID, req.StageIdOrName, req.EventId)
}

func (s *CanvasService) ListStages(ctx context.Context, req *pb.ListStagesRequest) (*pb.ListStagesResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stages.ListStages(ctx, canvasID)
}

func (s *CanvasService) ListStageEvents(ctx context.Context, req *pb.ListStageEventsRequest) (*pb.ListStageEventsResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stageevents.ListStageEvents(ctx, canvasID, req.StageIdOrName, req.States)
}

func (s *CanvasService) DeleteStage(ctx context.Context, req *pb.DeleteStageRequest) (*pb.DeleteStageResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stages.DeleteStage(ctx, canvasID, req.IdOrName)
}

//
// Methods for connection groups
//

func (s *CanvasService) CreateConnectionGroup(ctx context.Context, req *pb.CreateConnectionGroupRequest) (*pb.CreateConnectionGroupResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return groups.CreateConnectionGroup(ctx, canvasID, req.ConnectionGroup)
}

func (s *CanvasService) UpdateConnectionGroup(ctx context.Context, req *pb.UpdateConnectionGroupRequest) (*pb.UpdateConnectionGroupResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return groups.UpdateConnectionGroup(ctx, canvasID, req.IdOrName, req.ConnectionGroup)
}

func (s *CanvasService) DescribeConnectionGroup(ctx context.Context, req *pb.DescribeConnectionGroupRequest) (*pb.DescribeConnectionGroupResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return groups.DescribeConnectionGroup(ctx, canvasID, req.IdOrName)
}

func (s *CanvasService) ListConnectionGroups(ctx context.Context, req *pb.ListConnectionGroupsRequest) (*pb.ListConnectionGroupsResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return groups.ListConnectionGroups(ctx, canvasID)
}

func (s *CanvasService) ListConnectionGroupFieldSets(ctx context.Context, req *pb.ListConnectionGroupFieldSetsRequest) (*pb.ListConnectionGroupFieldSetsResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return groups.ListConnectionGroupFieldSets(ctx, canvasID, req.IdOrName)
}

func (s *CanvasService) DeleteConnectionGroup(ctx context.Context, req *pb.DeleteConnectionGroupRequest) (*pb.DeleteConnectionGroupResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return groups.DeleteConnectionGroup(ctx, canvasID, req.IdOrName)
}

func (s *CanvasService) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return events.ListEvents(ctx, canvasID, req.SourceType, req.SourceId)
}

func (s *CanvasService) BulkListEvents(ctx context.Context, req *pb.BulkListEventsRequest) (*pb.BulkListEventsResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return events.BulkListEvents(ctx, canvasID, req.Sources, req.LimitPerSource)
}

func (s *CanvasService) BulkListStageEvents(ctx context.Context, req *pb.BulkListStageEventsRequest) (*pb.BulkListStageEventsResponse, error) {
	canvasID := ctx.Value(authorization.DomainIdContextKey).(string)
	return stageevents.BulkListStageEvents(ctx, canvasID, req.Stages, req.LimitPerStage)
}
