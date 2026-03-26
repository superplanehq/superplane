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
	CreateAgentChat(ctx context.Context, req *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error)
	ListAgentChats(ctx context.Context, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error)
	DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error)
	ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error)
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

func (disabledService) CreateAgentChat(context.Context, *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (disabledService) ListAgentChats(context.Context, *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (disabledService) DescribeAgentChat(context.Context, *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (disabledService) ListAgentChatMessages(context.Context, *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	return nil, ErrAgentServiceDisabled
}

func (s *grpcService) Enabled() bool {
	return true
}

func (s *grpcService) CreateAgentChat(ctx context.Context, req *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.CreateAgentChat(callCtx, req)
}

func (s *grpcService) ListAgentChats(ctx context.Context, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.ListAgentChats(callCtx, req)
}

func (s *grpcService) DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.DescribeAgentChat(callCtx, req)
}

func (s *grpcService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.ListAgentChatMessages(callCtx, req)
}
