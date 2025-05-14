package grpcmock

import (
	"context"
	"fmt"

	pb "github.com/superplanehq/superplane/pkg/protos/plumber.pipeline"
)

type PipelineService struct {
	MockedPipelineResult pb.Pipeline_Result
	MockedWorkflowID     string
}

func NewPipelineService() *PipelineService {
	return &PipelineService{}
}

func (s *PipelineService) MockPipelineResult(result pb.Pipeline_Result) {
	s.MockedPipelineResult = result
}

func (s *PipelineService) MockWorkflow(id string) {
	s.MockedWorkflowID = id
}

func (s *PipelineService) Describe(ctx context.Context, request *pb.DescribeRequest) (*pb.DescribeResponse, error) {
	return &pb.DescribeResponse{
		Pipeline: &pb.Pipeline{
			WfId:   s.MockedWorkflowID,
			Result: s.MockedPipelineResult,
		},
	}, nil
}

func (s *PipelineService) DescribeMany(ctx context.Context, request *pb.DescribeManyRequest) (*pb.DescribeManyResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) DescribeTopology(ctx context.Context, request *pb.DescribeTopologyRequest) (*pb.DescribeTopologyResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) GetProjectId(ctx context.Context, request *pb.GetProjectIdRequest) (*pb.GetProjectIdResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) List(ctx context.Context, request *pb.ListRequest) (*pb.ListResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ListActivity(ctx context.Context, request *pb.ListActivityRequest) (*pb.ListActivityResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ListGrouped(ctx context.Context, request *pb.ListGroupedRequest) (*pb.ListGroupedResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ListKeyset(ctx context.Context, request *pb.ListKeysetRequest) (*pb.ListKeysetResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ListQueues(ctx context.Context, request *pb.ListQueuesRequest) (*pb.ListQueuesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ListRequesters(ctx context.Context, request *pb.ListRequestersRequest) (*pb.ListRequestersResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) PartialRebuild(ctx context.Context, request *pb.PartialRebuildRequest) (*pb.PartialRebuildResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) RunNow(ctx context.Context, request *pb.RunNowRequest) (*pb.RunNowResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) Schedule(ctx context.Context, request *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ScheduleExtension(ctx context.Context, request *pb.ScheduleExtensionRequest) (*pb.ScheduleExtensionResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) Terminate(ctx context.Context, request *pb.TerminateRequest) (*pb.TerminateResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) ValidateYaml(ctx context.Context, request *pb.ValidateYamlRequest) (*pb.ValidateYamlResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PipelineService) Delete(ctx context.Context, request *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
