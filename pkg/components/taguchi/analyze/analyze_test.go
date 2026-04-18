package analyze

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type memoryStub struct {
	arms   []any
	trials []any
}

func (m *memoryStub) Add(string, any) error { return nil }
func (m *memoryStub) Find(_ string, matches map[string]any) ([]any, error) {
	switch matches["kind"] {
	case MemoryKindArm:
		return m.arms, nil
	case MemoryKindTrial:
		return m.trials, nil
	}
	return nil, nil
}
func (m *memoryStub) FindFirst(string, map[string]any) (any, error) { return nil, nil }

func TestAnalyze_ConfidentWinner(t *testing.T) {
	// Arm a dominates arm b by a wide margin, with low variance → confident.
	mem := &memoryStub{
		arms: []any{
			map[string]any{"kind": MemoryKindArm, "arm_id": "a", "params": map[string]any{"icons": "classic", "board": "7x6"}},
			map[string]any{"kind": MemoryKindArm, "arm_id": "b", "params": map[string]any{"icons": "themed", "board": "9x7"}},
		},
		trials: []any{
			map[string]any{"kind": MemoryKindTrial, "arm_id": "a", "metric": "rematch_rate", "value": 0.9},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "a", "metric": "rematch_rate", "value": 0.88},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "a", "metric": "rematch_rate", "value": 0.92},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "b", "metric": "rematch_rate", "value": 0.5},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "b", "metric": "rematch_rate", "value": 0.48},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "b", "metric": "rematch_rate", "value": 0.52},
		},
	}
	exec := &contexts.ExecutionStateContext{}

	err := (&Analyze{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"experimentId":        "e",
			"metric":              "rematch_rate",
			"direction":           DirectionLargerIsBetter,
			"confidenceThreshold": 1.0,
		},
		Metadata:     &contexts.MetadataContext{},
		CanvasMemory: mem, ExecutionState: exec,
	})

	require.NoError(t, err)
	assert.Equal(t, ChannelNameConfident, exec.Channel)

	payload := exec.Payloads[0].(map[string]any)["data"].(map[string]any)
	winner := payload["winner"].(map[string]any)
	assert.Equal(t, "a", winner["arm_id"])
}

func TestAnalyze_InconclusiveWhenArmsTooClose(t *testing.T) {
	mem := &memoryStub{
		arms: []any{
			map[string]any{"kind": MemoryKindArm, "arm_id": "a", "params": map[string]any{"icons": "classic"}},
			map[string]any{"kind": MemoryKindArm, "arm_id": "b", "params": map[string]any{"icons": "themed"}},
		},
		trials: []any{
			map[string]any{"kind": MemoryKindTrial, "arm_id": "a", "metric": "rematch_rate", "value": 0.5},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "a", "metric": "rematch_rate", "value": 0.6},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "b", "metric": "rematch_rate", "value": 0.52},
			map[string]any{"kind": MemoryKindTrial, "arm_id": "b", "metric": "rematch_rate", "value": 0.55},
		},
	}
	exec := &contexts.ExecutionStateContext{}

	err := (&Analyze{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"experimentId":        "e",
			"metric":              "rematch_rate",
			"direction":           DirectionLargerIsBetter,
			"confidenceThreshold": 5.0,
		},
		Metadata:     &contexts.MetadataContext{},
		CanvasMemory: mem, ExecutionState: exec,
	})

	require.NoError(t, err)
	assert.Equal(t, ChannelNameInconclusive, exec.Channel)
}
