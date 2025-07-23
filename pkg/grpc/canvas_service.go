package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	groups "github.com/superplanehq/superplane/pkg/grpc/actions/connection_groups"
	eventsources "github.com/superplanehq/superplane/pkg/grpc/actions/event_sources"
	stageevents "github.com/superplanehq/superplane/pkg/grpc/actions/stage_events"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

type CanvasService struct {
	encryptor            crypto.Encryptor
	specValidator        executors.SpecValidator
	authorizationService authorization.Authorization
}

func NewCanvasService(encryptor crypto.Encryptor, authService authorization.Authorization) *CanvasService {
	return &CanvasService{
		encryptor:            encryptor,
		specValidator:        executors.SpecValidator{Encryptor: encryptor},
		authorizationService: authService,
	}
}

func (s *CanvasService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	return canvases.CreateCanvas(ctx, req, s.authorizationService)
}

func (s *CanvasService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	return canvases.DescribeCanvas(ctx, req)
}

func (s *CanvasService) CreateEventSource(ctx context.Context, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	return eventsources.CreateEventSource(ctx, s.encryptor, req)
}

func (s *CanvasService) DescribeEventSource(ctx context.Context, req *pb.DescribeEventSourceRequest) (*pb.DescribeEventSourceResponse, error) {
	return eventsources.DescribeEventSource(ctx, req)
}

func (s *CanvasService) ResetEventSourceKey(ctx context.Context, req *pb.ResetEventSourceKeyRequest) (*pb.ResetEventSourceKeyResponse, error) {
	return eventsources.ResetEventSourceKey(ctx, s.encryptor, req)
}

func (s *CanvasService) CreateStage(ctx context.Context, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	return stages.CreateStage(ctx, s.encryptor, s.specValidator, req)
}

func (s *CanvasService) DescribeStage(ctx context.Context, req *pb.DescribeStageRequest) (*pb.DescribeStageResponse, error) {
	return stages.DescribeStage(ctx, req)
}

func (s *CanvasService) UpdateStage(ctx context.Context, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	return stages.UpdateStage(ctx, s.encryptor, s.specValidator, req)
}

func (s *CanvasService) ApproveStageEvent(ctx context.Context, req *pb.ApproveStageEventRequest) (*pb.ApproveStageEventResponse, error) {
	return stageevents.ApproveStageEvent(ctx, req)
}

func (s *CanvasService) ListEventSources(ctx context.Context, req *pb.ListEventSourcesRequest) (*pb.ListEventSourcesResponse, error) {
	return eventsources.ListEventSources(ctx, req)
}

func (s *CanvasService) ListStages(ctx context.Context, req *pb.ListStagesRequest) (*pb.ListStagesResponse, error) {
	return stages.ListStages(ctx, req)
}

func (s *CanvasService) ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	return canvases.ListCanvases(ctx, req, s.authorizationService)
}

func (s *CanvasService) ListStageEvents(ctx context.Context, req *pb.ListStageEventsRequest) (*pb.ListStageEventsResponse, error) {
	return stageevents.ListStageEvents(ctx, req)
}

func (s *CanvasService) CreateConnectionGroup(ctx context.Context, req *pb.CreateConnectionGroupRequest) (*pb.CreateConnectionGroupResponse, error) {
	return groups.CreateConnectionGroup(ctx, req)
}

func (s *CanvasService) UpdateConnectionGroup(ctx context.Context, req *pb.UpdateConnectionGroupRequest) (*pb.UpdateConnectionGroupResponse, error) {
	return groups.UpdateConnectionGroup(ctx, req)
}

func (s *CanvasService) DescribeConnectionGroup(ctx context.Context, req *pb.DescribeConnectionGroupRequest) (*pb.DescribeConnectionGroupResponse, error) {
	return groups.DescribeConnectionGroup(ctx, req)
}

func (s *CanvasService) ListConnectionGroups(ctx context.Context, req *pb.ListConnectionGroupsRequest) (*pb.ListConnectionGroupsResponse, error) {
	return groups.ListConnectionGroups(ctx, req)
}

func (s *CanvasService) ListConnectionGroupFieldSets(ctx context.Context, req *pb.ListConnectionGroupFieldSetsRequest) (*pb.ListConnectionGroupFieldSetsResponse, error) {
	return groups.ListConnectionGroupFieldSets(ctx, req)
}
