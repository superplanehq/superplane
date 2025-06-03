package inputs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__Resolve(t *testing.T) {
	t.Run("no variables to resolve", func(t *testing.T) {
		executorSpec := support.ExecutorSpec()
		specBuilder := NewExecutorSpecBuilder(executorSpec, map[string]any{}, map[string]string{})
		spec, err := specBuilder.Build()
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.Equal(t, models.ExecutorSpecTypeSemaphore, spec.Type)
		assert.Equal(t, "demo-project", spec.Semaphore.ProjectID)
		assert.Equal(t, "demo-task", spec.Semaphore.TaskID)
		assert.Equal(t, ".semaphore/run.yml", spec.Semaphore.PipelineFile)
		assert.Equal(t, "main", spec.Semaphore.Branch)
		assert.Equal(t, map[string]string{
			"PARAM_1": "VALUE_1",
			"PARAM_2": "VALUE_2",
		}, spec.Semaphore.Parameters)
	})

	t.Run("with inputs and secrets to resolve", func(t *testing.T) {
		inputs := map[string]any{
			"BRANCH":     "hello",
			"PROJECT_ID": "hello",
			"PARAM_1":    "value1",
			"PARAM_2":    "value2",
		}

		secrets := map[string]string{
			"API_TOKEN": "XYZ",
		}

		executorSpec := support.ExecutorSpec()
		executorSpec.Semaphore.APIToken = "${{ secrets.API_TOKEN }}"
		executorSpec.Semaphore.Branch = "${{ inputs.BRANCH }}"
		executorSpec.Semaphore.ProjectID = "${{ inputs.PROJECT_ID }}"
		executorSpec.Semaphore.Parameters = map[string]string{
			"PARAM_1": "${{ inputs.PARAM_1 }}",
			"PARAM_2": "${{ inputs.PARAM_2 }}",
		}

		specBuilder := NewExecutorSpecBuilder(executorSpec, inputs, secrets)
		spec, err := specBuilder.Build()
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.Equal(t, models.ExecutorSpecTypeSemaphore, spec.Type)
		assert.Equal(t, "XYZ", spec.Semaphore.APIToken)
		assert.Equal(t, "hello", spec.Semaphore.ProjectID)
		assert.Equal(t, "hello", spec.Semaphore.Branch)
		assert.Equal(t, ".semaphore/run.yml", spec.Semaphore.PipelineFile)
		assert.Equal(t, "demo-task", spec.Semaphore.TaskID)
		assert.Equal(t, map[string]string{
			"PARAM_1": "value1",
			"PARAM_2": "value2",
		}, spec.Semaphore.Parameters)
	})
}
