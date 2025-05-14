package grpcmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/superplanehq/superplane/pkg/protos/repo_proxy"
)

type RepoProxyService struct {
	LastCreateRequest *pb.CreateRequest
}

func NewRepoProxyService() *RepoProxyService {
	return &RepoProxyService{}
}

func (s *RepoProxyService) Create(ctx context.Context, request *pb.CreateRequest) (*pb.CreateResponse, error) {
	s.LastCreateRequest = request
	return &pb.CreateResponse{
		WorkflowId: uuid.New().String(),
		PipelineId: uuid.New().String(),
	}, nil
}

func (s RepoProxyService) GetLastCreateRequest() *pb.CreateRequest {
	return s.LastCreateRequest
}

func (s RepoProxyService) CreateBlank(context.Context, *pb.CreateBlankRequest) (*pb.CreateBlankResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s RepoProxyService) Describe(context.Context, *pb.DescribeRequest) (*pb.DescribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s RepoProxyService) DescribeMany(context.Context, *pb.DescribeManyRequest) (*pb.DescribeManyResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s RepoProxyService) ListBlockedHooks(context.Context, *pb.ListBlockedHooksRequest) (*pb.ListBlockedHooksResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s RepoProxyService) ScheduleBlockedHook(context.Context, *pb.ScheduleBlockedHookRequest) (*pb.ScheduleBlockedHookResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
