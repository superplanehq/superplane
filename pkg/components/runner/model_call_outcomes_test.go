package runner

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func syncModelCallOutcomeInserts(t *testing.T) {
	t.Helper()
	previous := runModelCallOutcomeInsert
	runModelCallOutcomeInsert = func(fn func()) { fn() }
	t.Cleanup(func() { runModelCallOutcomeInsert = previous })
}

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
	syncModelCallOutcomeInserts(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id = ?", "task-hosted-1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	result := json.RawMessage(`{"modelCallOutcomes":[{"model":"claude-sonnet-4-6","outcome":"attempt_1"}]}`)
	configuration := map[string]any{
		"hostedProvider": models.UsageProviderAnthropic,
		"credentials":    map[string]any{"source": CredentialsSourceHosted},
	}
	RecordRunnerModelCallOutcomes(nil, "task-hosted-1", "runnerSuperPlane.finished", configuration, result, time.Time{})
	RecordRunnerModelCallOutcomes(nil, "task-hosted-1", "runnerSuperPlane.finished", configuration, result, time.Time{})

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
		time.Time{},
	)

	var count int64
	require.NoError(t, db.Model(&models.HostedModelCallOutcome{}).Where("runner_task_id = ?", "task-byok-1").Count(&count).Error)
	assert.Zero(t, count)
}

func TestProcessTaskStatusRecordsHostedModelCallOutcomes(t *testing.T) {
	syncModelCallOutcomeInserts(t)
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

func TestRecordRunnerModelCallOutcomesCapsSamplesPerTask(t *testing.T) {
	syncModelCallOutcomeInserts(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id = ?", "task-cap-1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	rows := make([]map[string]any, maxModelCallOutcomesPerTask+1)
	for i := range rows {
		rows[i] = map[string]any{"model": "gpt-5", "outcome": "attempt_1"}
	}
	result, err := json.Marshal(map[string]any{"modelCallOutcomes": rows})
	require.NoError(t, err)
	RecordRunnerModelCallOutcomes(
		nil,
		"task-cap-1",
		"runnerSuperPlane.finished",
		map[string]any{
			"hostedProvider": models.UsageProviderOpenAI,
			"credentials":    map[string]any{"source": CredentialsSourceHosted},
		},
		result,
		time.Time{},
	)

	var stored []models.HostedModelCallOutcome
	require.NoError(t, db.Where("runner_task_id = ?", "task-cap-1").Order("sequence").Find(&stored).Error)
	require.Len(t, stored, 500)
	assert.Equal(t, 1, stored[0].Sequence)
	assert.Equal(t, 500, stored[len(stored)-1].Sequence)
}

func TestRecordRunnerModelCallOutcomesClampsTimestampsBeforeTaskStart(t *testing.T) {
	syncModelCallOutcomeInserts(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = db.Where("runner_task_id = ?", "task-clamp-1").Delete(&models.HostedModelCallOutcome{}).Error
	})

	started := time.Now().UTC().Truncate(time.Millisecond)
	_, stamp := rfc3339Stamp(t, -time.Hour)
	result, err := json.Marshal(map[string]any{
		"modelCallOutcomes": []map[string]any{
			{"model": "gpt-5", "outcome": "attempt_1", "occurredAt": stamp},
		},
	})
	require.NoError(t, err)
	RecordRunnerModelCallOutcomes(
		nil,
		"task-clamp-1",
		"runnerSuperPlane.finished",
		map[string]any{
			"hostedProvider": models.UsageProviderOpenAI,
			"credentials":    map[string]any{"source": CredentialsSourceHosted},
		},
		result,
		started,
	)

	var rows []models.HostedModelCallOutcome
	require.NoError(t, db.Where("runner_task_id = ?", "task-clamp-1").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].OccurredAt.UTC().Equal(started))
}

