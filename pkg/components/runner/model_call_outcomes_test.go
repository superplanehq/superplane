package runner

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestParseRunnerModelCallOutcomesSkipsEmptyModels(t *testing.T) {
	occurredAt, stamp := rfc3339Stamp(t, -time.Minute)
	payload, err := json.Marshal(map[string]any{
		"modelCallOutcomes": []map[string]any{
			{"model": "openrouter/x-ai/grok-4.6", "outcome": "attempt_2", "occurredAt": stamp},
			{"model": "", "outcome": "error"},
		},
	})
	require.NoError(t, err)
	inputs := parseRunnerModelCallOutcomes(models.UsageProviderOpenRouter, "task-1", payload)
	require.Len(t, inputs, 1)
	assert.Equal(t, "x-ai/grok-4.6", inputs[0].Model)
	assert.Equal(t, models.HostedModelCallOutcomeAttempt2, inputs[0].Outcome)
	assert.Equal(t, 1, inputs[0].Sequence)
	assert.True(t, inputs[0].OccurredAt.UTC().Equal(occurredAt))
}

func TestParseRunnerModelCallOutcomesLeavesMissingOccurredAtZero(t *testing.T) {
	inputs := parseRunnerModelCallOutcomes(
		models.UsageProviderAnthropic,
		"task-1",
		json.RawMessage(`{"modelCallOutcomes":[{"model":"claude-sonnet-4-6","outcome":"attempt_1"}]}`),
	)
	require.Len(t, inputs, 1)
	assert.True(t, inputs[0].OccurredAt.IsZero())
}

func TestRecordRunnerModelCallOutcomesStoresHostedResultsOnce(t *testing.T) {
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id = ?", "task-hosted-1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	result := json.RawMessage(`{"modelCallOutcomes":[{"model":"claude-sonnet-4-6","outcome":"attempt_1"}]}`)
	configuration := map[string]any{
		"hostedProvider": models.UsageProviderAnthropic,
		"credentials":    map[string]any{"source": CredentialsSourceHosted},
	}
	RecordRunnerModelCallOutcomes(nil, "task-hosted-1", "runnerSuperPlane.finished", configuration, result)
	RecordRunnerModelCallOutcomes(nil, "task-hosted-1", "runnerSuperPlane.finished", configuration, result)

	var rows []models.HostedModelCallOutcome
	require.NoError(t, db.Where("runner_task_id = ?", "task-hosted-1").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, models.UsageProviderAnthropic, rows[0].Provider)
	assert.Equal(t, "claude-sonnet-4-6", rows[0].Model)
	assert.Equal(t, models.HostedModelCallOutcomeAttempt1, rows[0].Outcome)
}

func TestRecordRunnerModelCallOutcomesIgnoresCustomerKeys(t *testing.T) {
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id = ?", "task-byok-1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	RecordRunnerModelCallOutcomes(
		nil,
		"task-byok-1",
		"runnerOpenRouter.finished",
		map[string]any{"credentials": map[string]any{"source": "secret"}},
		json.RawMessage(`{"modelCallOutcomes":[{"model":"x-ai/grok-4.6","outcome":"attempt_1"}]}`),
	)

	var count int64
	require.NoError(t, db.Model(&models.HostedModelCallOutcome{}).Where("runner_task_id = ?", "task-byok-1").Count(&count).Error)
	assert.Zero(t, count)
}

func TestProcessTaskStatusRecordsHostedModelCallOutcomes(t *testing.T) {
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id = ?", "task-finish-1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	exit := 0
	occurredAt, stamp := rfc3339Stamp(t, -time.Minute)
	payload, err := json.Marshal(map[string]any{
		"modelCallOutcomes": []map[string]any{
			{"model": "gpt-5", "outcome": "error", "occurredAt": stamp},
		},
	})
	require.NoError(t, err)
	task := &core.RunnerTask{
		ID:       "task-finish-1",
		State:    "succeeded",
		ExitCode: &exit,
		Result:   payload,
	}
	require.NoError(t, processTaskStatus(
		&contexts.ExecutionStateContext{Finished: true},
		task,
		"runnerSuperPlane.finished",
		"",
		nil,
		nil,
		map[string]any{
			"hostedProvider": models.UsageProviderOpenAI,
			"credentials":    map[string]any{"source": CredentialsSourceHosted},
		},
	))

	var rows []models.HostedModelCallOutcome
	require.NoError(t, db.Where("runner_task_id = ?", "task-finish-1").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, models.UsageProviderOpenAI, rows[0].Provider)
	assert.Equal(t, "gpt-5", rows[0].Model)
	assert.Equal(t, models.HostedModelCallOutcomeError, rows[0].Outcome)
	assert.True(t, rows[0].OccurredAt.UTC().Equal(occurredAt))
}

func rfc3339Stamp(t *testing.T, delta time.Duration) (time.Time, string) {
	t.Helper()
	stamp := time.Now().UTC().Add(delta).Truncate(time.Millisecond)
	return stamp, stamp.Format(time.RFC3339Nano)
}
