package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	eventsources "github.com/superplanehq/superplane/pkg/grpc/actions/event_sources"
	"github.com/superplanehq/superplane/pkg/grpc/actions/secrets"
	stageevents "github.com/superplanehq/superplane/pkg/grpc/actions/stage_events"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

type DeliveryService struct {
	encryptor            crypto.Encryptor
	specValidator        executors.SpecValidator
	authorizationService authorization.Authorization
}

func NewDeliveryService(encryptor crypto.Encryptor, authService authorization.Authorization) *DeliveryService {
	return &DeliveryService{
		encryptor:            encryptor,
		specValidator:        executors.SpecValidator{},
		authorizationService: authService,
	}
}

func (s *DeliveryService) CreateCanvas(ctx context.Context, req *pb.CreateCanvasRequest) (*pb.CreateCanvasResponse, error) {
	return canvases.CreateCanvas(ctx, req, s.authorizationService)
}

func (s *DeliveryService) DescribeCanvas(ctx context.Context, req *pb.DescribeCanvasRequest) (*pb.DescribeCanvasResponse, error) {
	return canvases.DescribeCanvas(ctx, req)
}

func (s *DeliveryService) CreateEventSource(ctx context.Context, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	return eventsources.CreateEventSource(ctx, s.encryptor, req)
}

func (s *DeliveryService) DescribeEventSource(ctx context.Context, req *pb.DescribeEventSourceRequest) (*pb.DescribeEventSourceResponse, error) {
	return eventsources.DescribeEventSource(ctx, req)
}

func (s *DeliveryService) CreateStage(ctx context.Context, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	return stages.CreateStage(ctx, s.specValidator, req)
}

func (s *DeliveryService) DescribeStage(ctx context.Context, req *pb.DescribeStageRequest) (*pb.DescribeStageResponse, error) {
	return stages.DescribeStage(ctx, req)
}

func (s *DeliveryService) UpdateStage(ctx context.Context, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	return stages.UpdateStage(ctx, s.specValidator, req)
}

func (s *DeliveryService) ApproveStageEvent(ctx context.Context, req *pb.ApproveStageEventRequest) (*pb.ApproveStageEventResponse, error) {
	return stageevents.ApproveStageEvent(ctx, req)
}

func (s *DeliveryService) ListEventSources(ctx context.Context, req *pb.ListEventSourcesRequest) (*pb.ListEventSourcesResponse, error) {
	return eventsources.ListEventSources(ctx, req)
}

func (s *DeliveryService) ListStages(ctx context.Context, req *pb.ListStagesRequest) (*pb.ListStagesResponse, error) {
	return stages.ListStages(ctx, req)
}

func (s *DeliveryService) ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	return canvases.ListCanvases(ctx, req, s.authorizationService)
}

func (s *DeliveryService) ListStageEvents(ctx context.Context, req *pb.ListStageEventsRequest) (*pb.ListStageEventsResponse, error) {
	return stageevents.ListStageEvents(ctx, req)
}

func (s *DeliveryService) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	return secrets.CreateSecret(ctx, s.encryptor, req)
}

func (s *DeliveryService) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	return secrets.UpdateSecret(ctx, s.encryptor, req)
}

func (s *DeliveryService) DescribeSecret(ctx context.Context, req *pb.DescribeSecretRequest) (*pb.DescribeSecretResponse, error) {
	return secrets.DescribeSecret(ctx, s.encryptor, req)
}

func (s *DeliveryService) ListSecrets(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	return secrets.ListSecrets(ctx, s.encryptor, req)
}

func (s *DeliveryService) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	return secrets.DeleteSecret(ctx, req)
}