func TestProcessTaskStatusIgnoresModelStatusWriteFailures(t *testing.T) {
	syncModelCallOutcomeInserts(t)
	previous := insertHostedModelCallOutcomes
	t.Cleanup(func() {
		insertHostedModelCallOutcomes = previous
		_ = database.DB(context.Background()).
			Where("runner_task_id LIKE ?", "task-model-status-%").
			Delete(&models.HostedModelCallOutcome{}).Error
	})

	t.Run("insert fails", func(t *testing.T) {
		insertHostedModelCallOutcomes = func(tx *gorm.DB, inputs []models.HostedModelCallOutcomeInput) error {
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			err := models.InsertHostedModelCallOutcomes(tx.WithContext(canceled), inputs)
			require.Error(t, err)
			return err
		}
		assertModelStatusLeavesTask(t, "succeeded", 0, PassedOutputChannel, []any{
			map[string]any{"model": "gpt-5", "outcome": "attempt_1"},
		})
	})

	t.Run("insert panics", func(t *testing.T) {
		insertHostedModelCallOutcomes = func(*gorm.DB, []models.HostedModelCallOutcomeInput) error {
			panic("model status insert panic")
		}
		assertModelStatusLeavesTask(t, "failed", 1, FailedOutputChannel, []any{
			map[string]any{"model": "gpt-5", "outcome": "error"},
		})
	})

	t.Run("malformed result", func(t *testing.T) {
		insertHostedModelCallOutcomes = previous
		huge := make([]any, 2000)
		for i := range huge {
			huge[i] = map[string]any{"model": "gpt-5", "outcome": "attempt_1"}
		}
		for _, outcomes := range []any{
			"nope",
			[]any{1, true, "bad"},
			[]any{map[string]any{"model": "gpt-5", "outcome": "attempt_1", "occurredAt": "not-a-time"}},
			huge,
		} {
			assertModelStatusLeavesTask(t, "succeeded", 0, PassedOutputChannel, outcomes)
		}
	})
}

func TestProcessTaskStatusDoesNotWaitForModelStatusInsert(t *testing.T) {
	previousInsert := insertHostedModelCallOutcomes
	blocked := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(blocked) }) }
	t.Cleanup(func() {
		unblock()
		insertHostedModelCallOutcomes = previousInsert
	})
	insertHostedModelCallOutcomes = func(*gorm.DB, []models.HostedModelCallOutcomeInput) error {
		<-blocked
		close(finished)
		return context.Canceled
	}

	task := modelStatusTask(t, "succeeded", 0, []any{
		map[string]any{"model": "gpt-5", "outcome": "attempt_1"},
	})
	state := &contexts.ExecutionStateContext{}
	usage := &countingUsage{}
	done := make(chan error, 1)
	go func() {
		done <- processTaskStatus(state, task, "runnerSuperPlane.finished", "", nil, usage, hostedModelStatusConfiguration())
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("processTaskStatus waited for the model status insert")
	}
	assert.Equal(t, PassedOutputChannel, state.Channel)
	assert.True(t, state.Passed)
	assert.Equal(t, 1, usage.llm)
	assert.Equal(t, 1, usage.compute)
	unblock()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("model status insert did not finish")
	}
}

type countingUsage struct {
	llm     int
	compute int
}

func (u *countingUsage) Record(core.UsageRecord) error {
	u.llm++
	return nil
}

func (u *countingUsage) RecordCompute(core.ComputeUsageRecord) error {
	u.compute++
	return nil
}

func assertModelStatusLeavesTask(t *testing.T, stateName string, exit int, channel string, outcomes any) {
	t.Helper()
	usage := &countingUsage{}
	state := &contexts.ExecutionStateContext{}
	require.NoError(t, processTaskStatus(
		state,
		modelStatusTask(t, stateName, exit, outcomes),
		"runnerSuperPlane.finished",
		"",
		nil,
		usage,
		hostedModelStatusConfiguration(),
	))
	assert.Equal(t, channel, state.Channel)
	assert.True(t, state.Finished)
	assert.NotEmpty(t, state.Payloads)
	assert.Equal(t, 1, usage.llm)
	assert.Equal(t, 1, usage.compute)
}

func modelStatusTask(t *testing.T, stateName string, exit int, outcomes any) *core.RunnerTask {
	t.Helper()
	started := time.Now().Add(-time.Minute)
	finished := time.Now()
	payload, err := json.Marshal(map[string]any{
		"input_tokens":      3,
		"output_tokens":     4,
		"model":             "gpt-5",
		"modelCallOutcomes": outcomes,
	})
	require.NoError(t, err)
	return &core.RunnerTask{
		ID:         "task-model-status-" + stateName,
		State:      stateName,
		ExitCode:   &exit,
		Result:     payload,
		StartedAt:  &started,
		FinishedAt: &finished,
	}
}

func hostedModelStatusConfiguration() map[string]any {
	return map[string]any{
		"hostedProvider": models.UsageProviderOpenAI,
		"credentials":    map[string]any{"source": CredentialsSourceHosted},
	}
}

func rfc3339Stamp(t *testing.T, delta time.Duration) (time.Time, string) {
	t.Helper()
	stamp := time.Now().UTC().Add(delta).Truncate(time.Millisecond)
	return stamp, stamp.Format(time.RFC3339Nano)
}
