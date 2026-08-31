package runner

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type kvState struct {
	kv map[string]string
}

func (s *kvState) IsFinished() bool { return false }
func (s *kvState) SetKV(key, value string) error {
	if s.kv == nil {
		s.kv = map[string]string{}
	}
	s.kv[key] = value
	return nil
}
func (s *kvState) GetKV(key string) (string, error) {
	if s.kv == nil {
		return "", core.ErrExecutionKVNotFound
	}
	value, ok := s.kv[key]
	if !ok {
		return "", core.ErrExecutionKVNotFound
	}
	return value, nil
}
func (s *kvState) Emit(string, string, []any) error            { return nil }
func (s *kvState) EmitAndContinue(string, string, []any) error { return nil }
func (s *kvState) Pass() error                                 { return nil }
func (s *kvState) Fail(string, string) error                   { return errors.New("unused") }

type recordingComputeUsage struct {
	computes []core.ComputeUsageRecord
}

func (r *recordingComputeUsage) Record(core.UsageRecord) error { return nil }
func (r *recordingComputeUsage) RecordCompute(record core.ComputeUsageRecord) error {
	r.computes = append(r.computes, record)
	return nil
}

func TestRecordRunnerComputeUsageFromFinishedTask(t *testing.T) {
	t.Parallel()

	claimed := time.Now().Add(-3 * time.Second)
	finished := claimed.Add(1500 * time.Millisecond)
	state := &kvState{kv: map[string]string{
		executionKVMachineType: MachineTypeE1LargeAMD64,
		executionKVFleetID:     "local",
	}}
	recorder := &recordingComputeUsage{}
	RecordRunnerComputeUsage(recorder, nil, state, map[string]any{"machine_type": MachineTypeE1TinyAMD64}, &Task{
		ID:         "task-1",
		Status:     "succeeded",
		ClaimedAt:  &claimed,
		FinishedAt: &finished,
	})
	require.Len(t, recorder.computes, 1)
	assert.Equal(t, MachineTypeE1LargeAMD64, recorder.computes[0].MachineType)
	assert.Equal(t, "local", recorder.computes[0].FleetID)
	assert.Equal(t, int64(2), recorder.computes[0].DurationSeconds)
	assert.Equal(t, models.UsageIdempotencyKeyRunner+":compute:task-1", recorder.computes[0].IdempotencyKey)
}

func TestRecordRunnerComputeUsageSkipsMissingTimestamps(t *testing.T) {
	t.Parallel()

	recorder := &recordingComputeUsage{}
	RecordRunnerComputeUsage(recorder, nil, &kvState{}, map[string]any{"machine_type": MachineTypeE1LargeAMD64}, &Task{
		ID:     "task-2",
		Status: "failed",
	})
	assert.Empty(t, recorder.computes)
}

func TestRecordRunnerComputeUsageRecordsZeroDuration(t *testing.T) {
	t.Parallel()

	now := time.Now()
	recorder := &recordingComputeUsage{}
	RecordRunnerComputeUsage(recorder, nil, &kvState{}, map[string]any{"machineType": MachineTypeE1TinyARM64}, &Task{
		TaskID:     "task-3",
		Status:     "canceled",
		ClaimedAt:  &now,
		FinishedAt: &now,
	})
	require.Len(t, recorder.computes, 1)
	assert.Equal(t, MachineTypeE1TinyARM64, recorder.computes[0].MachineType)
	assert.Equal(t, int64(0), recorder.computes[0].DurationSeconds)
}
