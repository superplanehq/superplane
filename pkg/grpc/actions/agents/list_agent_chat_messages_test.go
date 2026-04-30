package agents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListAgentChatMessages(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		addr, _ := startMockAgentServer(t)

		resp, err := ListAgentChatMessages(context.Background(), addr, "org-1", "user-1", "canvas-1", "chat-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Messages, 1)
		assert.Equal(t, "msg-1", resp.Messages[0].Id)
	})

	t.Run("closes gRPC connection after call", func(t *testing.T) {
		addr, tracker := startMockAgentServer(t)

		_, err := ListAgentChatMessages(context.Background(), addr, "org-1", "user-1", "canvas-1", "chat-1")
		require.NoError(t, err)

		assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("returns unavailable when agent is down", func(t *testing.T) {
		_, err := ListAgentChatMessages(context.Background(), "127.0.0.1:1", "org-1", "user-1", "canvas-1", "chat-1")
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})
}
