package agents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListAgentChats(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		addr, _ := startMockAgentServer(t)

		resp, err := ListAgentChats(context.Background(), addr, "org-1", "user-1", "canvas-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Chats, 1)
		assert.Equal(t, "chat-1", resp.Chats[0].Id)
	})

	t.Run("closes gRPC connection after call", func(t *testing.T) {
		addr, tracker := startMockAgentServer(t)

		_, err := ListAgentChats(context.Background(), addr, "org-1", "user-1", "canvas-1")
		require.NoError(t, err)

		assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("returns unavailable when agent is down", func(t *testing.T) {
		_, err := ListAgentChats(context.Background(), "127.0.0.1:1", "org-1", "user-1", "canvas-1")
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})
}

func Test__SerializeChatUsage(t *testing.T) {
	t.Run("returns nil for nil input", func(t *testing.T) {
		assert.Nil(t, serializeChatUsage(nil))
	})

	t.Run("maps all fields", func(t *testing.T) {
		in := &internalpb.ChatUsage{
			TotalInputTokens:      10,
			TotalOutputTokens:     20,
			TotalTokens:           30,
			TotalEstimatedCostUsd: 0.01,
		}
		out := serializeChatUsage(in)
		require.NotNil(t, out)
		assert.Equal(t, int64(10), out.TotalInputTokens)
		assert.Equal(t, int64(20), out.TotalOutputTokens)
		assert.Equal(t, int64(30), out.TotalTokens)
		assert.InDelta(t, 0.01, out.TotalEstimatedCostUsd, 0.001)
	})
}

func Test__SerializeAgentChats(t *testing.T) {
	t.Run("skips nil entries", func(t *testing.T) {
		result := serializeAgentChats([]*internalpb.ChatInfo{nil, {Id: "chat-1"}})
		require.Len(t, result, 1)
		assert.Equal(t, "chat-1", result[0].Id)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := serializeAgentChats(nil)
		assert.Empty(t, result)
	})
}
