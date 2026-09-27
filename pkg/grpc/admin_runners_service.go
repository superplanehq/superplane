package grpc

import (
	"context"

	runneractions "github.com/superplanehq/superplane/pkg/grpc/actions/admin/runners"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
)

type AdminRunnersService struct {
	pb.UnimplementedRunnersServer
	actions *runneractions.Service
}

func NewAdminRunnersService(actions *runneractions.Service) *AdminRunnersService {
	return &AdminRunnersService{actions: actions}
}

func (s *AdminRunnersService) ListFleets(
	ctx context.Context,
	req *pb.ListFleetsRequest,
) (*pb.ListFleetsResponse, error) {
	return s.actions.ListFleets(ctx, req)
}

func (s *AdminRunnersService) DescribeFleet(
	ctx context.Context,
	req *pb.DescribeFleetRequest,
) (*pb.DescribeFleetResponse, error) {
	return s.actions.DescribeFleet(ctx, req)
}

func (s *AdminRunnersService) UpdateFleet(
	ctx context.Context,
	req *pb.UpdateFleetRequest,
) (*pb.UpdateFleetResponse, error) {
	return s.actions.UpdateFleet(ctx, req)
}

func (s *AdminRunnersService) GetFleetCapacity(
	ctx context.Context,
	req *pb.GetFleetCapacityRequest,
) (*pb.GetFleetCapacityResponse, error) {
	return s.actions.GetFleetCapacity(ctx, req)
}

func (s *AdminRunnersService) ListFleetTasks(
	ctx context.Context,
	req *pb.ListFleetTasksRequest,
) (*pb.ListFleetTasksResponse, error) {
	return s.actions.ListFleetTasks(ctx, req)
}

func (s *AdminRunnersService) CreateRunner(
	ctx context.Context,
	req *pb.CreateRunnerRequest,
) (*pb.CreateRunnerResponse, error) {
	return s.actions.CreateRunner(ctx, req)
}

func (s *AdminRunnersService) ListRunners(
	ctx context.Context,
	req *pb.ListRunnersRequest,
) (*pb.ListRunnersResponse, error) {
	return s.actions.ListRunners(ctx, req)
}

func (s *AdminRunnersService) DescribeRunner(
	ctx context.Context,
	req *pb.DescribeRunnerRequest,
) (*pb.DescribeRunnerResponse, error) {
	return s.actions.DescribeRunner(ctx, req)
}

func (s *AdminRunnersService) DeleteRunner(
	ctx context.Context,
	req *pb.DeleteRunnerRequest,
) (*pb.DeleteRunnerResponse, error) {
	return s.actions.DeleteRunner(ctx, req)
}
