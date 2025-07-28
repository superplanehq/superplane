package semaphore_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/executors"
	semaphorexec "github.com/superplanehq/superplane/pkg/executors/semaphore"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_Semaphore__Execute(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	executionID := uuid.New()
	stageID := uuid.New()
	projectID := uuid.NewString()

	t.Run("runs workflow if task ID is empty", func(t *testing.T) {
		integration, err := semaphore.NewSemaphoreIntegration(context.Background(), r.Integration, func() (string, error) { return "test", nil })
		require.NoError(t, err)

		executor, err := semaphorexec.NewSemaphoreExecutor(integration, &models.Resource{
			ResourceType: semaphore.ResourceTypeProject,
			ExternalID:   projectID,
		})

		require.NoError(t, err)
		require.NotNil(t, executor)

		spec, err := json.Marshal(&semaphorexec.SemaphoreSpec{
			PipelineFile: ".semaphore/semaphore.yml",
			Branch:       "main",
			Parameters:   map[string]string{"a": "b", "c": "d"},
		})

		require.NoError(t, err)
		_, err = executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
			Token:       "token",
		})

		require.NoError(t, err)

		params := r.SemaphoreAPIMock.LastRunWorkflow
		require.NotNil(t, params)
		assert.Equal(t, "refs/heads/main", params.Reference)
		assert.Equal(t, ".semaphore/semaphore.yml", params.PipelineFile)
		assert.Equal(t, projectID, params.ProjectID)
		assert.Len(t, params.Parameters, 5)
		assert.NotEmpty(t, params.Parameters["SEMAPHORE_STAGE_EXECUTION_TOKEN"])
		assert.Equal(t, stageID.String(), params.Parameters["SEMAPHORE_STAGE_ID"])
		assert.Equal(t, executionID.String(), params.Parameters["SEMAPHORE_STAGE_EXECUTION_ID"])
		assert.Equal(t, "b", params.Parameters["a"])
		assert.Equal(t, "d", params.Parameters["c"])
	})

	t.Run("runs task if task ID is not empty", func(t *testing.T) {
		integration, err := semaphore.NewSemaphoreIntegration(context.Background(), r.Integration, func() (string, error) { return "test", nil })
		require.NoError(t, err)

		executor, err := semaphorexec.NewSemaphoreExecutor(integration, &models.Resource{
			ResourceType: semaphore.ResourceTypeTask,
			ExternalID:   projectID,
		})

		require.NoError(t, err)
		require.NotNil(t, executor)

		task := uuid.NewString()
		spec, err := json.Marshal(&semaphorexec.SemaphoreSpec{
			Task:         task,
			PipelineFile: ".semaphore/semaphore.yml",
			Branch:       "main",
			Parameters:   map[string]string{"a": "b", "c": "d"},
		})

		require.NoError(t, err)
		_, err = executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
			Token:       "token",
		})

		require.NoError(t, err)

		runTaskRequest := r.SemaphoreAPIMock.LastRunTask
		require.NotNil(t, runTaskRequest)
		assert.Equal(t, "main", runTaskRequest.Branch)
		assert.Equal(t, ".semaphore/semaphore.yml", runTaskRequest.PipelineFile)
		assert.Len(t, runTaskRequest.Parameters, 5)
		assert.Len(t, runTaskRequest.Parameters, 5)

		assert.NotEmpty(t, runTaskRequest.Parameters["SEMAPHORE_STAGE_EXECUTION_TOKEN"])
		assert.Equal(t, stageID.String(), runTaskRequest.Parameters["SEMAPHORE_STAGE_ID"])
		assert.Equal(t, executionID.String(), runTaskRequest.Parameters["SEMAPHORE_STAGE_EXECUTION_ID"])
		assert.Equal(t, "b", runTaskRequest.Parameters["a"])
		assert.Equal(t, "d", runTaskRequest.Parameters["c"])
	})
}

func Test_Semaphore__Validate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Run("integration is required", func(t *testing.T) {
		executor, err := semaphorexec.NewSemaphoreExecutor(nil, nil)
		require.NoError(t, err)
		require.NotNil(t, executor)
		require.ErrorContains(t, executor.Validate(context.Background(), []byte(`{}`)), "integration is required")
	})

	t.Run("branch is required", func(t *testing.T) {
		integration, err := r.Registry.NewIntegration(context.Background(), r.Integration)
		require.NoError(t, err)

		_, _, resource := support.Executor(t, r)
		executor, err := semaphorexec.NewSemaphoreExecutor(integration, resource)
		require.NoError(t, err)
		require.NotNil(t, executor)

		spec, err := json.Marshal(&semaphorexec.SemaphoreSpec{
			Branch: "",
		})

		require.NoError(t, err)
		require.ErrorContains(t, executor.Validate(context.Background(), spec), "branch is required")
	})
}
