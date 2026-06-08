package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMintAndValidateLiveLogStreamToken(t *testing.T) {
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")
	now := time.Now()

	token, expiresAt, err := MintLiveLogStreamToken("task-123", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(liveLogStreamTokenTTL), expiresAt)

	err = ValidateLiveLogStreamToken(token, "task-123", "live-log-secret")
	require.NoError(t, err)

	err = ValidateLiveLogStreamToken(token, "other-task", "live-log-secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task id mismatch")
}

func TestNewLiveLogSession(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

	session, err := NewLiveLogSession("task-abc", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "https://broker.example/v1/tasks/task-abc/live-logs", session.StreamURL)
	assert.NotEmpty(t, session.Token)
	assert.False(t, session.ExpiresAt.IsZero())
}

func TestBrokerTaskIDFromExecutionMetadata(t *testing.T) {
	assert.Equal(t, "tb-1", BrokerTaskIDFromExecutionMetadata(map[string]any{
		ExecutionMetadataBrokerTaskID: "tb-1",
	}))
	assert.Equal(t, "99", BrokerTaskIDFromExecutionMetadata(map[string]any{
		ExecutionMetadataBrokerTaskID: 99,
	}))
	assert.Equal(t, "", BrokerTaskIDFromExecutionMetadata(map[string]any{}))
}
