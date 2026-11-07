package runners

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	runneractions "github.com/superplanehq/superplane/pkg/grpc/actions/runners"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type Service struct {
	manager *runneractions.RunnerManager
}

func NewService(signer *jwt.Signer, baseURL string) *Service {
	return &Service{manager: runneractions.NewRunnerManager(signer, baseURL)}
}

func (s *Service) ListFleets(ctx context.Context, _ *pb.ListFleetsRequest) (*pb.ListFleetsResponse, error) {
	fleets, err := models.ListInstallationRunnerFleets(database.DB(ctx))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list fleets")
	}

	out := make([]*pb.Fleet, 0, len(fleets))
	for i := range fleets {
		out = append(out, serializeFleet(&fleets[i]))
	}
	return &pb.ListFleetsResponse{Fleets: out}, nil
}

func (s *Service) CreateFleet(
	ctx context.Context,
	req *pb.CreateFleetRequest,
) (*pb.CreateFleetResponse, error) {
	fleetID, err := parseFleetID(req.GetFleetId())
	if err != nil {
		return nil, err
	}
	runnerVersion := strings.TrimSpace(req.GetRunnerVersion())
	if runnerVersion == "" {
		return nil, grpcerrors.InvalidArgument(nil, "runner version is required")
	}
	if req.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "fleet spec is required")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = req.GetEnabled()
	}
	now := time.Now()
	fleet := &models.RunnerFleet{
		ID:            uuid.New(),
		Slug:          fleetID,
		ScopeType:     models.RunnerFleetScopeInstallation,
		Enabled:       enabled,
		Spec:          datatypes.NewJSONType(fleetSpecFromProto(req.Spec)),
		RunnerVersion: runnerVersion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = fleet.Create(database.DB(ctx))
	if errors.Is(err, models.ErrRunnerFleetIDConflict) {
		return nil, grpcerrors.AlreadyExists(err, "fleet ID already exists")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create fleet")
	}
	return &pb.CreateFleetResponse{Fleet: serializeFleet(fleet)}, nil
}

func (s *Service) DescribeFleet(ctx context.Context, req *pb.DescribeFleetRequest) (*pb.DescribeFleetResponse, error) {
	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeFleetResponse{Fleet: serializeFleet(fleet)}, nil
}

func (s *Service) ListFleetTasks(
	ctx context.Context,
	req *pb.ListFleetTasksRequest,
) (*pb.ListFleetTasksResponse, error) {
	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	page, err := runneractions.ListFleetTasks(ctx, fleet, req.GetStates(), req.GetLimit(), req.GetAfterId())
	if err != nil {
		return nil, err
	}

	out := make([]*pb.Task, 0, len(page.Tasks))
	for i := range page.Tasks {
		out = append(out, serializeTask(&page.Tasks[i], fleet.Slug))
	}
	return &pb.ListFleetTasksResponse{
		Tasks:       out,
		TotalCount:  page.TotalCount,
		HasNextPage: page.HasNextPage,
	}, nil
}

func (s *Service) findFleet(ctx context.Context, rawID string) (*models.RunnerFleet, error) {
	id, err := parseFleetID(rawID)
	if err != nil {
		return nil, err
	}
	fleet, err := models.FindInstallationRunnerFleet(database.DB(ctx), id)
	if errors.Is(err, models.ErrRunnerFleetNotFound) {
		return nil, grpcerrors.NotFound(err, "fleet not found")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to find fleet")
	}
	return fleet, nil
}

func parseFleetID(value string) (string, error) {
	id := strings.TrimSpace(value)
	if id == "" {
		return "", grpcerrors.InvalidArgument(nil, "fleet ID is required")
	}
	return id, nil
}

func (s *Service) UpdateFleet(ctx context.Context, req *pb.UpdateFleetRequest) (*pb.UpdateFleetResponse, error) {

	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.UpdateFleet(ctx, fleet, runneractions.UpdateFleetOptions{RunnerVersion: req.RunnerVersion, Enabled: req.Enabled, Spec: optionalFleetSpec(req.Spec)})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateFleetResponse{Fleet: serializeFleet(out)}, nil
}

