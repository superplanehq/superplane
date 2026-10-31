package organizations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	lifecycle "github.com/superplanehq/superplane/pkg/grpc/actions/runners"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	adminpb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RunnerService adapts the organization contract to the fleet lifecycle.
// Authorization supplies the domain; request bodies cannot change ownership.
type RunnerService struct{ lifecycle *lifecycle.Service }

func NewRunnerService(service *lifecycle.Service) *RunnerService {
	return &RunnerService{lifecycle: service}
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

func (s *RunnerService) scoped(ctx context.Context, organizationRef, slug string) (*lifecycle.Service, error) {
	fleet, err := s.fleet(ctx, organizationRef, slug)
	if err != nil {
		return nil, err
	}
	return s.lifecycle.ForFleet(fleet), nil
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
	out := &pb.ListManagedRunnerFleetsResponse{Fleets: []*pb.ManagedRunnerFleet{}}
	for i := range fleets {
		out.Fleets = append(out.Fleets, managedFleet(lifecycle.SerializeFleet(&fleets[i])))
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
	return &pb.CreateRunnerFleetResponse{Fleet: managedFleet(lifecycle.SerializeFleet(fleet))}, nil
}

func (s *RunnerService) DescribeRunnerFleet(ctx context.Context, req *pb.DescribeRunnerFleetRequest) (*pb.DescribeRunnerFleetResponse, error) {
	fleet, err := s.fleet(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	return &pb.DescribeRunnerFleetResponse{Fleet: managedFleet(lifecycle.SerializeFleet(fleet))}, nil
}

func (s *RunnerService) UpdateRunnerFleet(ctx context.Context, req *pb.UpdateRunnerFleetRequest) (*pb.UpdateRunnerFleetResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.UpdateFleet(ctx, &adminpb.UpdateFleetRequest{FleetId: req.FleetId, RunnerVersion: req.RunnerVersion, Enabled: req.Enabled, Spec: lifecycleSpec(req.Spec)})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRunnerFleetResponse{Fleet: managedFleet(out.Fleet)}, nil
}

func (s *RunnerService) GetRunnerFleetCapacity(ctx context.Context, req *pb.GetRunnerFleetCapacityRequest) (*pb.GetRunnerFleetCapacityResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.GetFleetCapacity(ctx, &adminpb.GetFleetCapacityRequest{FleetId: req.FleetId, Generation: req.Generation, WaitSeconds: req.WaitSeconds})
	if err != nil {
		return nil, err
	}
	return &pb.GetRunnerFleetCapacityResponse{RunnableTasks: out.RunnableTasks, PendingRunners: out.PendingRunners,
		IdleRunners: out.IdleRunners, BusyRunners: out.BusyRunners, TerminatedRunners: out.TerminatedRunners, Generation: out.Generation}, nil
}

func (s *RunnerService) CreateFleetRunner(ctx context.Context, req *pb.CreateFleetRunnerRequest) (*pb.CreateFleetRunnerResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.CreateRunner(ctx, &adminpb.CreateRunnerRequest{FleetId: req.FleetId, TaskId: req.TaskId,
		IdempotencyKey: req.IdempotencyKey, Ephemeral: req.Ephemeral, FleetManagerId: req.FleetManagerId, Provider: req.Provider})
	if err != nil {
		return nil, err
	}
	return &pb.CreateFleetRunnerResponse{Runner: organizationRunner(out.Runner), RegistrationToken: out.RegistrationToken,
		RegistrationExpiresAt: out.RegistrationExpiresAt, RunnerApiUrl: out.RunnerApiUrl, DisplayName: out.DisplayName}, nil
}

func (s *RunnerService) ListFleetRunners(ctx context.Context, req *pb.ListFleetRunnersRequest) (*pb.ListFleetRunnersResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.ListRunners(ctx, &adminpb.ListRunnersRequest{FleetId: req.FleetId, States: req.States, Limit: req.Limit, AfterId: req.AfterId})
	if err != nil {
		return nil, err
	}
	response := &pb.ListFleetRunnersResponse{Runners: []*pb.OrganizationRunner{}, TotalCount: out.TotalCount, HasNextPage: out.HasNextPage}
	for _, runner := range out.Runners {
		response.Runners = append(response.Runners, organizationRunner(runner))
	}
	return response, nil
}

func (s *RunnerService) DescribeFleetRunner(ctx context.Context, req *pb.DescribeFleetRunnerRequest) (*pb.DescribeFleetRunnerResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.DescribeRunner(ctx, &adminpb.DescribeRunnerRequest{FleetId: req.FleetId, RunnerId: req.RunnerId})
	if err != nil {
		return nil, err
	}
	return &pb.DescribeFleetRunnerResponse{Runner: organizationRunner(out.Runner)}, nil
}

func (s *RunnerService) TerminateFleetRunner(ctx context.Context, req *pb.TerminateFleetRunnerRequest) (*pb.TerminateFleetRunnerResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.DeleteRunner(ctx, &adminpb.DeleteRunnerRequest{FleetId: req.FleetId, RunnerId: req.RunnerId, States: req.States})
	if err != nil {
		return nil, err
	}
	return &pb.TerminateFleetRunnerResponse{Runner: organizationRunner(out.Runner)}, nil
}

func (s *RunnerService) MarkFleetRunnerProviderDeleted(ctx context.Context, req *pb.MarkFleetRunnerProviderDeletedRequest) (*pb.MarkFleetRunnerProviderDeletedResponse, error) {
	service, err := s.scoped(ctx, req.GetId(), req.GetFleetId())
	if err != nil {
		return nil, err
	}
	out, err := service.MarkRunnerProviderDeleted(ctx, &adminpb.MarkRunnerProviderDeletedRequest{FleetId: req.FleetId,
		RunnerId: req.RunnerId, FleetManagerId: req.FleetManagerId, Provider: req.Provider})
	if err != nil {
		return nil, err
	}
	return &pb.MarkFleetRunnerProviderDeletedResponse{Runner: organizationRunner(out.Runner)}, nil
}

func lifecycleSpec(spec *pb.OrganizationRunnerFleetSpec) *adminpb.FleetSpec {
	if spec == nil {
		return nil
	}
	return &adminpb.FleetSpec{OperatingSystem: spec.OperatingSystem, Architecture: spec.Architecture,
		CpuMillicores: spec.CpuMillicores, MemoryMb: spec.MemoryMb, DiskGb: spec.DiskGb, Capabilities: spec.Capabilities}
}

func managedFleet(fleet *adminpb.Fleet) *pb.ManagedRunnerFleet {
	spec := fleet.GetSpec()
	return &pb.ManagedRunnerFleet{Id: fleet.Id, Enabled: fleet.Enabled, RunnerVersion: fleet.RunnerVersion,
		CreatedAt: fleet.CreatedAt, UpdatedAt: fleet.UpdatedAt, Spec: &pb.OrganizationRunnerFleetSpec{
			OperatingSystem: spec.GetOperatingSystem(), Architecture: spec.GetArchitecture(), CpuMillicores: spec.GetCpuMillicores(),
			MemoryMb: spec.GetMemoryMb(), DiskGb: spec.GetDiskGb(), Capabilities: spec.GetCapabilities()}}
}

func organizationRunner(runner *adminpb.Runner) *pb.OrganizationRunner {
	return &pb.OrganizationRunner{Id: runner.Id, FleetId: runner.FleetId, State: runner.State, RunnerVersion: runner.RunnerVersion,
		RegisteredAt: runner.RegisteredAt, LastSeenAt: runner.LastSeenAt, TerminationReason: runner.TerminationReason,
		CreatedAt: runner.CreatedAt, UpdatedAt: runner.UpdatedAt, TerminatedAt: runner.TerminatedAt, Ephemeral: runner.Ephemeral,
		Tags: runner.Tags, Os: runner.Os, Arch: runner.Arch, Hostname: runner.Hostname, Ip: runner.Ip, ProviderDeletedAt: runner.ProviderDeletedAt}
}
