package noop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/executors"
)

func Test_HTTP__Execute(t *testing.T) {
	executor := NewNoOpExecutor()
	executionID := uuid.New()
	stageID := uuid.New()

	t.Run("generates random outputs", func(t *testing.T) {
		spec, err := json.Marshal(NoOpSpec{})
		require.NoError(t, err)
		response, err := executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
			OutputNames: []string{"abc", "def"},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.True(t, response.Successful())
		outputs := response.Outputs()
		assert.NotEmpty(t, outputs["abc"])
		assert.NotEmpty(t, outputs["def"])
	})
}

func Test_HTTP__Validate(t *testing.T) {
	executor := NewNoOpExecutor()

	t.Run("empty spec is OK", func(t *testing.T) {
		spec := NoOpSpec{}
		data, err := json.Marshal(&spec)
		require.NoError(t, err)
		err = executor.Validate(context.Background(), data)
		require.NoError(t, err)
	})
}