func (s *Service) GetFleetCapacity(ctx context.Context, req *pb.GetFleetCapacityRequest) (*pb.GetFleetCapacityResponse, error) {

	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.GetFleetCapacity(ctx, fleet, runneractions.CapacityOptions{Generation: req.Generation, WaitSeconds: req.GetWaitSeconds()})
	if err != nil {
		return nil, err
	}
	return &pb.GetFleetCapacityResponse{RunnableTasks: out.RunnableTasks, PendingRunners: out.PendingRunners, IdleRunners: out.IdleRunners, BusyRunners: out.BusyRunners, TerminatedRunners: out.TerminatedRunners, Generation: out.Generation}, nil
}

func (s *Service) CreateRunner(ctx context.Context, req *pb.CreateRunnerRequest) (*pb.CreateRunnerResponse, error) {
	if !req.GetEphemeral() {
		return nil, grpcerrors.InvalidArgument(nil, "installation fleet runners must be ephemeral")
	}
	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.CreateRunner(ctx, fleet, runneractions.CreateRunnerOptions{TaskID: req.TaskId, Ephemeral: req.Ephemeral, IdempotencyKey: req.IdempotencyKey, FleetManagerID: req.FleetManagerId, Provider: req.Provider})
	if err != nil {
		return nil, err
	}
	return &pb.CreateRunnerResponse{Runner: serializeRunner(out.Runner, fleet.Slug), RegistrationToken: out.RegistrationToken, RegistrationExpiresAt: timestamppb.New(out.RegistrationExpiresAt), RunnerApiUrl: out.RunnerAPIURL, DisplayName: out.DisplayName}, nil
}

func (s *Service) DescribeRunner(ctx context.Context, req *pb.DescribeRunnerRequest) (*pb.DescribeRunnerResponse, error) {

	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.DescribeRunner(ctx, fleet, req.GetRunnerId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeRunnerResponse{Runner: serializeRunner(out, fleet.Slug)}, nil
}

func (s *Service) DeleteRunner(ctx context.Context, req *pb.DeleteRunnerRequest) (*pb.DeleteRunnerResponse, error) {

	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.TerminateRunner(ctx, fleet, runneractions.TerminateRunnerOptions{RunnerID: req.RunnerId, States: req.States})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteRunnerResponse{Runner: serializeRunner(out, fleet.Slug)}, nil
}

func (s *Service) MarkRunnerProviderDeleted(ctx context.Context, req *pb.MarkRunnerProviderDeletedRequest) (*pb.MarkRunnerProviderDeletedResponse, error) {

	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.MarkRunnerProviderDeleted(ctx, fleet, runneractions.ProviderDeletionOptions{RunnerID: req.RunnerId, FleetManagerID: req.FleetManagerId, Provider: req.Provider})
	if err != nil {
		return nil, err
	}
	return &pb.MarkRunnerProviderDeletedResponse{Runner: serializeRunner(out, fleet.Slug)}, nil
}

func (s *Service) ListRunners(ctx context.Context, req *pb.ListRunnersRequest) (*pb.ListRunnersResponse, error) {
	fleet, err := s.findFleet(ctx, req.GetFleetId())
	if err != nil {
		return nil, err
	}
	page, err := s.manager.ListRunners(ctx, fleet, runneractions.ListRunnersOptions{States: req.States, Limit: req.GetLimit(), AfterID: req.GetAfterId()})
	if err != nil {
		return nil, err
	}
	out := make([]*pb.Runner, 0, len(page.Runners))
	for i := range page.Runners {
		out = append(out, serializeRunner(&page.Runners[i], fleet.Slug))
	}
	return &pb.ListRunnersResponse{Runners: out, TotalCount: page.TotalCount, HasNextPage: page.HasNextPage}, nil
}

func optionalFleetSpec(spec *pb.FleetSpec) *models.RunnerFleetSpec {
	if spec == nil {
		return nil
	}
	result := fleetSpecFromProto(spec)
	return &result
}
