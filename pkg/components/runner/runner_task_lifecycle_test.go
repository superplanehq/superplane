package runner

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestBillableSeconds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration time.Duration
		expected int64
	}{
		{"sub-second rounds up", 117 * time.Millisecond, 1},
		{"whole second", time.Second, 1},
		{"partial second rounds up", 7739 * time.Millisecond, 8},
		{"exact multiple", 5 * time.Minute, 300},
		{"zero", 0, 0},
		{"sub-second negative from clock skew", -117 * time.Millisecond, 0},
		{"negative", -30 * time.Second, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, billableSeconds(tc.duration))
		})
	}
}

func TestProcessBrokerTaskStatusRetriesWhenPublishFails(t *testing.T) {
	original := publishRunnerTaskFinished
	t.Cleanup(func() { publishRunnerTaskFinished = original })

	publishRunnerTaskFinished = func(organizationID, taskID string, durationSeconds int64) error {
		assert.Equal(t, "org-1", organizationID)
		assert.Equal(t, "task-1", taskID)
		assert.Equal(t, int64(2), durationSeconds)
		return errors.New("rabbitmq down")
	}

	claimed := time.Now().UTC().Add(-2 * time.Second)
	finished := time.Now().UTC()
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	task := &Task{
		ID:         "task-1",
		Status:     "succeeded",
		ExitCode:   intPtr(0),
		ClaimedAt:  &claimed,
		FinishedAt: &finished,
	}

	err := processBrokerTaskStatus(state, task, "finished", "org-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish runner usage")
	assert.False(t, state.IsFinished())
}

func TestProcessBrokerTaskStatusPublishesThenFinishes(t *testing.T) {
	original := publishRunnerTaskFinished
	t.Cleanup(func() { publishRunnerTaskFinished = original })

	var published bool
	publishRunnerTaskFinished = func(organizationID, taskID string, durationSeconds int64) error {
		published = true
		assert.Equal(t, "org-1", organizationID)
		assert.Equal(t, "task-1", taskID)
		assert.Equal(t, int64(1), durationSeconds)
		return nil
	}

	claimed := time.Now().UTC().Add(-500 * time.Millisecond)
	finished := time.Now().UTC()
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	task := &Task{
		ID:         "task-1",
		Status:     "succeeded",
		ExitCode:   intPtr(0),
		ClaimedAt:  &claimed,
		FinishedAt: &finished,
	}

	require.NoError(t, processBrokerTaskStatus(state, task, "finished", "org-1", nil))
	assert.True(t, published)
	assert.True(t, state.IsFinished())
}
