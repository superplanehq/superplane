package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	groups "github.com/superplanehq/superplane/pkg/grpc/actions/connection_groups"
	eventsources "github.com/superplanehq/superplane/pkg/grpc/actions/event_sources"
	"github.com/superplanehq/superplane/pkg/grpc/actions/secrets"
	stageevents "github.com/superplanehq/superplane/pkg/grpc/actions/stage_events"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

type SuperplaneService struct {
	encryptor            crypto.Encryptor
	specValidator        executors.SpecValidator
	authorizationService authorization.Authorization
}

func NewSuperplaneService(encryptor crypto.Encryptor, authService authorization.Authorization) *SuperplaneService {
	return &SuperplaneService{
		encryptor:            encryptor,
		specValidator:        executors.SpecValidator{},
		authorizationService: authService,
	}
}

func (s *SuperplaneService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	return canvases.CreateCanvas(ctx, req, s.authorizationService)
}

func (s *SuperplaneService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	return canvases.DescribeCanvas(ctx, req)
}

func (s *SuperplaneService) CreateEventSource(ctx context.Context, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	return eventsources.CreateEventSource(ctx, s.encryptor, req)
}

func (s *SuperplaneService) DescribeEventSource(ctx context.Context, req *pb.DescribeEventSourceRequest) (*pb.DescribeEventSourceResponse, error) {
	return eventsources.DescribeEventSource(ctx, req)
}

func (s *SuperplaneService) ResetEventSourceKey(ctx context.Context, req *pb.ResetEventSourceKeyRequest) (*pb.ResetEventSourceKeyResponse, error) {
	return eventsources.ResetEventSourceKey(ctx, s.encryptor, req)
}

func (s *SuperplaneService) CreateStage(ctx context.Context, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	return stages.CreateStage(ctx, s.specValidator, req)
}

func (s *SuperplaneService) DescribeStage(ctx context.Context, req *pb.DescribeStageRequest) (*pb.DescribeStageResponse, error) {
	return stages.DescribeStage(ctx, req)
}

func (s *SuperplaneService) UpdateStage(ctx context.Context, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	return stages.UpdateStage(ctx, s.specValidator, req)
}

func (s *SuperplaneService) ApproveStageEvent(ctx context.Context, req *pb.ApproveStageEventRequest) (*pb.ApproveStageEventResponse, error) {
	return stageevents.ApproveStageEvent(ctx, req)
}

func (s *SuperplaneService) ListEventSources(ctx context.Context, req *pb.ListEventSourcesRequest) (*pb.ListEventSourcesResponse, error) {
	return eventsources.ListEventSources(ctx, req)
}

func (s *SuperplaneService) ListStages(ctx context.Context, req *pb.ListStagesRequest) (*pb.ListStagesResponse, error) {
	return stages.ListStages(ctx, req)
}

func (s *SuperplaneService) ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	return canvases.ListCanvases(ctx, req, s.authorizationService)
}

func (s *SuperplaneService) ListStageEvents(ctx context.Context, req *pb.ListStageEventsRequest) (*pb.ListStageEventsResponse, error) {
	return stageevents.ListStageEvents(ctx, req)
}

func (s *SuperplaneService) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	return secrets.CreateSecret(ctx, s.encryptor, req)
}

func (s *SuperplaneService) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	return secrets.UpdateSecret(ctx, s.encryptor, req)
}

func (s *SuperplaneService) DescribeSecret(ctx context.Context, req *pb.DescribeSecretRequest) (*pb.DescribeSecretResponse, error) {
	return secrets.DescribeSecret(ctx, s.encryptor, req)
}

func (s *SuperplaneService) ListSecrets(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	return secrets.ListSecrets(ctx, s.encryptor, req)
}

func (s *SuperplaneService) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	return secrets.DeleteSecret(ctx, req)
}

func (s *SuperplaneService) CreateConnectionGroup(ctx context.Context, req *pb.CreateConnectionGroupRequest) (*pb.CreateConnectionGroupResponse, error) {
	return groups.CreateConnectionGroup(ctx, req)
}

func (s *SuperplaneService) UpdateConnectionGroup(ctx context.Context, req *pb.UpdateConnectionGroupRequest) (*pb.UpdateConnectionGroupResponse, error) {
	return groups.UpdateConnectionGroup(ctx, req)
}

func (s *SuperplaneService) DescribeConnectionGroup(ctx context.Context, req *pb.DescribeConnectionGroupRequest) (*pb.DescribeConnectionGroupResponse, error) {
	return groups.DescribeConnectionGroup(ctx, req)
}

func (s *SuperplaneService) ListConnectionGroups(ctx context.Context, req *pb.ListConnectionGroupsRequest) (*pb.ListConnectionGroupsResponse, error) {
	return groups.ListConnectionGroups(ctx, req)
}

func (s *SuperplaneService) ListConnectionGroupFieldSets(ctx context.Context, req *pb.ListConnectionGroupFieldSetsRequest) (*pb.ListConnectionGroupFieldSetsResponse, error) {
	return groups.ListConnectionGroupFieldSets(ctx, req)
}
