package grpc

import (
	"context"

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
	encryptor     crypto.Encryptor
	specValidator executors.SpecValidator
}

func NewSuperplaneService(encryptor crypto.Encryptor) *SuperplaneService {
	return &SuperplaneService{
		encryptor:     encryptor,
		specValidator: executors.SpecValidator{},
	}
}

func (s *SuperplaneService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	return canvases.CreateCanvas(ctx, req)
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
	return canvases.ListCanvases(ctx, req)
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
