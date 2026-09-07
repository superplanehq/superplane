package runner

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestParseRunnerLLMUsageFromClaudeCodeResult(t *testing.T) {
	t.Parallel()

	result := json.RawMessage(`{
		"type": "result",
		"model": "claude-sonnet-4-6",
		"usage": {
			"input_tokens": 1200,
			"output_tokens": 340,
			"cache_read_input_tokens": 10,
			"cache_creation_input_tokens": 20
		},
		"total_cost_usd": 0.0123
	}`)
	configuration := map[string]any{
		"model": "sonnet",
		"credentials": map[string]any{
			"source": CredentialsSourceHosted,
		},
	}

	record, ok := ParseRunnerLLMUsage(models.UsageProviderAnthropic, configuration, result)
	require.True(t, ok)
	assert.Equal(t, models.UsageProviderAnthropic, record.Provider)
	assert.Equal(t, "claude-sonnet-4-6", record.Model)
	assert.Equal(t, int64(1200), record.InputTokens)
	assert.Equal(t, int64(340), record.OutputTokens)
	assert.Equal(t, int64(10), record.CacheReadTokens)
	assert.Equal(t, int64(20), record.CacheWriteTokens)
	assert.Equal(t, int64(1570), record.TotalTokens)
	require.NotNil(t, record.CostMicros)
	assert.Equal(t, int64(12300), *record.CostMicros)
	assert.Equal(t, "hosted", record.FundingSource)
}

func TestParseRunnerLLMUsageBYOKUsesConfigurationModel(t *testing.T) {
	t.Parallel()

	result := json.RawMessage(`{"usage":{"prompt_tokens":11,"completion_tokens":7}}`)
	configuration := map[string]any{
		"model": "gpt-5",
		"credentials": map[string]any{
			"source": CredentialsSourceSecret,
		},
	}

	record, ok := ParseRunnerLLMUsage(models.UsageProviderOpenAI, configuration, result)
	require.True(t, ok)
	assert.Equal(t, "gpt-5", record.Model)
	assert.Equal(t, int64(11), record.InputTokens)
	assert.Equal(t, int64(7), record.OutputTokens)
	assert.Equal(t, "byok", record.FundingSource)
}

func TestParseRunnerLLMUsageSkipsEmptyResult(t *testing.T) {
	t.Parallel()

	_, ok := ParseRunnerLLMUsage(models.UsageProviderOpenRouter, map[string]any{}, json.RawMessage(`{"status":"ok"}`))
	assert.False(t, ok)
}

func TestParseRunnerLLMUsageFromMergedPlanResult(t *testing.T) {
	t.Parallel()

	result := json.RawMessage(`{
		"plan": "cGxhbg==",
		"model": "google/gemini-3.7-flash",
		"usage": {
			"input_tokens": 800,
			"output_tokens": 120,
			"cache_read_input_tokens": 4,
			"reasoning_tokens": 10
		},
		"total_cost_usd": 0.0025
	}`)
	configuration := map[string]any{
		"model": "google/gemini-3.7-flash",
		"credentials": map[string]any{
			"source": CredentialsSourceHosted,
		},
	}

	record, ok := ParseRunnerLLMUsage(models.UsageProviderOpenRouter, configuration, result)
	require.True(t, ok)
	assert.Equal(t, models.UsageProviderOpenRouter, record.Provider)
	assert.Equal(t, "google/gemini-3.7-flash", record.Model)
	assert.Equal(t, int64(800), record.InputTokens)
	assert.Equal(t, int64(120), record.OutputTokens)
	assert.Equal(t, int64(4), record.CacheReadTokens)
	assert.Equal(t, int64(10), record.ReasoningTokens)
	assert.Equal(t, int64(934), record.TotalTokens)
	require.NotNil(t, record.CostMicros)
	assert.Equal(t, int64(2500), *record.CostMicros)
	assert.Equal(t, "hosted", record.FundingSource)
}

type recordingUsage struct {
	records []core.UsageRecord
}

func (r *recordingUsage) Record(record core.UsageRecord) error {
	r.records = append(r.records, record)
	return nil
}

func (r *recordingUsage) RecordCompute(core.ComputeUsageRecord) error {
	return nil
}

func TestRecordRunnerLLMUsageFromFinishedEvent(t *testing.T) {
	t.Parallel()

	recorder := &recordingUsage{}
	RecordRunnerLLMUsage(
		recorder,
		nil,
		"runnerClaudeCode.finished",
		map[string]any{"credentials": map[string]any{"source": "hosted"}, "model": "sonnet"},
		json.RawMessage(`{"usage":{"input_tokens":5,"output_tokens":2}}`),
	)
	require.Len(t, recorder.records, 1)
	assert.Equal(t, models.UsageProviderAnthropic, recorder.records[0].Provider)
	assert.Equal(t, "hosted", recorder.records[0].FundingSource)
	assert.Equal(t, models.UsageIdempotencyKeyRunner, recorder.records[0].IdempotencyKey)
}

func TestRecordRunnerLLMUsageFromSuperPlaneFinishedEvent(t *testing.T) {
	t.Parallel()

	recorder := &recordingUsage{}
	RecordRunnerLLMUsage(
		recorder,
		nil,
		"runnerSuperPlane.finished",
		map[string]any{
			"hostedProvider": models.UsageProviderOpenRouter,
			"model":          "anthropic/claude-sonnet-4-6",
			"credentials":    map[string]any{"source": "hosted"},
		},
		json.RawMessage(`{"usage":{"input_tokens":5,"output_tokens":2},"model":"anthropic/claude-sonnet-4-6"}`),
	)
	require.Len(t, recorder.records, 1)
	assert.Equal(t, models.UsageProviderOpenRouter, recorder.records[0].Provider)
	assert.Equal(t, "hosted", recorder.records[0].FundingSource)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", recorder.records[0].Model)
}

func TestProcessBrokerTaskStatusRecordsUsageWhenExecutionAlreadyFinished(t *testing.T) {
	t.Parallel()

	recorder := &recordingUsage{}
	state := &contexts.ExecutionStateContext{Finished: true}
	exit := 0
	task := &Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"usage":{"input_tokens":1200,"output_tokens":80},"model":"claude-sonnet-4-6"}`),
	}

	require.NoError(t, processBrokerTaskStatus(
		state,
		task,
		"runnerClaudeCode.finished",
		"",
		nil,
		recorder,
		map[string]any{"credentials": map[string]any{"source": "hosted"}},
	))
	require.Len(t, recorder.records, 1)
	assert.Equal(t, int64(1280), recorder.records[0].TotalTokens)
	assert.Empty(t, state.Payloads)
}
