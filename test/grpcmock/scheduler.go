package grpcmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/superplanehq/superplane/pkg/protos/periodic_scheduler"
	"github.com/superplanehq/superplane/pkg/protos/status"
	"google.golang.org/genproto/googleapis/rpc/code"
)

type SchedulerService struct {
	LastRunNowRequest *pb.RunNowRequest
}

func NewSchedulerService() *SchedulerService {
	return &SchedulerService{}
}

func (s *SchedulerService) GetLastRunNowRequest() *pb.RunNowRequest {
	return s.LastRunNowRequest
}

func (s *SchedulerService) RunNow(ctx context.Context, request *pb.RunNowRequest) (*pb.RunNowResponse, error) {
	s.LastRunNowRequest = request
	return &pb.RunNowResponse{
		Status: &status.Status{
			Code:    code.Code_OK,
			Message: "OK",
		},
		Trigger: &pb.Trigger{
			ScheduledWorkflowId: uuid.New().String(),
		},
	}, nil
}

func (s *SchedulerService) Apply(ctx context.Context, in *pb.ApplyRequest) (*pb.ApplyResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SchedulerService) Persist(ctx context.Context, in *pb.PersistRequest) (*pb.PersistResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *SchedulerService) Pause(ctx context.Context, in *pb.PauseRequest) (*pb.PauseResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SchedulerService) Unpause(ctx context.Context, in *pb.UnpauseRequest) (*pb.UnpauseResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SchedulerService) Describe(ctx context.Context, in *pb.DescribeRequest) (*pb.DescribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SchedulerService) LatestTriggers(ctx context.Context, in *pb.LatestTriggersRequest) (*pb.LatestTriggersResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SchedulerService) History(ctx context.Context, in *pb.HistoryRequest) (*pb.HistoryResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SchedulerService) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *SchedulerService) ListKeyset(ctx context.Context, in *pb.ListKeysetRequest) (*pb.ListKeysetResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *SchedulerService) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *SchedulerService) GetProjectId(ctx context.Context, in *pb.GetProjectIdRequest) (*pb.GetProjectIdResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *SchedulerService) Version(ctx context.Context, in *pb.VersionRequest) (*pb.VersionResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
