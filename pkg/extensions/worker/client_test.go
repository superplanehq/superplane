package worker

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

func TestClientRespondsToPing(t *testing.T) {
	t.Parallel()

	conn := &fakeMessageConn{}
	client := NewClient(ClientConfig{}, nil)
	client.currentConn = conn

	err := client.handleMessage(context.Background(), []byte(`{"type":"ping"}`))
	require.NoError(t, err)

	messages := conn.JSONMessages()
	require.Len(t, messages, 1)

	var pong protocol.PongMessage
	require.NoError(t, json.Unmarshal(messages[0], &pong))
	require.Equal(t, protocol.MessageTypePong, pong.Type)
}

func TestClientProcessesAssignedJobs(t *testing.T) {
	t.Parallel()

	conn := &fakeMessageConn{}
	client := NewClient(ClientConfig{}, func(_ context.Context, message protocol.JobAssignMessage) (json.RawMessage, error) {
		require.Equal(t, "job-1", message.JobID)
		return json.RawMessage(`{"handled":true}`), nil
	})
	client.currentConn = conn

	err := client.handleMessage(context.Background(), []byte(`{"type":"job.assign","jobId":"job-1","extensionId":"ext-1","versionId":"ver-1"}`))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(conn.JSONMessages()) == 1
	}, time.Second, 10*time.Millisecond)

	messages := conn.JSONMessages()

	var result protocol.JobCompleteMessage
	require.NoError(t, json.Unmarshal(messages[0], &result))
	require.Equal(t, protocol.MessageTypeJobComplete, result.Type)
	require.Equal(t, "job-1", result.JobID)
	require.True(t, result.Success)
	require.JSONEq(t, `{"handled":true}`, string(result.Output))
}

type fakeMessageConn struct {
	mu           sync.Mutex
	jsonMessages [][]byte
	closed       bool
}

func (c *fakeMessageConn) WriteJSON(v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.jsonMessages = append(c.jsonMessages, payload)
	return nil
}

func (c *fakeMessageConn) WriteMessage(_ int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jsonMessages = append(c.jsonMessages, append([]byte(nil), data...))
	return nil
}

func (c *fakeMessageConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *fakeMessageConn) JSONMessages() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make([][]byte, 0, len(c.jsonMessages))
	for _, message := range c.jsonMessages {
		messages = append(messages, append([]byte(nil), message...))
	}

	return messages
}

var _ messageConn = (*fakeMessageConn)(nil)
