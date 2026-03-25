package agentservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/config"
	pb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultTimeout = 5 * time.Second

var ErrAgentServiceDisabled = errors.New("agent service is disabled")

type Service interface {
	Enabled() bool
	CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.CreateAgentResponse, error)
	ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error)
	DescribeAgent(ctx context.Context, req *pb.DescribeAgentRequest) (*pb.DescribeAgentResponse, error)
	ListAgentMessages(ctx context.Context, req *pb.ListAgentMessagesRequest) (*pb.ListAgentMessagesResponse, error)
}

type disabledService struct{}

type grpcService struct {
	client pb.AgentsClient
}

func NewServiceFromEnv() (Service, error) {
	url := config.AgentGRPCURL()
	if url == "" {
		return disabledService{}, nil
	}

	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create agent grpc client: %w", err)
	}

	return &grpcService{
		client: pb.NewAgentsClient(conn),
	}, nil
}

func (disabledService) Enabled() bool {
	return false
}

func (disabledService) CreateAgent(context.Context, *pb.CreateAgentRequest) (*pb.CreateAgentResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (disabledService) ListAgents(context.Context, *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (disabledService) DescribeAgent(context.Context, *pb.DescribeAgentRequest) (*pb.DescribeAgentResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (disabledService) ListAgentMessages(context.Context, *pb.ListAgentMessagesRequest) (*pb.ListAgentMessagesResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (s *grpcService) Enabled() bool {
	return true
}

func (s *grpcService) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.CreateAgentResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.CreateAgent(callCtx, req)
}

func (s *grpcService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.ListAgents(callCtx, req)
}

func (s *grpcService) DescribeAgent(ctx context.Context, req *pb.DescribeAgentRequest) (*pb.DescribeAgentResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.DescribeAgent(callCtx, req)
}

func (s *grpcService) ListAgentMessages(ctx context.Context, req *pb.ListAgentMessagesRequest) (*pb.ListAgentMessagesResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.ListAgentMessages(callCtx, req)
}
