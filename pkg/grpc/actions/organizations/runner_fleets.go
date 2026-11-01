package organizations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	runneractions "github.com/superplanehq/superplane/pkg/grpc/actions/runners"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RunnerService implements the organization fleet and runner endpoints.
// Authorization supplies the domain; request bodies cannot change ownership.
type RunnerService struct {
	manager *runneractions.RunnerManager
}

func NewRunnerService(manager *runneractions.RunnerManager) *RunnerService {
	return &RunnerService{manager: manager}
}

func runnerOrganization(ctx context.Context, requestedRef string) (uuid.UUID, error) {
	ref, ok := ctx.Value(authorization.DomainIdContextKey).(string)
	if !ok || ref == "" {
		return uuid.Nil, grpcerrors.PermissionDenied(nil, "organization authorization is required")
	}
	organizationID, err := resolveOrganizationID(ctx, ref)
	if err != nil {
		return uuid.Nil, err
	}
	if requestedRef != "" {
		requestedID, err := resolveOrganizationID(ctx, requestedRef)
		if err != nil {
			return uuid.Nil, err
		}
		if requestedID != organizationID {
			return uuid.Nil, grpcerrors.NotFound(errors.New("organization does not match authorized domain"), "organization not found")
		}
	}
	return organizationID, nil
}

func (s *RunnerService) fleet(ctx context.Context, organizationRef, slug string) (*models.RunnerFleet, error) {
	organizationID, err := runnerOrganization(ctx, organizationRef)
	if err != nil {
		return nil, err
	}
	fleet, err := models.FindOrganizationRunnerFleet(database.DB(ctx), organizationID, strings.TrimSpace(slug))
	if errors.Is(err, models.ErrRunnerFleetNotFound) {
		return nil, grpcerrors.NotFound(err, "fleet not found")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to find fleet")
	}
	return fleet, nil
}

func (s *RunnerService) ListManagedRunnerFleets(ctx context.Context, req *pb.ListManagedRunnerFleetsRequest) (*pb.ListManagedRunnerFleetsResponse, error) {
	organizationID, err := runnerOrganization(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	fleets, err := models.ListOrganizationRunnerFleets(database.DB(ctx), organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list fleets")
	}
	out := &pb.ListManagedRunnerFleetsResponse{Fleets: []*pb.Fleet{}}
	for i := range fleets {
		out.Fleets = append(out.Fleets, managedFleet(&fleets[i]))
	}
	return out, nil
}

func (s *RunnerService) CreateRunnerFleet(ctx context.Context, req *pb.CreateRunnerFleetRequest) (*pb.CreateRunnerFleetResponse, error) {
	organizationID, err := runnerOrganization(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	slug, version := strings.TrimSpace(req.GetFleetId()), strings.TrimSpace(req.GetRunnerVersion())
	if slug == "" || version == "" || req.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "fleet ID, runner version, and fleet spec are required")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = req.GetEnabled()
	}
	now := time.Now()
	fleet := &models.RunnerFleet{ID: uuid.New(), Slug: slug, ScopeType: models.RunnerFleetScopeOrganization,
		ScopeID: &organizationID, Enabled: enabled, RunnerVersion: version, CreatedAt: now, UpdatedAt: now,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{OperatingSystem: req.Spec.OperatingSystem,
			Architecture: req.Spec.Architecture, CPUMillicores: req.Spec.CpuMillicores, MemoryMB: req.Spec.MemoryMb,
			DiskGB: req.Spec.DiskGb, Capabilities: req.Spec.Capabilities})}
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := models.RequireOrganizationRunnerFeature(tx, organizationID); err != nil {
			return err
		}
		return fleet.Create(tx)
	})
	if errors.Is(err, models.ErrOrganizationRunnersDisabled) {
		return nil, grpcerrors.FailedPrecondition(err, err.Error())
	}
	if errors.Is(err, models.ErrRunnerFleetIDConflict) {
		return nil, grpcerrors.AlreadyExists(err, "fleet ID already exists")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create fleet")
	}
	return &pb.CreateRunnerFleetResponse{Fleet: managedFleet(fleet)}, nil
}

func (s *RunnerService) DescribeRunnerFleet(ctx context.Context, req *pb.DescribeRunnerFleetRequest) (*pb.DescribeRunnerFleetResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeRunnerFleetResponse{Fleet: managedFleet(fleet)}, nil
}

func (s *RunnerService) UpdateRunnerFleet(ctx context.Context, req *pb.UpdateRunnerFleetRequest) (*pb.UpdateRunnerFleetResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.UpdateFleet(ctx, fleet, runneractions.UpdateFleetOptions{RunnerVersion: req.RunnerVersion, Enabled: req.Enabled, Spec: runnerFleetSpec(req.Spec)})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRunnerFleetResponse{Fleet: managedFleet(out)}, nil
}

func (s *RunnerService) GetRunnerFleetCapacity(ctx context.Context, req *pb.GetRunnerFleetCapacityRequest) (*pb.GetRunnerFleetCapacityResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.GetFleetCapacity(ctx, fleet, runneractions.CapacityOptions{Generation: req.Generation, WaitSeconds: req.GetWaitSeconds()})
	if err != nil {
		return nil, err
	}
	return &pb.GetRunnerFleetCapacityResponse{RunnableTasks: out.RunnableTasks, PendingRunners: out.PendingRunners,
		IdleRunners: out.IdleRunners, BusyRunners: out.BusyRunners, TerminatedRunners: out.TerminatedRunners, Generation: out.Generation}, nil
}

