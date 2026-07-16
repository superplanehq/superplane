package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The execution runtime (ExecutionStateContext.Emit) wraps each emitted payload
// as a single object under "data" (event = {type, timestamp, data: <object>}).
// ExampleOutput must mirror that shape so expression autocomplete suggests the
// correct path ($["Node"].data.result...) instead of misleading users into
// array indexing ($["Node"].data[0]...), which resolves to nothing at runtime.
// Regression test for issue #5944.
func TestRunnerExampleOutputDataIsObject(t *testing.T) {
	components := map[string]interface {
		ExampleOutput() map[string]any
	}{
		"RunPython": &RunPython{},
		"RunBash":   &RunBash{},
		"RunJS":     &RunJS{},
		"Runner":    &Runner{},
	}

	for name, component := range components {
		t.Run(name, func(t *testing.T) {
			example := component.ExampleOutput()

			require.Contains(t, example, "type")
			require.Contains(t, example, "timestamp")
			require.Contains(t, example, "data")

			data, ok := example["data"].(map[string]any)
			require.Truef(t, ok, "data must be an object (map), got %T", example["data"])

			assert.Contains(t, data, "status")
			assert.Contains(t, data, "exit_code")
			assert.Contains(t, data, "result")
		})
	}
}
