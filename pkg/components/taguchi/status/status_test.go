package status

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

func arm(id string) any {
	return map[string]any{"kind": MemoryKindArm, "arm_id": id}
}
func trial(id string) any {
	return map[string]any{"kind": MemoryKindTrial, "arm_id": id}
}

func TestStatus_SampleSizeMet(t *testing.T) {
	mem := &memoryStub{
		arms: []any{arm("a"), arm("b")},
		trials: []any{
			trial("a"), trial("a"), trial("a"),
			trial("b"), trial("b"), trial("b"),
		},
	}
	exec := &contexts.ExecutionStateContext{}

	err := (&Status{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{"experimentId": "e", "minPerArm": 3},
		Metadata:      &contexts.MetadataContext{},
		CanvasMemory:  mem, ExecutionState: exec,
	})

	require.NoError(t, err)
	assert.Equal(t, ChannelNameSampleMet, exec.Channel)
}

func TestStatus_PendingWhenOneArmBelow(t *testing.T) {
	mem := &memoryStub{
		arms:   []any{arm("a"), arm("b")},
		trials: []any{trial("a"), trial("a"), trial("a"), trial("b")},
	}
	exec := &contexts.ExecutionStateContext{}

	err := (&Status{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{"experimentId": "e", "minPerArm": 3},
		Metadata:      &contexts.MetadataContext{},
		CanvasMemory:  mem, ExecutionState: exec,
	})

	require.NoError(t, err)
	assert.Equal(t, ChannelNamePending, exec.Channel)
}
