package agents

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
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

// connTracker is a gRPC stats handler that counts open connections.
type connTracker struct {
	stats.Handler
	open atomic.Int64
}

func (c *connTracker) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context { return ctx }
func (c *connTracker) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context  { return ctx }
func (c *connTracker) HandleRPC(ctx context.Context, s stats.RPCStats)                     {}

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

func Test__ListAgentChats_ClosesConnection(t *testing.T) {
	addr, tracker := startMockAgentServer(t)

	resp, err := ListAgentChats(context.Background(), addr, "org-1", "user-1", "canvas-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Chats, 1)

	assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond, "gRPC connection should be closed after ListAgentChats")
}

func Test__DescribeAgentChat_ClosesConnection(t *testing.T) {
	addr, tracker := startMockAgentServer(t)

	resp, err := DescribeAgentChat(context.Background(), addr, "org-1", "user-1", "canvas-1", "chat-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "chat-1", resp.Chat.Id)

	assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond, "gRPC connection should be closed after DescribeAgentChat")
}

func Test__ListAgentChatMessages_ClosesConnection(t *testing.T) {
	addr, tracker := startMockAgentServer(t)

	resp, err := ListAgentChatMessages(context.Background(), addr, "org-1", "user-1", "canvas-1", "chat-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Messages, 1)

	assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond, "gRPC connection should be closed after ListAgentChatMessages")
}

func Test__ListAgentChats_ReturnsUnavailable_WhenAgentDown(t *testing.T) {
	// Use a port with no server listening
	_, err := ListAgentChats(context.Background(), "127.0.0.1:1", "org-1", "user-1", "canvas-1")
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}
