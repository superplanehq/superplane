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

func Test__DeleteAgentChat(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		addr, _ := startMockAgentServer(t)

		resp, err := DeleteAgentChat(context.Background(), addr, "org-1", "user-1", "canvas-1", "chat-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("returns not found when chat does not exist", func(t *testing.T) {
		addr, _ := startMockAgentServer(t)

		_, err := DeleteAgentChat(context.Background(), addr, "org-1", "user-1", "canvas-1", "not-found")
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("closes gRPC connection after call", func(t *testing.T) {
		addr, tracker := startMockAgentServer(t)

		_, err := DeleteAgentChat(context.Background(), addr, "org-1", "user-1", "canvas-1", "chat-1")
		require.NoError(t, err)

		assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("returns unavailable when agent is down", func(t *testing.T) {
		_, err := DeleteAgentChat(context.Background(), "127.0.0.1:1", "org-1", "user-1", "canvas-1", "chat-1")
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})
}
