package runner

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestHandleBrokerWebhookIgnoresFinishedExecution(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	//
	// A finished execution has already recorded its metadata and finished
	// timestamp. A late or duplicate webhook must not touch it again, otherwise
	// the metadata write would move the execution's finished_at timestamp.
	//
	originalMetadata := map[string]any{ExecutionMetadataBrokerTaskID: "broker-1"}
	metadata := &contexts.MetadataContext{Metadata: originalMetadata}
	state := &contexts.ExecutionStateContext{Finished: true, KVs: map[string]string{}}

	code, _, err := handleBrokerWebhook(core.WebhookRequestContext{
		Body: []byte(`{"task_id":"broker-1","status":"succeeded","exit_code":0,"task_log":{"type":"cloudwatch","cloudwatch":{"log_group_name":"g","log_stream_name":"s"}}}`),
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				Metadata:       metadata,
				ExecutionState: state,
			}, nil
		},
	}, RunnerFinishedEventType)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Metadata must be left untouched, and no new events must be emitted.
	assert.Equal(t, originalMetadata, metadata.Metadata)
	assert.Empty(t, state.Channel)
	assert.Empty(t, state.Payloads)
}

func TestHandleBrokerWebhookProcessesUnfinishedExecution(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	metadata := &contexts.MetadataContext{Metadata: map[string]any{}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	code, _, err := handleBrokerWebhook(core.WebhookRequestContext{
		Body: []byte(`{"task_id":"broker-1","status":"succeeded","exit_code":0,"task_log":{"type":"cloudwatch","cloudwatch":{"log_group_name":"g","log_stream_name":"s"}}}`),
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				Metadata:       metadata,
				ExecutionState: state,
			}, nil
		},
	}, RunnerFinishedEventType)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Metadata gets the broker task id, and the execution finishes on the passed channel.
	updated, ok := metadata.Metadata.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "broker-1", updated[ExecutionMetadataBrokerTaskID])
	assert.True(t, state.IsFinished())
	assert.Equal(t, PassedOutputChannel, state.Channel)
}
