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

func Test__DescribeAgentUsage(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		addr, _ := startMockAgentServer(t)

		resp, err := DescribeAgentUsage(context.Background(), addr, "org-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Usage)
		assert.Equal(t, int64(100), resp.Usage.TotalInputTokens)
		assert.Equal(t, int64(200), resp.Usage.TotalOutputTokens)
		assert.Equal(t, int64(300), resp.Usage.TotalTokens)
		assert.InDelta(t, 0.05, resp.Usage.TotalEstimatedCostUsd, 0.001)
	})

	t.Run("closes gRPC connection after call", func(t *testing.T) {
		addr, tracker := startMockAgentServer(t)

		_, err := DescribeAgentUsage(context.Background(), addr, "org-1")
		require.NoError(t, err)

		assert.Eventually(t, func() bool { return tracker.open.Load() == 0 }, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("returns unavailable when agent is down", func(t *testing.T) {
		_, err := DescribeAgentUsage(context.Background(), "127.0.0.1:1", "org-1")
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})
}