func (s *RunnerService) CreateFleetRunner(ctx context.Context, req *pb.CreateFleetRunnerRequest) (*pb.CreateFleetRunnerResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.CreateRunner(ctx, fleet, runneractions.CreateRunnerOptions{TaskID: req.TaskId,
		IdempotencyKey: req.IdempotencyKey, Ephemeral: req.Ephemeral, FleetManagerID: req.FleetManagerId, Provider: req.Provider})
	if err != nil {
		return nil, err
	}
	return &pb.CreateFleetRunnerResponse{Runner: organizationRunner(out.Runner, fleet.Slug), RegistrationToken: out.RegistrationToken,
		RegistrationExpiresAt: timestamppb.New(out.RegistrationExpiresAt), RunnerApiUrl: out.RunnerAPIURL, DisplayName: out.DisplayName}, nil
}

func (s *RunnerService) ListFleetRunners(ctx context.Context, req *pb.ListFleetRunnersRequest) (*pb.ListFleetRunnersResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.ListRunners(ctx, fleet, runneractions.ListRunnersOptions{States: req.States, Limit: req.GetLimit(), AfterID: req.GetAfterId()})
	if err != nil {
		return nil, err
	}
	response := &pb.ListFleetRunnersResponse{Runners: []*pb.Runner{}, TotalCount: out.TotalCount, HasNextPage: out.HasNextPage}
	for i := range out.Runners {
		response.Runners = append(response.Runners, organizationRunner(&out.Runners[i], fleet.Slug))
	}
	return response, nil
}

func (s *RunnerService) DescribeFleetRunner(ctx context.Context, req *pb.DescribeFleetRunnerRequest) (*pb.DescribeFleetRunnerResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.DescribeRunner(ctx, fleet, req.GetRunnerId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeFleetRunnerResponse{Runner: organizationRunner(out, fleet.Slug)}, nil
}

func (s *RunnerService) TerminateFleetRunner(ctx context.Context, req *pb.TerminateFleetRunnerRequest) (*pb.TerminateFleetRunnerResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.TerminateRunner(ctx, fleet, runneractions.TerminateRunnerOptions{RunnerID: req.RunnerId, States: req.States})
	if err != nil {
		return nil, err
	}
	return &pb.TerminateFleetRunnerResponse{Runner: organizationRunner(out, fleet.Slug)}, nil
}

func (s *RunnerService) MarkFleetRunnerProviderDeleted(ctx context.Context, req *pb.MarkFleetRunnerProviderDeletedRequest) (*pb.MarkFleetRunnerProviderDeletedResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := s.manager.MarkRunnerProviderDeleted(ctx, fleet, runneractions.ProviderDeletionOptions{RunnerID: req.RunnerId, FleetManagerID: req.FleetManagerId, Provider: req.Provider})
	if err != nil {
		return nil, err
	}
	return &pb.MarkFleetRunnerProviderDeletedResponse{Runner: organizationRunner(out, fleet.Slug)}, nil
}

func runnerFleetSpec(spec *pb.OrganizationRunnerFleetSpec) *models.RunnerFleetSpec {
	if spec == nil {
		return nil
	}
	return &models.RunnerFleetSpec{OperatingSystem: spec.OperatingSystem, Architecture: spec.Architecture,
		CPUMillicores: spec.CpuMillicores, MemoryMB: spec.MemoryMb, DiskGB: spec.DiskGb, Capabilities: append([]string(nil), spec.Capabilities...)}
}

func managedFleet(fleet *models.RunnerFleet) *pb.Fleet {
	spec := fleet.Spec.Data()
	return &pb.Fleet{Id: fleet.Slug, Enabled: fleet.Enabled, RunnerVersion: fleet.RunnerVersion,
		CreatedAt: timestamppb.New(fleet.CreatedAt), UpdatedAt: timestamppb.New(fleet.UpdatedAt), Spec: &pb.OrganizationRunnerFleetSpec{
			OperatingSystem: spec.OperatingSystem, Architecture: spec.Architecture, CpuMillicores: spec.CPUMillicores,
			MemoryMb: spec.MemoryMB, DiskGb: spec.DiskGB, Capabilities: append([]string(nil), spec.Capabilities...)}}
}

func organizationRunner(runner *models.Runner, fleetID string) *pb.Runner {
	out := &pb.Runner{
		Id:            runner.ID.String(),
		FleetId:       fleetID,
		State:         runner.State,
		RunnerVersion: runner.RunnerVersion,
		Os:            runner.OS,
		Arch:          runner.Arch,
		Hostname:      runner.Hostname,
		Ip:            runner.IP,
		Tags:          runner.Tags.Data(),
		Ephemeral:     runner.Ephemeral,
		CreatedAt:     timestamppb.New(runner.CreatedAt),
		UpdatedAt:     timestamppb.New(runner.UpdatedAt),
	}
	if runner.RegisteredAt != nil {
		out.RegisteredAt = timestamppb.New(*runner.RegisteredAt)
	}
	if runner.LastSeenAt != nil {
		out.LastSeenAt = timestamppb.New(*runner.LastSeenAt)
	}
	if runner.TerminationReason != nil {
		reason := *runner.TerminationReason
		out.TerminationReason = &reason
	}
	if runner.TerminatedAt != nil {
		out.TerminatedAt = timestamppb.New(*runner.TerminatedAt)
	}
	if runner.ProviderDeletedAt != nil {
		out.ProviderDeletedAt = timestamppb.New(*runner.ProviderDeletedAt)
	}
	return out
}
