package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type memoryStub struct {
	added []map[string]any
}

func (m *memoryStub) Add(namespace string, values any) error {
	row, _ := values.(map[string]any)
	row["__ns"] = namespace
	m.added = append(m.added, row)
	return nil
}
func (m *memoryStub) Find(string, map[string]any) ([]any, error) { return nil, nil }
func (m *memoryStub) FindFirst(string, map[string]any) (any, error) {
	return nil, nil
}

func TestPlan_Execute_L9FromThreeThreeLevelFactors(t *testing.T) {
	exec := &contexts.ExecutionStateContext{}
	mem := &memoryStub{}

	err := (&Plan{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"experimentId": "exp-1",
			"factors": []map[string]any{
				{"name": "icons", "levels": []string{"classic", "themed_a", "themed_b"}},
				{"name": "board", "levels": []string{"7x6", "9x7", "11x8"}},
				{"name": "time", "levels": []string{"none", "blitz", "classical"}},
			},
		},
		Metadata:       &contexts.MetadataContext{},
		CanvasMemory:   mem,
		ExecutionState: exec,
	})

	require.NoError(t, err)
	require.Len(t, exec.Payloads, 1)

	payload := exec.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "L9", payload["arrayName"])
	arms := payload["arms"].([]map[string]any)
	assert.Len(t, arms, 9, "L9 yields 9 arms")

	// Arms must have distinct param combinations
	seen := map[string]struct{}{}
	for _, arm := range arms {
		params := arm["params"].(map[string]any)
		key := params["icons"].(string) + "|" + params["board"].(string) + "|" + params["time"].(string)
		_, dup := seen[key]
		assert.False(t, dup, "duplicate arm param combination: %s", key)
		seen[key] = struct{}{}
	}

	assert.Len(t, mem.added, 9, "one memory row per arm")
	for _, row := range mem.added {
		assert.Equal(t, "taguchi:exp-1", row["__ns"])
		assert.Equal(t, MemoryKindArm, row["kind"])
	}
}

func TestPlan_Execute_PicksL4ForTwoLevelFactors(t *testing.T) {
	exec := &contexts.ExecutionStateContext{}

	err := (&Plan{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"experimentId": "exp-2",
			"factors": []map[string]any{
				{"name": "a", "levels": []string{"on", "off"}},
				{"name": "b", "levels": []string{"on", "off"}},
				{"name": "c", "levels": []string{"on", "off"}},
			},
		},
		Metadata:       &contexts.MetadataContext{},
		CanvasMemory:   &memoryStub{},
		ExecutionState: exec,
	})

	require.NoError(t, err)
	payload := exec.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "L4", payload["arrayName"])
	assert.Len(t, payload["arms"].([]map[string]any), 4)
}

func TestPlan_Execute_RejectsEmptyExperimentID(t *testing.T) {
	err := (&Plan{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"experimentId": "  ",
			"factors": []map[string]any{
				{"name": "a", "levels": []string{"x", "y"}},
			},
		},
	})
	assert.ErrorContains(t, err, "experimentId is required")
}
