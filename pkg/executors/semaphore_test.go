package executors

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	semaphoremock "github.com/superplanehq/superplane/test/semaphore"
)

func Test_Semaphore(t *testing.T) {
	executionID := uuid.New()
	stageID := uuid.New()
	projectID := uuid.NewString()

	t.Run("runs workflow if task ID is empty", func(t *testing.T) {
		semaphoreMock := semaphoremock.NewSemaphoreAPIMock()
		semaphoreMock.Init()
		defer semaphoreMock.Close()

		integration, err := integrations.NewSemaphoreIntegration(semaphoreMock.Server.URL, "test")
		require.NoError(t, err)

		executor, err := NewSemaphoreExecutor(integration, &models.Resource{
			ResourceType: integrations.ResourceTypeProject,
			ExternalID:   projectID,
		})

		require.NoError(t, err)
		require.NotNil(t, executor)

		_, err = executor.Execute(models.ExecutorSpec{
			Semaphore: &models.SemaphoreExecutorSpec{
				PipelineFile: ".semaphore/semaphore.yml",
				Branch:       "main",
				Parameters:   map[string]string{"a": "b", "c": "d"},
			},
		}, ExecutionParameters{StageID: stageID.String(), ExecutionID: executionID.String()})

		require.NoError(t, err)

		params := semaphoreMock.LastRunWorkflow
		require.NotNil(t, params)
		assert.Equal(t, "refs/heads/main", params.Reference)
		assert.Equal(t, ".semaphore/semaphore.yml", params.PipelineFile)
		assert.Equal(t, projectID, params.ProjectID)
		assert.Len(t, params.Parameters, 4)
		assert.Equal(t, stageID.String(), params.Parameters["SEMAPHORE_STAGE_ID"])
		assert.Equal(t, executionID.String(), params.Parameters["SEMAPHORE_STAGE_EXECUTION_ID"])
		assert.Equal(t, "b", params.Parameters["a"])
		assert.Equal(t, "d", params.Parameters["c"])
	})

	t.Run("runs task if task ID is not empty", func(t *testing.T) {
		semaphoreMock := semaphoremock.NewSemaphoreAPIMock()
		semaphoreMock.Init()
		defer semaphoreMock.Close()

		integration, err := integrations.NewSemaphoreIntegration(semaphoreMock.Server.URL, "test")
		require.NoError(t, err)

		executor, err := NewSemaphoreExecutor(integration, &models.Resource{
			ResourceType: integrations.ResourceTypeProject,
			ExternalID:   projectID,
		})
		require.NoError(t, err)
		require.NotNil(t, executor)

		taskID := uuid.NewString()
		_, err = executor.Execute(models.ExecutorSpec{
			Semaphore: &models.SemaphoreExecutorSpec{
				TaskId:       &taskID,
				PipelineFile: ".semaphore/semaphore.yml",
				Branch:       "main",
				Parameters:   map[string]string{"a": "b", "c": "d"},
			},
		}, ExecutionParameters{StageID: stageID.String(), ExecutionID: executionID.String()})

		require.NoError(t, err)

		runTaskRequest := semaphoreMock.LastRunTask
		require.NotNil(t, runTaskRequest)
		assert.Equal(t, "main", runTaskRequest.Branch)
		assert.Equal(t, ".semaphore/semaphore.yml", runTaskRequest.PipelineFile)
		assert.Len(t, runTaskRequest.Parameters, 4)
		assert.Equal(t, stageID.String(), runTaskRequest.Parameters["SEMAPHORE_STAGE_ID"])
		assert.Equal(t, executionID.String(), runTaskRequest.Parameters["SEMAPHORE_STAGE_EXECUTION_ID"])
		assert.Equal(t, "b", runTaskRequest.Parameters["a"])
		assert.Equal(t, "d", runTaskRequest.Parameters["c"])
	})
}
