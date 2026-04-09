package agents

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockAgentsServer implements the internal AgentsServer interface for testing.
type mockAgentsServer struct {
	internalpb.UnimplementedAgentsServer
}

func (m *mockAgentsServer) ListAgentChats(ctx context.Context, req *internalpb.ListAgentChatsRequest) (*internalpb.ListAgentChatsResponse, error) {
	return &internalpb.ListAgentChatsResponse{
		Chats: []*internalpb.ChatInfo{
			{Id: "chat-1", InitialMessage: "hello", CreatedAt: timestamppb.Now()},
		},
	}, nil
}

func (m *mockAgentsServer) DescribeAgentChat(ctx context.Context, req *internalpb.DescribeAgentChatRequest) (*internalpb.DescribeAgentChatResponse, error) {
	return &internalpb.DescribeAgentChatResponse{
		Chat: &internalpb.ChatInfo{Id: req.ChatId, InitialMessage: "hello", CreatedAt: timestamppb.Now()},
	}, nil
}

func (m *mockAgentsServer) ListAgentChatMessages(ctx context.Context, req *internalpb.ListAgentChatMessagesRequest) (*internalpb.ListAgentChatMessagesResponse, error) {
	return &internalpb.ListAgentChatMessagesResponse{
		Messages: []*internalpb.AgentChatMessage{
			{Id: "msg-1", Role: "user", Content: "hello"},
		},
	}, nil
}

func (m *mockAgentsServer) DescribeOrganizationAgentUsage(ctx context.Context, req *internalpb.DescribeOrganizationAgentUsageRequest) (*internalpb.DescribeOrganizationAgentUsageResponse, error) {
	return &internalpb.DescribeOrganizationAgentUsageResponse{
		Usage: &internalpb.ChatUsage{
			TotalInputTokens:      100,
			TotalOutputTokens:     200,
			TotalTokens:           300,
			TotalEstimatedCostUsd: 0.05,
		},
	}, nil
}

// connTracker is a gRPC stats handler that counts open connections.
type connTracker struct {
	stats.Handler
	open atomic.Int64
}

func (c *connTracker) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}
func (c *connTracker) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context { return ctx }
func (c *connTracker) HandleRPC(ctx context.Context, s stats.RPCStats)                    {}

func (c *connTracker) HandleConn(ctx context.Context, s stats.ConnStats) {
	switch s.(type) {
	case *stats.ConnBegin:
		c.open.Add(1)
	case *stats.ConnEnd:
		c.open.Add(-1)
	}
}

// startMockAgentServer creates a gRPC server with the mock agent service
// and returns the listener address and a connection tracker.
func startMockAgentServer(t *testing.T) (string, *connTracker) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tracker := &connTracker{}
	server := grpc.NewServer(grpc.StatsHandler(tracker))
	internalpb.RegisterAgentsServer(server, &mockAgentsServer{})

	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.GracefulStop)

	return lis.Addr().String(), tracker
}
