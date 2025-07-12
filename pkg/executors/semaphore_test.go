package executors

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	semaphoremock "github.com/superplanehq/superplane/test/semaphore"
)

func Test_Semaphore(t *testing.T) {
	signer := jwt.NewSigner("test")
	executionID := uuid.New()
	stageID := uuid.New()
	projectID := uuid.NewString()
	execution := models.StageExecution{
		ID:      executionID,
		StageID: stageID,
	}

	t.Run("runs workflow if task ID is empty", func(t *testing.T) {
		semaphoreMock := semaphoremock.NewSemaphoreAPIMock()
		semaphoreMock.Init()
		defer semaphoreMock.Close()

		integration, err := integrations.NewSemaphoreIntegration(semaphoreMock.Server.URL, "test")
		require.NoError(t, err)

		executor, err := NewSemaphoreExecutor(integration, &execution, signer)
		require.NoError(t, err)
		require.NotNil(t, executor)

		_, err = executor.Execute(models.ExecutorSpec{
			Semaphore: &models.SemaphoreExecutorSpec{
				ProjectID:    projectID,
				PipelineFile: ".semaphore/semaphore.yml",
				Branch:       "main",
				Parameters:   map[string]string{"a": "b", "c": "d"},
			},
		})

		require.NoError(t, err)

		params := semaphoreMock.LastRunWorkflow
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
		semaphoreMock := semaphoremock.NewSemaphoreAPIMock()
		semaphoreMock.Init()
		defer semaphoreMock.Close()

		integration, err := integrations.NewSemaphoreIntegration(semaphoreMock.Server.URL, "test")
		require.NoError(t, err)

		executor, err := NewSemaphoreExecutor(integration, &execution, signer)
		require.NoError(t, err)
		require.NotNil(t, executor)

		taskID := uuid.NewString()
		_, err = executor.Execute(models.ExecutorSpec{
			Semaphore: &models.SemaphoreExecutorSpec{
				ProjectID:    projectID,
				PipelineFile: ".semaphore/semaphore.yml",
				Branch:       "main",
				Parameters:   map[string]string{"a": "b", "c": "d"},
				TaskID:       taskID,
			},
		})

		require.NoError(t, err)

		taskTrigger := semaphoreMock.LastTaskTrigger
		require.NotNil(t, taskTrigger)
		assert.Equal(t, "main", taskTrigger.Spec.Branch)
		assert.Equal(t, ".semaphore/semaphore.yml", taskTrigger.Spec.PipelineFile)
		assert.Len(t, taskTrigger.Spec.Parameters, 5)
		assert.Len(t, taskTrigger.Spec.Parameters, 5)

		assert.True(t, slices.ContainsFunc(taskTrigger.Spec.Parameters, func(p integrations.TaskTriggerParameter) bool {
			return p.Name == "SEMAPHORE_STAGE_EXECUTION_TOKEN" && p.Value != ""
		}))

		assert.True(t, slices.ContainsFunc(taskTrigger.Spec.Parameters, func(p integrations.TaskTriggerParameter) bool {
			return p.Name == "SEMAPHORE_STAGE_ID" && p.Value == stageID.String()
		}))

		assert.True(t, slices.ContainsFunc(taskTrigger.Spec.Parameters, func(p integrations.TaskTriggerParameter) bool {
			return p.Name == "SEMAPHORE_STAGE_EXECUTION_ID" && p.Value == execution.ID.String()
		}))

		assert.True(t, slices.ContainsFunc(taskTrigger.Spec.Parameters, func(p integrations.TaskTriggerParameter) bool {
			return p.Name == "a" && p.Value == "b"
		}))

		assert.True(t, slices.ContainsFunc(taskTrigger.Spec.Parameters, func(p integrations.TaskTriggerParameter) bool {
			return p.Name == "c" && p.Value == "d"
		}))
	})
}
